package limitless

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/platforms"
	"github.com/shopspring/decimal"
)

const defaultBaseURL = "https://api.limitless.exchange"

// Limitless is the Limitless Exchange REST client (wallet-scoped trading, cryptomaid limitless_full shape).
type Limitless struct {
	baseURL string
	apiKey  string
	wallet  string
	client  *http.Client
}

var _ platforms.Platform = (*Limitless)(nil)

// NewLimitless creates a client. apiKey is required; wallet is required for trading and account endpoints.
func NewLimitless(baseURL, apiKey, walletAddress string) *Limitless {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultBaseURL
	}
	w := strings.TrimSpace(strings.ToLower(walletAddress))
	return &Limitless{
		baseURL: strings.TrimSuffix(strings.TrimSpace(baseURL), "/"),
		apiKey:  strings.TrimSpace(apiKey),
		wallet:  w,
		client:  platforms.DefaultHTTPClient(),
	}
}

func (l *Limitless) Name() string { return "limitless" }

func (l *Limitless) headers() map[string]string {
	if l.apiKey == "" {
		return map[string]string{}
	}
	return map[string]string{"X-API-Key": l.apiKey}
}

func (l *Limitless) HealthCheck(ctx context.Context) error {
	_, err := l.GetAllMarkets(ctx)
	return err
}

func (l *Limitless) GetAllMarkets(ctx context.Context) ([]platforms.Market, error) {
	b, err := platforms.DoJSONExpect2xx(ctx, l.client, http.MethodGet, l.baseURL+"/markets/active", l.headers(), nil)
	if err != nil {
		return nil, err
	}
	items, err := platforms.DecodeJSONArrayEnvelope(b)
	if err != nil {
		return nil, err
	}
	out := make([]platforms.Market, 0, len(items))
	for _, raw := range items {
		m, err := parseLimitlessMarket(raw)
		if err != nil {
			continue
		}
		out = append(out, m)
	}
	return out, nil
}

