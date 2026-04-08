package x402svc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Service implements x402-seller edge function behavior (list, call, health, networks).
type Service struct {
	http *http.Client
}

func NewService(hc *http.Client) *Service {
	if hc == nil {
		hc = &http.Client{Timeout: 120 * time.Second}
	}
	return &Service{http: hc}
}

func discoveryURL() string {
	return strings.TrimSpace(os.Getenv("X402_DISCOVERY_URL"))
}

func evmKey() string { return strings.TrimSpace(os.Getenv("X402_EVM_PRIVATE_KEY")) }
func solKey() string { return strings.TrimSpace(os.Getenv("X402_SOLANA_PRIVATE_KEY")) }

// CheckHealth mirrors checkX402Health.
func (s *Service) CheckHealth() bool {
	u := discoveryURL()
	if u == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u+"?limit=1", nil)
	if err != nil {
		return false
	}
	req.Header.Set("Accept", "application/json")
	resp, err := s.http.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// ListSellers mirrors listBazaarSellers transform.
func (s *Service) ListSellers(network, typ string, limit, offset int) ([]map[string]any, error) {
	base := discoveryURL()
	if base == "" {
		return nil, fmt.Errorf("X402_DISCOVERY_URL environment variable is not set")
	}
	u, err := url.Parse(base)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	if typ != "" {
		q.Set("type", typ)
	}
	if limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", limit))
	}
	if offset > 0 {
		q.Set("offset", fmt.Sprintf("%d", offset))
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("x402 discovery: status %d body=%s", resp.StatusCode, string(b))
	}
	var data any
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, err
	}
	raw := extractSellerSlice(data)
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, transformSeller(m))
	}
	if network != "" {
		filtered := out[:0]
		for _, sel := range out {
			nets, _ := sel["networks"].([]string)
			for _, n := range nets {
				if n == network {
					filtered = append(filtered, sel)
					break
				}
			}
		}
		out = filtered
	}
	return out, nil
}

func extractSellerSlice(data any) []any {
	switch v := data.(type) {
	case []any:
		return v
	case map[string]any:
		if a, ok := v["items"].([]any); ok {
			return a
		}
		if a, ok := v["resources"].([]any); ok {
			return a
		}
	}
	return nil
}

func transformSeller(seller map[string]any) map[string]any {
	resource, _ := seller["resource"].(string)
	accepts, _ := seller["accepts"].([]any)
	lastUpdated, _ := seller["lastUpdated"].(string)
	meta, _ := seller["metadata"].(map[string]any)

	networkSet := map[string]struct{}{}
	lowest := "Unknown"
	for _, a := range accepts {
		pm, ok := a.(map[string]any)
		if !ok {
			continue
		}
		if n, ok := pm["network"].(string); ok && n != "" {
			networkSet[n] = struct{}{}
		}
		amt, _ := pm["amount"].(string)
		if amt == "" {
			amt, _ = pm["maxAmountRequired"].(string)
		}
		if amt != "" {
			p := parseUsdcPrice(amt)
			if p != "Unknown" && (lowest == "Unknown" || comparePrice(p, lowest)) {
				lowest = p
			}
		}
	}
	nets := make([]string, 0, len(networkSet))
	for n := range networkSet {
		nets = append(nets, n)
	}

	name := extractSellerName(resource, meta)
	desc := extractDescription(accepts, meta)

	return map[string]any{
		"id":               resource,
		"name":             name,
		"description":      desc,
		"resourceUrl":      resource,
		"priceUsdc":        lowest,
		"networks":         nets,
		"lastUpdated":      lastUpdated,
		"inputDescription": nil,
	}
}

func extractSellerName(resourceURL string, metadata map[string]any) string {
	if metadata != nil {
		if n, ok := metadata["name"].(string); ok && n != "" {
			return n
		}
		if n, ok := metadata["title"].(string); ok && n != "" {
			return n
		}
	}
	u, err := url.Parse(resourceURL)
	if err != nil {
		if len(resourceURL) > 30 {
			return resourceURL[:30] + "..."
		}
		return resourceURL
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) > 0 {
		return titleCase(strings.ReplaceAll(parts[len(parts)-1], "-", " "))
	}
	return u.Hostname()
}

func titleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) == 0 {
			continue
		}
		words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
	}
	return strings.Join(words, " ")
}

func extractDescription(accepts []any, metadata map[string]any) string {
	if metadata != nil {
		if d, ok := metadata["description"].(string); ok {
			return d
		}
	}
	for _, a := range accepts {
		pm, ok := a.(map[string]any)
		if !ok {
			continue
		}
		if d, ok := pm["description"].(string); ok && d != "" {
			return d
		}
	}
	return ""
}

func parseUsdcPrice(atomic string) string {
	x := new(big.Int)
	if _, ok := x.SetString(atomic, 10); !ok {
		return "Unknown"
	}
	f := new(big.Rat).SetFrac(x, big.NewInt(1_000_000))
	v, _ := f.Float64()
	return fmt.Sprintf("$%.4f", v)
}

