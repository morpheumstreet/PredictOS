package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/config"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/platforms/polymarket/gamma"
	"github.com/shopspring/decimal"
)

// Trading proxies Polymarket order flows to the polyback executor and public APIs.
type Trading struct {
	root *config.Root
	hc   *http.Client
}

func NewTrading(root *config.Root, hc *http.Client) *Trading {
	if hc == nil {
		hc = &http.Client{Timeout: 120 * time.Second}
	}
	return &Trading{root: root, hc: hc}
}

func (t *Trading) executorBase() string {
	b := strings.TrimSpace(t.root.Hft.Executor.BaseURL)
	if b == "" {
		b = "http://127.0.0.1:8080"
	}
	return strings.TrimSuffix(b, "/")
}

func (t *Trading) postExecutor(path string, body any) (int, []byte, error) {
	raw, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, t.executorBase()+path, bytes.NewReader(raw))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := t.httpClient().Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	return resp.StatusCode, b, err
}

func (t *Trading) httpClient() *http.Client {
	if t.hc == nil {
		return http.DefaultClient
	}
	return t.hc
}

// PolymarketPutOrder handles mapper and legacy-shaped bodies like the edge function.
func (t *Trading) PolymarketPutOrder(ctx context.Context, body []byte) (int, map[string]any) {
	var wrap map[string]any
	if err := json.Unmarshal(body, &wrap); err != nil {
		return 400, map[string]any{"success": false, "error": "Invalid JSON"}
	}
	if op, ok := wrap["orderParams"].(map[string]any); ok {
		return t.placeFromMapperParams(op)
	}
	legacy, err := t.legacyToLimitOrder(wrap)
	if err != nil {
		return 400, map[string]any{"success": false, "error": err.Error()}
	}
	return t.postLimitAndShape(legacy)
}

func (t *Trading) placeFromMapperParams(op map[string]any) (int, map[string]any) {
	tok, _ := op["tokenId"].(string)
	if tok == "" {
		return 400, map[string]any{"success": false, "error": "Missing orderParams.tokenId"}
	}
	price := numFloat(op["price"])
	size := numFloat(op["size"])
	if size < 1 {
		return 400, map[string]any{"success": false, "error": "Invalid size"}
	}
	tick := "0.01"
	if ts, ok := op["tickSize"].(string); ok && ts != "" {
		tick = ts
	}
	neg := false
	if n, ok := op["negRisk"].(bool); ok {
		neg = n
	}
	tickDec, _ := decimal.NewFromString(tick)
	negP := neg
	req := map[string]any{
		"tokenId": tok,
		"side":    "BUY",
		"price":   decimal.NewFromFloat(price).StringFixed(4),
		"size":    decimal.NewFromFloat(math.Floor(size)).StringFixed(0),
		"tickSize": tickDec.String(),
		"negRisk": &negP,
	}
	return t.postLimitAndShape(req)
}