func (l *Limitless) GetMarket(ctx context.Context, marketID string) (*platforms.Market, error) {
	b, err := platforms.DoJSONExpect2xx(ctx, l.client, http.MethodGet, l.baseURL+"/markets/"+url.PathEscape(marketID), l.headers(), nil)
	if err != nil {
		return nil, err
	}
	var wrap struct {
		Market json.RawMessage `json:"market"`
	}
	if err := json.Unmarshal(b, &wrap); err == nil && len(wrap.Market) > 0 {
		m, err := parseLimitlessMarketDetail(wrap.Market)
		if err != nil {
			return nil, err
		}
		return &m, nil
	}
	m, err := parseLimitlessMarket(b)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (l *Limitless) GetPrices(ctx context.Context, marketID string) (yes, no decimal.Decimal, err error) {
	m, err := l.GetMarket(ctx, marketID)
	if err != nil {
		return decimal.Zero, decimal.Zero, err
	}
	return m.YesPrice, m.NoPrice, nil
}

func (l *Limitless) GetOrderbook(ctx context.Context, marketID string) (platforms.Orderbook, error) {
	b, err := platforms.DoJSONExpect2xx(ctx, l.client, http.MethodGet, l.baseURL+"/markets/"+url.PathEscape(marketID)+"/orderbook", l.headers(), nil)
	if err != nil {
		return platforms.Orderbook{}, err
	}
	return parseLimitlessOrderbook(b)
}

func (l *Limitless) SendOrder(ctx context.Context, req platforms.PlaceOrderRequest) (*platforms.OrderResult, error) {
	if l.apiKey == "" || l.wallet == "" {
		return nil, platforms.ErrNotConfigured
	}
	typ := strings.TrimSpace(req.Type)
	if typ == "" {
		typ = "limit"
	}
	payload, err := json.Marshal(map[string]string{
		"marketId":      req.MarketID,
		"side":          strings.ToUpper(strings.TrimSpace(req.Side)),
		"size":          req.Size.String(),
		"price":         req.Price.String(),
		"type":          strings.ToUpper(typ),
		"walletAddress": l.wallet,
	})
	if err != nil {
		return nil, err
	}
	b, err := platforms.DoJSONExpect2xx(ctx, l.client, http.MethodPost, l.baseURL+"/orders", l.headers(), payload)
	if err != nil {
		return &platforms.OrderResult{Success: false, Error: err.Error()}, err
	}
	var env struct {
		OrderID string `json:"orderId"`
		ID      string `json:"id"`
		Status  string `json:"status"`
	}
	_ = json.Unmarshal(b, &env)
	oid := env.OrderID
	if oid == "" {
		oid = env.ID
	}
	return &platforms.OrderResult{Success: true, OrderID: oid, Status: env.Status, Raw: b}, nil
}

func (l *Limitless) CancelOrder(ctx context.Context, orderID string) error {
	if l.apiKey == "" || l.wallet == "" {
		return platforms.ErrNotConfigured
	}
	body, err := json.Marshal(map[string]string{"walletAddress": l.wallet})
	if err != nil {
		return err
	}
	_, err = platforms.DoJSONExpect2xx(ctx, l.client, http.MethodDelete, l.baseURL+"/orders/"+url.PathEscape(orderID), l.headers(), body)
	return err
}

func (l *Limitless) ListOrders(ctx context.Context, marketID *string) ([]platforms.Order, error) {
	if l.apiKey == "" || l.wallet == "" {
		return nil, platforms.ErrNotConfigured
	}
	q := url.Values{}
	q.Set("walletAddress", l.wallet)
	if marketID != nil && *marketID != "" {
		q.Set("marketId", *marketID)
	}
	u := l.baseURL + "/orders?" + q.Encode()
	b, err := platforms.DoJSONExpect2xx(ctx, l.client, http.MethodGet, u, l.headers(), nil)
	if err != nil {
		return nil, err
	}
	var wrap struct {
		Orders []json.RawMessage `json:"orders"`
	}
	if err := json.Unmarshal(b, &wrap); err != nil {
		return nil, err
	}
	out := make([]platforms.Order, 0, len(wrap.Orders))
	for _, raw := range wrap.Orders {
		o, err := parseLimitlessOrder(raw)
		if err != nil {
			continue
		}
		out = append(out, o)
	}
	return out, nil
}

func (l *Limitless) GetPositions(ctx context.Context) ([]platforms.Position, error) {
	if l.apiKey == "" || l.wallet == "" {
		return nil, platforms.ErrNotConfigured
	}
	u := l.baseURL + "/account/positions?" + url.Values{"walletAddress": {l.wallet}}.Encode()
	b, err := platforms.DoJSONExpect2xx(ctx, l.client, http.MethodGet, u, l.headers(), nil)
	if err != nil {
		return nil, err
	}
	var wrap struct {
		Positions []json.RawMessage `json:"positions"`
	}
	if err := json.Unmarshal(b, &wrap); err != nil {
		return nil, err
	}
	out := make([]platforms.Position, 0, len(wrap.Positions))
	for _, raw := range wrap.Positions {
		p, err := parseLimitlessPosition(raw)
		if err != nil {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

func (l *Limitless) GetBalance(ctx context.Context) (map[string]decimal.Decimal, error) {
	if l.apiKey == "" || l.wallet == "" {
		return nil, platforms.ErrNotConfigured
	}
	u := l.baseURL + "/account/balance?" + url.Values{"walletAddress": {l.wallet}}.Encode()
	b, err := platforms.DoJSONExpect2xx(ctx, l.client, http.MethodGet, u, l.headers(), nil)
	if err != nil {
		return nil, err
	}
	var data struct {
		USDCBalance string `json:"usdcBalance"`
		Points      string `json:"points"`
	}
	_ = json.Unmarshal(b, &data)
	usdc, _ := decimal.NewFromString(data.USDCBalance)
	pts, _ := decimal.NewFromString(data.Points)
	return map[string]decimal.Decimal{
		"USDC":   usdc,
		"points": pts,
	}, nil
}

func parseLimitlessMarket(raw []byte) (platforms.Market, error) {
	var m struct {
		ID                  interface{}       `json:"id"`
		Slug                string            `json:"slug"`
		Title               string            `json:"title"`
		Question            string            `json:"question"`
		Description         string            `json:"description"`
		Category            string            `json:"category"`
		Categories          []string          `json:"categories"`
		Prices              []json.RawMessage `json:"prices"`
		PositionIds         []string          `json:"positionIds"`
		VolumeFormatted     json.RawMessage   `json:"volumeFormatted"`
		LiquidityFormatted  json.RawMessage   `json:"liquidityFormatted"`
		Volume              json.RawMessage   `json:"volume"`
		Liquidity           json.RawMessage   `json:"liquidity"`
		EndDate             string            `json:"endDate"`
		ExpirationTimestamp int64             `json:"expirationTimestamp"`
		Status              string            `json:"status"`
		Expired             bool              `json:"expired"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return platforms.Market{}, err
	}
	yes := decimal.NewFromFloat(0.5)
	no := decimal.NewFromFloat(0.5)
	if len(m.Prices) >= 2 {
		yes = platforms.DecimalFromJSON(m.Prices[0]).Div(decimal.NewFromInt(100))
		no = platforms.DecimalFromJSON(m.Prices[1]).Div(decimal.NewFromInt(100))
	}
	q := m.Title
	if q == "" {
		q = m.Question
	}
	cat := m.Category
	if cat == "" && len(m.Categories) > 0 {
		cat = strings.Join(m.Categories, ",")
	}
	vol := platforms.DecimalFromJSON(m.VolumeFormatted)
	if vol.IsZero() {
		vol = platforms.DecimalFromJSON(m.Volume)
	}
	liq := platforms.DecimalFromJSON(m.LiquidityFormatted)
	if liq.IsZero() {
		liq = platforms.DecimalFromJSON(m.Liquidity)
	}
	var exp *time.Time
	if m.ExpirationTimestamp > 0 {
		t := time.UnixMilli(m.ExpirationTimestamp)
		exp = &t
	} else if m.EndDate != "" {
		if t, err := time.Parse(time.RFC3339, strings.ReplaceAll(m.EndDate, "Z", "+00:00")); err == nil {
			exp = &t
		}
	}
	yesTok, noTok := "", ""
	if len(m.PositionIds) > 0 {
		yesTok = m.PositionIds[0]
	}
	if len(m.PositionIds) > 1 {
		noTok = m.PositionIds[1]
	}
	// Live API uses FUNDED, ACTIVE, etc.; treat explicit terminal states as inactive.
	active := !m.Expired &&
		!strings.EqualFold(m.Status, "resolved") &&
		!strings.EqualFold(m.Status, "closed")

	return platforms.Market{
		ID:          fmt.Sprint(m.ID),
		Slug:        m.Slug,
		Question:    q,
		Description: m.Description,
		Category:    cat,
		YesPrice:    yes,
		NoPrice:     no,
		YesTokenID:  yesTok,
		NoTokenID:   noTok,
		Volume24h:   vol,
		Liquidity:   liq,
		ExpiresAt:   exp,
		Active:      active,
		Raw:         raw,
	}, nil
}

func parseLimitlessMarketDetail(raw []byte) (platforms.Market, error) {
	var m struct {
		ID          interface{}     `json:"id"`
		Slug        string          `json:"slug"`
		Question    string          `json:"question"`
		YesPrice    json.RawMessage `json:"yesPrice"`
		NoPrice     json.RawMessage `json:"noPrice"`
		Volume24h   json.RawMessage `json:"volume24h"`
		Liquidity   json.RawMessage `json:"liquidity"`
		Description string          `json:"description"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return platforms.Market{}, err
	}
	return platforms.Market{
		ID:          fmt.Sprint(m.ID),
		Slug:        m.Slug,
		Question:    m.Question,
		Description: m.Description,
		YesPrice:    platforms.DecimalFromJSON(m.YesPrice),
		NoPrice:     platforms.DecimalFromJSON(m.NoPrice),
		Volume24h:   platforms.DecimalFromJSON(m.Volume24h),
		Liquidity:   platforms.DecimalFromJSON(m.Liquidity),
		Active:      true,
		Raw:         raw,
	}, nil
}

func parseLimitlessOrderbook(raw []byte) (platforms.Orderbook, error) {
	var ob struct {
		Bids    []map[string]interface{} `json:"bids"`
		Asks    []map[string]interface{} `json:"asks"`
		YesBids []map[string]interface{} `json:"yesBids"`
		YesAsks []map[string]interface{} `json:"yesAsks"`
	}
	if err := json.Unmarshal(raw, &ob); err != nil {
		return platforms.Orderbook{}, err
	}
	bids, asks := ob.Bids, ob.Asks
	if len(bids) == 0 && len(asks) == 0 {
		bids, asks = ob.YesBids, ob.YesAsks
	}
	return levelsFromMaps(bids, asks)
}

func levelsFromMaps(bids, asks []map[string]interface{}) (platforms.Orderbook, error) {
	out := platforms.Orderbook{}
	for _, b := range bids {
		p, s, ok := mapPriceSize(b)
		if !ok {
			continue
		}
		out.Bids = append(out.Bids, platforms.BookLevel{Price: p, Size: s})
	}
	for _, a := range asks {
		p, s, ok := mapPriceSize(a)
		if !ok {
			continue
		}
		out.Asks = append(out.Asks, platforms.BookLevel{Price: p, Size: s})
	}
	return out, nil
}

func mapPriceSize(m map[string]interface{}) (decimal.Decimal, decimal.Decimal, bool) {
	pv, ok1 := m["price"]
	sv, ok2 := m["size"]
	if !ok1 || !ok2 {
		return decimal.Zero, decimal.Zero, false
	}
	return platforms.DecimalFromInterface(pv), platforms.DecimalFromInterface(sv), true
}

func parseLimitlessOrder(raw []byte) (platforms.Order, error) {
	var o struct {
		ID            string          `json:"id"`
		OrderID       string          `json:"orderId"`
		MarketID      string          `json:"marketId"`
		Side          string          `json:"side"`
		Size          json.RawMessage `json:"size"`
		Price         json.RawMessage `json:"price"`
		Status        string          `json:"status"`
		Type          string          `json:"type"`
		FilledSize    json.RawMessage `json:"filledSize"`
		RemainingSize json.RawMessage `json:"remainingSize"`
	}
	if err := json.Unmarshal(raw, &o); err != nil {
		return platforms.Order{}, err
	}
	id := o.ID
	if id == "" {
		id = o.OrderID
	}
	return platforms.Order{
		ID:            id,
		MarketID:      o.MarketID,
		Side:          strings.ToLower(o.Side),
		Size:          platforms.DecimalFromJSON(o.Size),
		Price:         platforms.DecimalFromJSON(o.Price),
		OrderType:     strings.ToLower(o.Type),
		Status:        strings.ToLower(o.Status),
		FilledSize:    platforms.DecimalFromJSON(o.FilledSize),
		RemainingSize: platforms.DecimalFromJSON(o.RemainingSize),
		Raw:           raw,
	}, nil
}

func parseLimitlessPosition(raw []byte) (platforms.Position, error) {
	var p struct {
		MarketID     string          `json:"marketId"`
		Side         string          `json:"side"`
		Size         json.RawMessage `json:"size"`
		AvgPrice     json.RawMessage `json:"avgPrice"`
		CurrentPrice json.RawMessage `json:"currentPrice"`
		PnL          json.RawMessage `json:"pnl"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return platforms.Position{}, err
	}
	return platforms.Position{
		MarketID:     p.MarketID,
		Side:         strings.ToLower(p.Side),
		Size:         platforms.DecimalFromJSON(p.Size),
		AvgPrice:     platforms.DecimalFromJSON(p.AvgPrice),
		CurrentPrice: platforms.DecimalFromJSON(p.CurrentPrice),
		PnL:          platforms.DecimalFromJSON(p.PnL),
	}, nil
}
