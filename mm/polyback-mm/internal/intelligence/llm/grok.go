package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func callGrokResponses(ctx context.Context, hc *http.Client, model, systemPrompt, userPrompt string, jsonObject bool, tools []string, temperature *float64) (map[string]any, error) {
	key := strings.TrimSpace(os.Getenv("XAI_API_KEY"))
	if key == "" {
		return nil, fmt.Errorf("XAI_API_KEY is not set")
	}
	if hc == nil {
		hc = http.DefaultClient
	}
	payload := map[string]any{
		"model": model,
		"input": []map[string]any{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
	}
	if jsonObject {
		payload["response_format"] = map[string]any{"type": "json_object"}
	}
	if temperature != nil {
		payload["temperature"] = *temperature
	}
	if len(tools) > 0 {
		tl := make([]map[string]string, 0, len(tools))
		for _, t := range tools {
			tl = append(tl, map[string]string{"type": t})
		}
		payload["tools"] = tl
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.x.ai/v1/responses", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("grok: %d %s", resp.StatusCode, string(b))
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}