func (t *Trading) legacyToLimitOrder(wrap map[string]any) (map[string]any, error) {
	slug, _ := wrap["marketSlug"].(string)
	cond, _ := wrap["conditionId"].(string)
	side, _ := wrap["side"].(string)
	budget, _ := wrap["budgetUsd"].(float64)
	if slug == "" || cond == "" {
		return nil, fmt.Errorf("Missing conditionId or marketSlug or orderParams")
	}
	if side != "YES" && side != "NO" {
		return nil, fmt.Errorf("Invalid side")
	}
	if budget < 1 || budget > 100 {
		return nil, fmt.Errorf("Invalid budgetUsd")
	}
	gammaURL := strings.TrimSpace(t.root.Hft.Polymarket.GammaURL)
	if gammaURL == "" {
		gammaURL = "https://gamma-api.polymarket.com"
	}
	gc := gamma.New(gammaURL)
	raw, err := gc.MarketBySlug(slug)
	if err != nil {
		return nil, fmt.Errorf("Market not found: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	clob, _ := m["clobTokenIds"].(string)
	var ids []string
	_ = json.Unmarshal([]byte(clob), &ids)
	if len(ids) < 2 {
		return nil, fmt.Errorf("invalid clobTokenIds")
	}
	outcomes := `["Yes","No"]`
	if o, ok := m["outcomes"].(string); ok && o != "" {
		outcomes = o
	}
	pricesStr := `["0.5","0.5"]`
	if p, ok := m["outcomePrices"].(string); ok && p != "" {
		pricesStr = p
	}
	var outs []string
	var prices []float64
	_ = json.Unmarshal([]byte(outcomes), &outs)
	var ps []string
	_ = json.Unmarshal([]byte(pricesStr), &ps)
	for _, x := range ps {
		var f float64
		_, _ = fmt.Sscanf(x, "%f", &f)
		prices = append(prices, f)
	}
	yesIdx, noIdx := -1, -1
	for i, o := range outs {
		ol := strings.ToLower(o)
		if ol == "yes" || ol == "up" {
			yesIdx = i
		}
		if ol == "no" || ol == "down" {
			noIdx = i
		}
	}
	var tokenID string
	var cur float64
	if side == "YES" {
		if yesIdx == 0 {
			tokenID = ids[0]
		} else {
			tokenID = ids[1]
		}
		if yesIdx >= 0 && yesIdx < len(prices) {
			cur = prices[yesIdx]
		} else {
			cur = prices[0]
		}
	} else {
		if noIdx == 0 {
			tokenID = ids[0]
		} else {
			tokenID = ids[1]
		}
		if noIdx >= 0 && noIdx < len(prices) {
			cur = prices[noIdx]
		} else if len(prices) > 1 {
			cur = prices[1]
		}
	}
	if rp, ok := wrap["price"].(float64); ok && rp > 0 {
		cur = rp
	}
	tick := "0.01"
	orderPrice := math.Round(cur/0.01) * 0.01
	shares := math.Floor(budget / orderPrice)
	if shares < 5 {
		return nil, fmt.Errorf("Budget too small for minimum shares")
	}
	tickDec, _ := decimal.NewFromString(tick)
	f := false
	return map[string]any{
		"tokenId":  tokenID,
		"side":     "BUY",
		"price":    decimal.NewFromFloat(orderPrice).StringFixed(4),
		"size":     decimal.NewFromFloat(shares).StringFixed(0),
		"tickSize": tickDec.String(),
		"negRisk":  &f,
	}, nil
}

func (t *Trading) postLimitAndShape(req map[string]any) (int, map[string]any) {
	code, b, err := t.postExecutor("/api/polymarket/orders/limit", req)
	if err != nil {
		return 500, map[string]any{"success": false, "error": err.Error()}
	}
	var exec map[string]any
	_ = json.Unmarshal(b, &exec)
	ok := code >= 200 && code < 300
	if !ok {
		return code, map[string]any{"success": false, "error": string(b)}
	}
	// Shape similar to edge function order result
	tok, _ := req["tokenId"].(string)
	priceStr, _ := req["price"].(string)
	sizeStr, _ := req["size"].(string)
	var price, size float64
	_, _ = fmt.Sscanf(priceStr, "%f", &price)
	_, _ = fmt.Sscanf(sizeStr, "%f", &size)
	ord := map[string]any{
		"success": true, "orderId": nil, "status": "submitted", "tokenId": tok,
		"side": "YES", "price": price, "size": int(size), "costUsd": math.Round(size*price*100) / 100,
	}
	if id := orderIDFromExecutor(exec); id != "" {
		ord["orderId"] = id
	}
	return 200, map[string]any{
		"success": true,
		"data": map[string]any{
			"order": ord,
			"market": map[string]any{
				"slug": "", "title": "", "conditionId": "",
			},
		},
	}
}

func orderIDFromExecutor(exec map[string]any) string {
	if cr, ok := exec["clobResponse"].(map[string]any); ok {
		if id, ok := cr["orderID"].(string); ok && id != "" {
			return id
		}
		if id, ok := cr["orderId"].(string); ok && id != "" {
			return id
		}
	}
	return ""
}

// executorLimitOrderOutcome interprets POST /api/polymarket/orders/limit JSON (OrderSubmissionResult + nested clobResponse).
func executorLimitOrderOutcome(httpCode int, body []byte) (success bool, orderID string, errMsg string) {
	if httpCode < 200 || httpCode >= 300 {
		s := strings.TrimSpace(string(body))
		if s == "" {
			s = fmt.Sprintf("executor HTTP %d", httpCode)
		}
		return false, "", s
	}
	var exec map[string]any
	if err := json.Unmarshal(body, &exec); err != nil {
		return false, "", "executor returned invalid JSON"
	}
	cr, _ := exec["clobResponse"].(map[string]any)
	if cr != nil {
		if s, _ := cr["status"].(string); strings.EqualFold(s, "REJECTED") {
			reason, _ := cr["reason"].(string)
			if strings.TrimSpace(reason) == "" {
				reason = "order rejected"
			}
			return false, "", reason
		}
	}
	id := orderIDFromExecutor(exec)
	if id != "" {
		return true, id, ""
	}
	if cr != nil {
		if s, _ := cr["status"].(string); strings.EqualFold(s, "OPEN") {
			return true, "", ""
		}
	}
	return false, "", "missing order id in executor response"
}

// upDownClobTokenIDs maps Gamma outcome order to Up (Yes) and Down (No) CLOB token ids.
func upDownClobTokenIDs(m map[string]any, ids []string) (upTok, downTok string) {
	if len(ids) < 2 {
		return "", ""
	}
	outcomes := `["Yes","No"]`
	if o, ok := m["outcomes"].(string); ok && o != "" {
		outcomes = o
	}
	var outs []string
	_ = json.Unmarshal([]byte(outcomes), &outs)
	yesIdx, noIdx := -1, -1
	for i, o := range outs {
		ol := strings.ToLower(strings.TrimSpace(o))
		if ol == "yes" || ol == "up" {
			yesIdx = i
		}
		if ol == "no" || ol == "down" {
			noIdx = i
		}
	}
	if yesIdx >= 0 && yesIdx < len(ids) && noIdx >= 0 && noIdx < len(ids) {
		return ids[yesIdx], ids[noIdx]
	}
	return ids[0], ids[1]
}

func limitOrderBotOrderResponse(ok bool, orderID, errMsg string) map[string]any {
	o := map[string]any{"success": ok}
	if orderID != "" {
		o["orderId"] = orderID
	}
	if errMsg != "" {
		o["errorMsg"] = errMsg
	}
	return o
}

func numFloat(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case json.Number:
		f, err := t.Float64()
		if err != nil {
			return 0
		}
		return f
	default:
		return 0
	}
}

var assetSlugPrefix = map[string]string{
	"BTC": "btc-updown-15m-",
	"SOL": "sol-updown-15m-",
	"ETH": "eth-updown-15m-",
	"XRP": "xrp-updown-15m-",
}

// limitOrderBotLoad15mMarket resolves the next 15m Gamma market for asset (slug ceil(ts/900)*900).
// On failure failBody is non-nil and should be returned as the HTTP handler body; failCode is the status.
func (t *Trading) limitOrderBotLoad15mMarket(asset string) (next int, slug string, upTok, downTok, title string, failCode int, failBody map[string]any) {
	ts := time.Now().Unix()
	next = int(math.Ceil(float64(ts)/900) * 900)
	slug = assetSlugPrefix[asset] + fmt.Sprintf("%d", next)
	gammaURL := strings.TrimSpace(t.root.Hft.Polymarket.GammaURL)
	if gammaURL == "" {
		gammaURL = "https://gamma-api.polymarket.com"
	}
	gc := gamma.New(gammaURL)
	raw, err := gc.MarketBySlug(slug)
	if err != nil {
		return 0, "", "", "", "", 200, map[string]any{"success": false, "error": "Market not found - may not be created yet", "logs": []any{}}
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return 0, "", "", "", "", 500, map[string]any{"success": false, "error": "market json", "logs": []any{}}
	}
	clob, _ := m["clobTokenIds"].(string)
	var ids []string
	_ = json.Unmarshal([]byte(clob), &ids)
	if len(ids) < 2 {
		return 0, "", "", "", "", 500, map[string]any{"success": false, "error": "token ids", "logs": []any{}}
	}
	upTok, downTok = upDownClobTokenIDs(m, ids)
	if upTok == "" || downTok == "" {
		return 0, "", "", "", "", 500, map[string]any{"success": false, "error": "could not resolve up/down token ids", "logs": []any{}}
	}
	for _, k := range []string{"question", "title"} {
		if s, ok := m[k].(string); ok && strings.TrimSpace(s) != "" {
			title = strings.TrimSpace(s)
			break
		}
	}
	return next, slug, upTok, downTok, title, 0, nil
}

func (t *Trading) limitOrderBotPostLimitBuy(tokenID string, price, shares float64, tickDec decimal.Decimal, negRisk bool) map[string]any {
	f := negRisk
	lr := map[string]any{
		"tokenId":  tokenID,
		"side":     "BUY",
		"price":    decimal.NewFromFloat(price).StringFixed(4),
		"size":     decimal.NewFromFloat(shares).StringFixed(0),
		"tickSize": tickDec.String(),
		"negRisk":  &f,
	}
	code, b, perr := t.postExecutor("/api/polymarket/orders/limit", lr)
	if perr != nil {
		return limitOrderBotOrderResponse(false, "", perr.Error())
	}
	ok, oid, emsg := executorLimitOrderOutcome(code, b)
	return limitOrderBotOrderResponse(ok, oid, emsg)
}

func (t *Trading) limitOrderBotLadder(asset string, sizeUsd float64, lad map[string]any) (int, map[string]any) {
	maxP := defaultLadderMaxPrice
	minP := defaultLadderMinPrice
	taper := defaultLadderTaper
	if lad != nil {
		if v := numFloat(lad["maxPrice"]); v > 0 {
			maxP = int(v)
		}
		if v := numFloat(lad["minPrice"]); v > 0 {
			minP = int(v)
		}
		if v := numFloat(lad["taperFactor"]); v > 0 {
			taper = v
		}
	}
	rungs, err := CalculateLadderRungs(sizeUsd, maxP, minP, taper)
	if err != nil {
		return 400, map[string]any{"success": false, "error": err.Error(), "logs": []any{}}
	}
	for _, r := range rungs {
		price := float64(r.PricePercent) / 100
		sharesEach := math.Floor((r.SizeUsd / 2) / price)
		if sharesEach < 5 {
			return 400, map[string]any{
				"success": false,
				"error": fmt.Sprintf(
					"rung at %d%%: allocation $%.2f yields only %.0f shares per side (min 5); increase bankroll or narrow ladder",
					r.PricePercent, r.SizeUsd, sharesEach,
				),
				"logs": []any{},
			}
		}
	}
	next, slug, upTok, downTok, title, failCode, failBody := t.limitOrderBotLoad15mMarket(asset)
	if failBody != nil {
		return failCode, failBody
	}
	tickDec := decimal.RequireFromString("0.01")
	var ladderOrders []any
	totalOrders := 0
	successCount := 0
	for _, r := range rungs {
		price := float64(r.PricePercent) / 100
		sharesEach := math.Floor((r.SizeUsd / 2) / price)
		upRes := t.limitOrderBotPostLimitBuy(upTok, price, sharesEach, tickDec, false)
		downRes := t.limitOrderBotPostLimitBuy(downTok, price, sharesEach, tickDec, false)
		totalOrders += 2
		if ok, _ := upRes["success"].(bool); ok {
			successCount++
		}
		if ok, _ := downRes["success"].(bool); ok {
			successCount++
		}
		ladderOrders = append(ladderOrders, map[string]any{
			"pricePercent": r.PricePercent,
			"sizeUsd":      r.SizeUsd,
			"up":           upRes,
			"down":         downRes,
		})
	}
	startRFC3339 := time.Unix(int64(next), 0).UTC().Format(time.RFC3339)
	market := map[string]any{
		"marketSlug":             slug,
		"marketStartTime":        startRFC3339,
		"targetTimestamp":        next,
		"ladderOrdersPlaced":     ladderOrders,
		"ladderTotalOrders":      totalOrders,
		"ladderSuccessfulOrders": successCount,
	}
	if title != "" {
		market["marketTitle"] = title
	}
	if successCount == 0 && totalOrders > 0 {
		market["error"] = "all ladder order attempts failed"
	}
	return 200, map[string]any{
		"success": true,
		"data": map[string]any{
			"asset":        asset,
			"pricePercent": float64(maxP),
			"sizeUsd":      sizeUsd,
			"ladderMode":   true,
			"market":       market,
		},
		"logs": []any{},
	}
}

// LimitOrderBot places up to two BUY limits on up/down tokens via executor (paper mode).
func (t *Trading) LimitOrderBot(ctx context.Context, body []byte) (int, map[string]any) {
	_ = ctx
	var req map[string]any
	_ = json.Unmarshal(body, &req)
	asset, _ := req["asset"].(string)
	asset = strings.ToUpper(strings.TrimSpace(asset))
	if _, ok := assetSlugPrefix[asset]; !ok {
		return 400, map[string]any{"success": false, "error": "Invalid asset. Must be one of: BTC, SOL, ETH, XRP", "logs": []any{}}
	}
	sizeUsd := 25.0
	if v := numFloat(req["sizeUsd"]); v > 0 {
		sizeUsd = v
	}
	var lad map[string]any
	if l, ok := req["ladder"].(map[string]any); ok {
		lad = l
	}
	if lad != nil {
		if en, ok := lad["enabled"].(bool); ok && en {
			return t.limitOrderBotLadder(asset, sizeUsd, lad)
		}
	}
	pricePct := 48.0
	if v := numFloat(req["price"]); v > 0 {
		pricePct = v
	}
	price := pricePct / 100
	next, slug, upTok, downTok, title, failCode, failBody := t.limitOrderBotLoad15mMarket(asset)
	if failBody != nil {
		return failCode, failBody
	}
	sharesEach := math.Floor(sizeUsd / 2 / price)
	if sharesEach < 5 {
		return 400, map[string]any{
			"success": false,
			"error":   "sizeUsd too small: each side needs at least 5 shares (Polymarket minimum). Increase sizeUsd or lower price.",
			"logs":    []any{},
		}
	}
	tickDec := decimal.RequireFromString("0.01")
	upRes := t.limitOrderBotPostLimitBuy(upTok, price, sharesEach, tickDec, false)
	downRes := t.limitOrderBotPostLimitBuy(downTok, price, sharesEach, tickDec, false)
	upOk, _ := upRes["success"].(bool)
	downOk, _ := downRes["success"].(bool)
	marketErr := ""
	if !upOk && !downOk {
		uem, _ := upRes["errorMsg"].(string)
		dem, _ := downRes["errorMsg"].(string)
		marketErr = fmt.Sprintf("up: %s; down: %s", uem, dem)
	}
	startRFC3339 := time.Unix(int64(next), 0).UTC().Format(time.RFC3339)
	market := map[string]any{
		"marketSlug":      slug,
		"marketStartTime": startRFC3339,
		"targetTimestamp": next,
		"ordersPlaced": map[string]any{
			"up":   upRes,
			"down": downRes,
		},
	}
	if title != "" {
		market["marketTitle"] = title
	}
	if marketErr != "" {
		market["error"] = marketErr
	}
	return 200, map[string]any{
		"success": true,
		"data": map[string]any{
			"asset":        asset,
			"pricePercent": pricePct,
			"sizeUsd":      sizeUsd,
			"ladderMode":   false,
			"market":       market,
		},
		"logs": []any{},
	}
}

func errString(e error) string {
	if e == nil {
		return ""
	}
	return e.Error()
}

// PositionTracker uses Gamma + data-api (public) for wallet positions.
func (t *Trading) PositionTracker(ctx context.Context, body []byte) (int, map[string]any) {
	var req map[string]any
	_ = json.Unmarshal(body, &req)
	asset, _ := req["asset"].(string)
	asset = strings.ToUpper(strings.TrimSpace(asset))
	if _, ok := assetSlugPrefix[asset]; !ok {
		return 200, map[string]any{"success": false, "error": "Invalid asset", "logs": []any{}}
	}
	addr := strings.TrimSpace(os.Getenv("POLYMARKET_PROXY_WALLET_ADDRESS"))
	if addr == "" {
		return 500, map[string]any{"success": false, "error": "POLYMARKET_PROXY_WALLET_ADDRESS not set", "logs": []any{}}
	}
	ts := time.Now().Unix()
	block := int(math.Floor(float64(ts)/900) * 900)
	slug, _ := req["marketSlug"].(string)
	if slug == "" {
		slug = assetSlugPrefix[asset] + fmt.Sprintf("%d", block)
	}
	gammaURL := strings.TrimSpace(t.root.Hft.Polymarket.GammaURL)
	if gammaURL == "" {
		gammaURL = "https://gamma-api.polymarket.com"
	}
	gc := gamma.New(gammaURL)
	raw, err := gc.MarketBySlug(slug)
	if err != nil {
		return 200, map[string]any{"success": false, "error": "Market not found - may not be created yet", "logs": []any{}}
	}
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	clob, _ := m["clobTokenIds"].(string)
	var ids []string
	_ = json.Unmarshal([]byte(clob), &ids)
	if len(ids) < 2 {
		return 200, map[string]any{"success": false, "error": "tokens", "logs": []any{}}
	}
	u := fmt.Sprintf("https://data-api.polymarket.com/positions?user=%s&sizeThreshold=0", url.QueryEscape(addr))
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return 500, map[string]any{"success": false, "error": err.Error()}
	}
	resp, err := t.httpClient().Do(r)
	if err != nil {
		return 502, map[string]any{"success": false, "error": err.Error()}
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	var positions []map[string]any
	_ = json.Unmarshal(b, &positions)
	up, down := ids[0], ids[1]
	var upSz, downSz float64
	for _, p := range positions {
		aid, _ := p["asset"].(string)
		sz, _ := p["size"].(float64)
		if aid == up {
			upSz = sz
		}
		if aid == down {
			downSz = sz
		}
	}
	return 200, map[string]any{
		"success": true,
		"data": map[string]any{
			"asset": asset,
			"position": map[string]any{
				"marketSlug": slug,
				"upShares":   upSz,
				"downShares": downSz,
			},
		},
		"logs": []any{},
	}
}
