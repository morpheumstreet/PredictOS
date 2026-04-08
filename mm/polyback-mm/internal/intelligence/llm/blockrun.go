package llm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"strings"

	x402svc "github.com/profitlock/PredictOS/mm/polyback-mm/internal/intelligence/adapters/x402svc"
)

const blockrunChatURL = "https://blockrun.ai/api/v1/chat/completions"

var blockrunAliases = map[string]string{
	"blockrun/gpt-4o":            "openai/gpt-4o",
	"blockrun/gpt-4o-mini":       "openai/gpt-4o-mini",
	"blockrun/gpt-4.1":           "openai/gpt-4.1",
	"blockrun/gpt-5":             "openai/gpt-5",
	"blockrun/claude-sonnet-4":   "anthropic/claude-sonnet-4",
	"blockrun/claude-opus-4":     "anthropic/claude-opus-4",
	"blockrun/grok-3":            "xai/grok-3",
	"blockrun/grok-3-fast":       "xai/grok-3-fast",
	"blockrun/gemini-2.5-pro":    "google/gemini-2.5-pro-preview-06-05",
	"blockrun/gemini-2.5-flash":  "google/gemini-2.5-flash-preview-05-20",
	"blockrun/deepseek-chat":     "deepseek/deepseek-chat",
	"blockrun/deepseek-reasoner": "deepseek/deepseek-reasoner",
}

func resolveBlockRunModel(model string) string {
	if m, ok := blockrunAliases[strings.TrimSpace(model)]; ok {
		return m
	}
	if strings.Contains(model, "/") {
		return model
	}
	return model
}

func isBlockRunModel(model string) bool {
	m := strings.TrimSpace(model)
	return strings.HasPrefix(m, "blockrun/") || blockrunAliases[m] != ""
}

func callBlockRun(ctx context.Context, hc *http.Client, model, systemPrompt, userPrompt string, jsonMode, enableSearch bool) (text string, outModel string, totalTokens *int, paymentCost *string, err error) {
	walletKey := strings.TrimSpace(os.Getenv("BLOCKRUN_WALLET_KEY"))
	if walletKey == "" {
		return "", "", nil, nil, fmt.Errorf("BLOCKRUN_WALLET_KEY is not set")
	}
	if hc == nil {
		hc = http.DefaultClient
	}
	actual := resolveBlockRunModel(model)
	payload := map[string]any{
		"model": actual,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
	}
	if jsonMode {
		payload["response_format"] = map[string]string{"type": "json_object"}
	}
	if enableSearch && strings.HasPrefix(actual, "xai/") {
		payload["search"] = true
	}
	body, _ := json.Marshal(payload)

	doReq := func(extra map[string]string) (*http.Response, []byte, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, blockrunChatURL, bytes.NewReader(body))
		if err != nil {
			return nil, nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		for k, v := range extra {
			req.Header.Set(k, v)
		}
		resp, err := hc.Do(req)
		if err != nil {
			return nil, nil, err
		}
		b, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		return resp, b, err
	}

	resp, b, err := doReq(nil)
	if err != nil {
		return "", "", nil, nil, err
	}

	if resp.StatusCode != http.StatusPaymentRequired {
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return "", "", nil, nil, fmt.Errorf("blockrun: %d %s", resp.StatusCode, string(b))
		}
		t, m, tok, err := parseChatCompletion(b)
		if err != nil {
			return "", "", nil, nil, err
		}
		return t, m, tok, nil, nil
	}

	hdr := resp.Header.Get("payment-required")
	if hdr == "" {
		return "", "", nil, nil, fmt.Errorf("blockrun 402: missing payment-required header")
	}
	raw, err := base64.StdEncoding.DecodeString(hdr)
	if err != nil {
		return "", "", nil, nil, fmt.Errorf("blockrun: decode payment header: %w", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", "", nil, nil, err
	}
	x402v := 2
	if v, ok := parsed["x402Version"].(float64); ok {
		x402v = int(v)
	}
	accepts, _ := parsed["accepts"].([]any)
	var acceptList []map[string]any
	for _, a := range accepts {
		if m, ok := a.(map[string]any); ok {
			acceptList = append(acceptList, m)
		}
	}
	if len(acceptList) == 0 {
		return "", "", nil, nil, fmt.Errorf("blockrun: no payment accepts")
	}
	var pay map[string]any
	for _, opt := range acceptList {
		n, _ := opt["network"].(string)
		if n == "eip155:8453" || strings.EqualFold(n, "base") {
			pay = opt
			break
		}
	}
	if pay == nil {
		return "", "", nil, nil, fmt.Errorf("blockrun: no Base payment option")
	}
	amt, _ := pay["amount"].(string)
	if amt == "" {
		amt, _ = pay["maxAmountRequired"].(string)
	}
	asset, _ := pay["asset"].(string)
	payTo, _ := pay["payTo"].(string)
	netw, _ := pay["network"].(string)
	scheme, _ := pay["scheme"].(string)
	maxTO := 300
	if v, ok := pay["maxTimeoutSeconds"].(float64); ok {
		maxTO = int(v)
	}
	var extra map[string]any
	if e, ok := pay["extra"].(map[string]any); ok {
		extra = e
	}
	pc := formatUsdc(amt)

	headerB64, err := x402svc.BuildEVMPaymentHeaderBase64(walletKey, payTo, asset, amt, netw, x402v, scheme, maxTO, extra, blockrunChatURL, "BlockRun AI Chat Completion", "application/json")
	if err != nil {
		return "", "", nil, nil, err
	}

	resp2, b2, err := doReq(map[string]string{"PAYMENT-SIGNATURE": headerB64})
	if err != nil {
		return "", "", nil, nil, err
	}
	if resp2.StatusCode < 200 || resp2.StatusCode >= 300 {
		return "", "", nil, &pc, fmt.Errorf("blockrun paid: %d %s", resp2.StatusCode, string(b2))
	}
	t, m, tok, err := parseChatCompletion(b2)
	return t, m, tok, &pc, err
}

func formatUsdc(atomic string) string {
	x := new(big.Int)
	if _, ok := x.SetString(atomic, 10); !ok {
		return "Unknown"
	}
	f := new(big.Rat).SetFrac(x, big.NewInt(1_000_000))
	v, _ := f.Float64()
	return fmt.Sprintf("$%.6f", v)
}

func parseChatCompletion(b []byte) (text string, model string, totalTokens *int, err error) {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return "", "", nil, err
	}
	if m, ok := raw["model"].(string); ok {
		model = m
	}
	ch, _ := raw["choices"].([]any)
	if len(ch) > 0 {
		if c0, ok := ch[0].(map[string]any); ok {
			if msg, ok := c0["message"].(map[string]any); ok {
				text, _ = msg["content"].(string)
			}
		}
	}
	if u, ok := raw["usage"].(map[string]any); ok {
		if v, ok := u["total_tokens"].(float64); ok {
			t := int(v)
			totalTokens = &t
		}
	}
	return text, model, totalTokens, nil
}