func comparePrice(a, b string) bool {
	// crude: strip $ and compare float
	var fa, fb float64
	_, _ = fmt.Sscanf(a, "$%f", &fa)
	_, _ = fmt.Sscanf(b, "$%f", &fb)
	return fa < fb
}

func isSolanaNetwork(network string) bool {
	n := strings.ToLower(network)
	return n == "solana" || strings.HasPrefix(n, "solana:")
}

func isEvmNetwork(network string) bool {
	n := strings.ToLower(network)
	return n == "base" || strings.HasPrefix(n, "eip155:")
}

// CallSeller mirrors callX402Seller (EVM only; Solana returns error).
func (s *Service) CallSeller(resourceURL, query, preferredNetwork string) (success bool, data any, errMsg string, payment map[string]any) {
	evmK := evmKey()
	solK := solKey()

	u, err := url.Parse(resourceURL)
	if err != nil {
		return false, nil, fmt.Sprintf("invalid resourceUrl: %v", err), nil
	}
	if query != "" {
		var obj map[string]any
		if json.Unmarshal([]byte(query), &obj) == nil {
			for k, v := range obj {
				u.Query().Set(k, fmt.Sprint(v))
			}
		} else {
			u.Query().Set("q", query)
		}
	}
	finalURL := u.String()

	req, err := http.NewRequest(http.MethodGet, finalURL, nil)
	if err != nil {
		return false, nil, err.Error(), nil
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.http.Do(req)
	if err != nil {
		return false, nil, err.Error(), nil
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusPaymentRequired {
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			var parsed any
			_ = json.Unmarshal(body, &parsed)
			return true, parsed, "", map[string]any{"network": "free"}
		}
		return false, nil, fmt.Sprintf("unexpected status %d: %s", resp.StatusCode, truncate(string(body), 200)), nil
	}

	var payReq map[string]any
	if err := json.Unmarshal(body, &payReq); err != nil {
		return false, nil, "failed to parse 402 payment json", nil
	}
	x402Ver := 2
	if v, ok := payReq["x402Version"].(float64); ok {
		x402Ver = int(v)
	}
	accepts, _ := payReq["accepts"].([]any)
	resourceInfo, _ := payReq["resource"].(map[string]any)

	var option map[string]any
	var useNet string // "solana" | "evm"

	if preferredNetwork != "" {
		for _, a := range accepts {
			pm, ok := a.(map[string]any)
			if !ok {
				continue
			}
			n, _ := pm["network"].(string)
			if n == preferredNetwork {
				option = pm
				if isSolanaNetwork(n) {
					useNet = "solana"
				} else if isEvmNetwork(n) {
					useNet = "evm"
				}
				break
			}
		}
	}
	if option == nil && solK != "" {
		for _, a := range accepts {
			pm, ok := a.(map[string]any)
			if !ok {
				continue
			}
			n, _ := pm["network"].(string)
			if isSolanaNetwork(n) {
				option = pm
				useNet = "solana"
				break
			}
		}
	}
	if option == nil && evmK != "" {
		for _, a := range accepts {
			pm, ok := a.(map[string]any)
			if !ok {
				continue
			}
			n, _ := pm["network"].(string)
			if isEvmNetwork(n) {
				option = pm
				useNet = "evm"
				break
			}
		}
	}
	if option == nil || useNet == "" {
		return false, nil, "no compatible payment option or missing private keys", nil
	}

	if useNet == "solana" {
		return false, nil, "solana x402 payment is not implemented in polyback intelligence; use an EVM network or preferredNetwork eip155:8453", nil
	}

	if evmK == "" {
		return false, nil, "X402_EVM_PRIVATE_KEY not set", nil
	}

	asset, _ := option["asset"].(string)
	payTo, _ := option["payTo"].(string)
	amt, _ := option["amount"].(string)
	if amt == "" {
		amt, _ = option["maxAmountRequired"].(string)
	}
	netw, _ := option["network"].(string)
	scheme, _ := option["scheme"].(string)
	maxTO := 60
	if v, ok := option["maxTimeoutSeconds"].(float64); ok {
		maxTO = int(v)
	}
	var extra map[string]any
	if e, ok := option["extra"].(map[string]any); ok {
		extra = e
	}

	resURL := finalURL
	resDesc := ""
	mime := "application/json"
	if resourceInfo != nil {
		if uu, ok := resourceInfo["url"].(string); ok && uu != "" {
			resURL = uu
		}
		if d, ok := resourceInfo["description"].(string); ok {
			resDesc = d
		}
		if m, ok := resourceInfo["mimeType"].(string); ok && m != "" {
			mime = m
		}
	}

	headerB64, err := BuildEVMPaymentHeaderBase64(evmK, payTo, asset, amt, netw, x402Ver, scheme, maxTO, extra, resURL, resDesc, mime)
	if err != nil {
		return false, nil, err.Error(), nil
	}

	req2, err := http.NewRequest(http.MethodGet, finalURL, nil)
	if err != nil {
		return false, nil, err.Error(), nil
	}
	req2.Header.Set("Accept", "application/json")
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("PAYMENT-SIGNATURE", headerB64)
	req2.Header.Set("X-Payment", headerB64)

	resp2, err := s.http.Do(req2)
	if err != nil {
		return false, nil, err.Error(), nil
	}
	defer resp2.Body.Close()
	body2, _ := io.ReadAll(resp2.Body)

	cost := parseUsdcPrice(amt)
	pinfo := map[string]any{"cost": cost, "network": netw}
	if resp2.StatusCode < 200 || resp2.StatusCode >= 300 {
		return false, nil, fmt.Sprintf("payment failed: %d %s", resp2.StatusCode, truncate(string(body2), 200)), pinfo
	}

	var parsed any
	if json.Unmarshal(body2, &parsed) != nil {
		parsed = map[string]any{"rawText": string(body2)}
	}
	txID := resp2.Header.Get("X-Payment-Receipt")
	if txID != "" {
		pinfo["txId"] = txID
	}
	return true, parsed, "", pinfo
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// HandlePOST dispatches { action: ... } bodies like the edge function.
func (s *Service) HandlePOST(body map[string]any) (status int, out map[string]any) {
	start := time.Now()
	meta := func(extra map[string]any) map[string]any {
		m := map[string]any{
			"requestId":          fmt.Sprintf("%d", time.Now().UnixNano()),
			"timestamp":          time.Now().UTC().Format(time.RFC3339Nano),
			"processingTimeMs":   time.Since(start).Milliseconds(),
		}
		for k, v := range extra {
			m[k] = v
		}
		return m
	}

	action, _ := body["action"].(string)
	switch action {
	case "health":
		return 200, map[string]any{
			"success":  true,
			"healthy":  s.CheckHealth(),
			"config":   map[string]any{"discoveryUrl": discoveryURL(), "preferredNetwork": os.Getenv("X402_PREFERRED_NETWORK")},
			"metadata": meta(nil),
		}
	case "list":
		network, _ := body["network"].(string)
		typ, _ := body["type"].(string)
		limit := intFromAny(body["limit"])
		offset := intFromAny(body["offset"])
		sellers, err := s.ListSellers(network, typ, limit, offset)
		if err != nil {
			return 500, map[string]any{"success": false, "error": err.Error(), "metadata": meta(map[string]any{"total": 0})}
		}
		return 200, map[string]any{"success": true, "sellers": sellers, "metadata": meta(map[string]any{"total": len(sellers)})}
	case "call":
		resourceURL, _ := body["resourceUrl"].(string)
		q, _ := body["query"].(string)
		netw, _ := body["network"].(string)
		if netw == "" {
			netw = "solana:5eykt4UsFv8P8NJdTREpY1vzqKqZKvdp"
		}
		if resourceURL == "" {
			return 400, map[string]any{"success": false, "error": "Missing required parameter: 'resourceUrl'"}
		}
		if q == "" {
			return 400, map[string]any{"success": false, "error": "Missing required parameter: 'query'"}
		}
		ok, data, errStr, pay := s.CallSeller(resourceURL, q, netw)
		st := 200
		if !ok {
			st = 402
		}
		resp := map[string]any{
			"success": ok,
			"metadata": meta(map[string]any{
				"network":       netw,
				"paymentTxId":   nil,
				"costUsdc":      nil,
			}),
		}
		if pay != nil {
			if v, ok := pay["txId"].(string); ok {
				resp["metadata"].(map[string]any)["paymentTxId"] = v
			}
			if v, ok := pay["cost"].(string); ok {
				resp["metadata"].(map[string]any)["costUsdc"] = v
			}
		}
		if ok {
			resp["data"] = data
		} else {
			resp["error"] = errStr
		}
		return st, resp
	case "networks":
		return 200, map[string]any{
			"success": true,
			"networks": []map[string]any{
				{"id": "solana:5eykt4UsFv8P8NJdTREpY1vzqKqZKvdp", "name": "Solana Mainnet", "type": "solana"},
				{"id": "solana", "name": "Solana Mainnet (legacy)", "type": "solana"},
				{"id": "eip155:8453", "name": "Base Mainnet", "type": "evm"},
				{"id": "base", "name": "Base Mainnet (legacy)", "type": "evm"},
			},
			"metadata": meta(nil),
		}
	default:
		return 400, map[string]any{"success": false, "error": fmt.Sprintf("Unknown action: %q", action)}
	}
}

func intFromAny(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case json.Number:
		i, _ := t.Int64()
		return int(i)
	default:
		return 0
	}
}

// DecodePaymentHeader decodes base64 payment-required (for BlockRun-style 402).
func DecodePaymentHeader(b64 string) (accepts []map[string]any, x402Version int, err error) {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, 0, err
	}
	var wrap map[string]any
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return nil, 0, err
	}
	if v, ok := wrap["x402Version"].(float64); ok {
		x402Version = int(v)
	}
	a, ok := wrap["accepts"].([]any)
	if !ok {
		return []map[string]any{wrap}, x402Version, nil
	}
	out := make([]map[string]any, 0, len(a))
	for _, x := range a {
		if m, ok := x.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out, x402Version, nil
}
