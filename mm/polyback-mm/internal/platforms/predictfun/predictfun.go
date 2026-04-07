package predictfun

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/platforms"
	"github.com/shopspring/decimal"
)

// API routes are under /v1 (see public 404 on /markets without version).
const defaultBaseURL = "https://api.predict.fun/v1"

// PredictFun is the Predict.fun REST client (API key + optional JWT via wallet signature).
type PredictFun struct {
	baseURL    string
	apiKey     string
	privateKey string

	mu     sync.Mutex
	jwt    string
	client *http.Client
}

var _ platforms.Platform = (*PredictFun)(nil)

// NewPredictFun creates a client. apiKey is required; privateKey is needed for Authenticate and protected writes.
func NewPredictFun(baseURL, apiKey, privateKeyHex string) *PredictFun {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultBaseURL
	}
	return &PredictFun{
		baseURL:    strings.TrimSuffix(strings.TrimSpace(baseURL), "/"),
		apiKey:     strings.TrimSpace(apiKey),
		privateKey: strings.TrimSpace(privateKeyHex),
		client:     platforms.DefaultHTTPClient(),
	}
}

func (p *PredictFun) Name() string { return "predict_fun" }

func (p *PredictFun) baseHeaders() map[string]string {
	h := map[string]string{"X-API-Key": p.apiKey}
	p.mu.Lock()
	tok := p.jwt
	p.mu.Unlock()
	if tok != "" {
		h["Authorization"] = "Bearer " + tok
	}
	return h
}

// Authenticate performs EIP-191 sign + JWT exchange.
func (p *PredictFun) Authenticate(ctx context.Context) error {
	if p.apiKey == "" {
		return platforms.ErrNotConfigured
	}
	if p.privateKey == "" {
		return fmt.Errorf("predictfun: %w (private key)", platforms.ErrNotConfigured)
	}
	key, err := crypto.HexToECDSA(strings.TrimPrefix(p.privateKey, "0x"))
	if err != nil {
		return fmt.Errorf("predictfun: private key: %w", err)
	}

	b, err := platforms.DoJSONExpect2xx(ctx, p.client, http.MethodGet, p.baseURL+"/auth/message", map[string]string{
		"X-API-Key": p.apiKey,
	}, nil)
	if err != nil {
		return err
	}
	var msgWrap struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(b, &msgWrap); err != nil {
		return err
	}
	if msgWrap.Message == "" {
		return fmt.Errorf("predictfun: empty auth message")
	}
	sig, err := signEthereumPersonal(msgWrap.Message, key)
	if err != nil {
		return err
	}
	addr := crypto.PubkeyToAddress(key.PublicKey).Hex()
	body, err := json.Marshal(map[string]string{
		"message":   msgWrap.Message,
		"signature": sig,
		"address":   addr,
	})
	if err != nil {
		return err
	}
	b2, err := platforms.DoJSONExpect2xx(ctx, p.client, http.MethodPost, p.baseURL+"/auth/jwt", map[string]string{
		"X-API-Key":    p.apiKey,
		"Content-Type": "application/json",
		"Accept":       "application/json",
	}, body)
	if err != nil {
		return err
	}
	var jwtWrap struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(b2, &jwtWrap); err != nil {
		return err
	}
	if jwtWrap.Token == "" {
		return fmt.Errorf("predictfun: empty jwt in response")
	}
	p.mu.Lock()
	p.jwt = jwtWrap.Token
	p.mu.Unlock()
	return nil
}

func signEthereumPersonal(message string, key *ecdsa.PrivateKey) (string, error) {
	prefix := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), message)
	hash := crypto.Keccak256Hash([]byte(prefix))
	sig, err := crypto.Sign(hash.Bytes(), key)
	if err != nil {
		return "", err
	}
	return "0x" + hex.EncodeToString(sig), nil
}

func (p *PredictFun) HealthCheck(ctx context.Context) error {
	if p.apiKey == "" {
		return platforms.ErrNotConfigured
	}
	_, err := p.GetAllMarkets(ctx)
	return err
}

func (p *PredictFun) GetAllMarkets(ctx context.Context) ([]platforms.Market, error) {
	if p.apiKey == "" {
		return nil, platforms.ErrNotConfigured
	}
	u := p.baseURL + "/markets?" + url.Values{
		"status": {"active"},
		"limit":  {"100"},
	}.Encode()
	b, err := platforms.DoJSONExpect2xx(ctx, p.client, http.MethodGet, u, p.baseHeaders(), nil)
	if err != nil {
		return nil, err
	}
	var wrap struct {
		Markets []json.RawMessage `json:"markets"`
	}
	if err := json.Unmarshal(b, &wrap); err != nil {
		return nil, err
	}
	out := make([]platforms.Market, 0, len(wrap.Markets))
	for _, raw := range wrap.Markets {
		m, err := parsePredictFunMarket(raw)
		if err != nil {
			continue
		}
		out = append(out, m)
	}
	return out, nil
}

func (p *PredictFun) GetMarket(ctx context.Context, marketID string) (*platforms.Market, error) {
	if p.apiKey == "" {
		return nil, platforms.ErrNotConfigured
	}
	b, err := platforms.DoJSONExpect2xx(ctx, p.client, http.MethodGet, p.baseURL+"/markets/"+url.PathEscape(marketID), p.baseHeaders(), nil)
	if err != nil {
		return nil, err
	}
	m, err := parsePredictFunMarket(b)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (p *PredictFun) GetPrices(ctx context.Context, marketID string) (yes, no decimal.Decimal, err error) {
	m, err := p.GetMarket(ctx, marketID)
	if err != nil {
		return decimal.Zero, decimal.Zero, err
	}
	return m.YesPrice, m.NoPrice, nil
}

func (p *PredictFun) GetOrderbook(ctx context.Context, marketID string) (platforms.Orderbook, error) {
	if p.apiKey == "" {
		return platforms.Orderbook{}, platforms.ErrNotConfigured
	}
	b, err := platforms.DoJSONExpect2xx(ctx, p.client, http.MethodGet, p.baseURL+"/markets/"+url.PathEscape(marketID)+"/orderbook", p.baseHeaders(), nil)
	if err != nil {
		return platforms.Orderbook{}, err
	}
	var ob struct {
		Bids []struct {
			Price interface{} `json:"price"`
			Size  interface{} `json:"size"`
		} `json:"bids"`
		Asks []struct {
			Price interface{} `json:"price"`
			Size  interface{} `json:"size"`
		} `json:"asks"`
	}
	if err := json.Unmarshal(b, &ob); err != nil {
		return platforms.Orderbook{}, err
	}
	out := platforms.Orderbook{}
	for _, x := range ob.Bids {
		out.Bids = append(out.Bids, platforms.BookLevel{
			Price: platforms.DecimalFromInterface(x.Price),
			Size:  platforms.DecimalFromInterface(x.Size),
		})
	}
	for _, x := range ob.Asks {
		out.Asks = append(out.Asks, platforms.BookLevel{
			Price: platforms.DecimalFromInterface(x.Price),
			Size:  platforms.DecimalFromInterface(x.Size),
		})
	}
	return out, nil
}

func (p *PredictFun) ensureJWT(ctx context.Context) error {
	p.mu.Lock()
	ok := p.jwt != ""
	p.mu.Unlock()
	if ok {
		return nil
	}
	return p.Authenticate(ctx)
}

func (p *PredictFun) SendOrder(ctx context.Context, req platforms.PlaceOrderRequest) (*platforms.OrderResult, error) {
	if p.apiKey == "" {
		return nil, platforms.ErrNotConfigured
	}
	if err := p.ensureJWT(ctx); err != nil {
		return &platforms.OrderResult{Success: false, Error: err.Error()}, err
	}
	payload, err := json.Marshal(map[string]string{
		"marketId": req.MarketID,
		"side":     strings.ToUpper(strings.TrimSpace(req.Side)),
		"size":     req.Size.String(),
		"price":    req.Price.String(),
		"type":     "LIMIT",
	})
	if err != nil {
		return nil, err
	}
	b, err := platforms.DoJSONExpect2xx(ctx, p.client, http.MethodPost, p.baseURL+"/orders", p.baseHeaders(), payload)
	if err != nil {
		return &platforms.OrderResult{Success: false, Error: err.Error()}, err
	}
	var res struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	_ = json.Unmarshal(b, &res)
	return &platforms.OrderResult{Success: true, OrderID: res.ID, Status: res.Status, Raw: b}, nil
}

func (p *PredictFun) CancelOrder(ctx context.Context, orderID string) error {
	if p.apiKey == "" {
		return platforms.ErrNotConfigured
	}
	if err := p.ensureJWT(ctx); err != nil {
		return err
	}
	_, err := platforms.DoJSONExpect2xx(ctx, p.client, http.MethodDelete, p.baseURL+"/orders/"+url.PathEscape(orderID), p.baseHeaders(), nil)
	return err
}

func (p *PredictFun) ListOrders(ctx context.Context, marketID *string) ([]platforms.Order, error) {
	if p.apiKey == "" {
		return nil, platforms.ErrNotConfigured
	}
	if err := p.ensureJWT(ctx); err != nil {
		return nil, err
	}
	q := url.Values{}
	if marketID != nil && *marketID != "" {
		q.Set("marketId", *marketID)
	}
	u := p.baseURL + "/orders"
	if enc := q.Encode(); enc != "" {
		u += "?" + enc
	}
	b, err := platforms.DoJSONExpect2xx(ctx, p.client, http.MethodGet, u, p.baseHeaders(), nil)
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
		o, err := parsePredictFunOrder(raw)
		if err != nil {
			continue
		}
		out = append(out, o)
	}
	return out, nil
}

func (p *PredictFun) GetPositions(ctx context.Context) ([]platforms.Position, error) {
	if p.apiKey == "" {
		return nil, platforms.ErrNotConfigured
	}
	if err := p.ensureJWT(ctx); err != nil {
		return nil, err
	}
	b, err := platforms.DoJSONExpect2xx(ctx, p.client, http.MethodGet, p.baseURL+"/positions", p.baseHeaders(), nil)
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
		pos, err := parsePredictFunPosition(raw)
		if err != nil {
			continue
		}
		out = append(out, pos)
	}
	return out, nil
}

func (p *PredictFun) GetBalance(ctx context.Context) (map[string]decimal.Decimal, error) {
	if p.apiKey == "" {
		return nil, platforms.ErrNotConfigured
	}
	if err := p.ensureJWT(ctx); err != nil {
		return nil, err
	}
	b, err := platforms.DoJSONExpect2xx(ctx, p.client, http.MethodGet, p.baseURL+"/account", p.baseHeaders(), nil)
	if err != nil {
		return nil, err
	}
	var a struct {
		Balance       json.RawMessage `json:"balance"`
		LockedBalance json.RawMessage `json:"lockedBalance"`
	}
	_ = json.Unmarshal(b, &a)
	return map[string]decimal.Decimal{
		"USDC":   platforms.DecimalFromJSON(a.Balance),
		"locked": platforms.DecimalFromJSON(a.LockedBalance),
	}, nil
}

func parsePredictFunMarket(raw []byte) (platforms.Market, error) {
	var m struct {
		ID          interface{}     `json:"id"`
		Slug        string          `json:"slug"`
		Question    string          `json:"question"`
		Title       string          `json:"title"`
		Description string          `json:"description"`
		Category    string          `json:"category"`
		YesPrice    json.RawMessage `json:"yesPrice"`
		NoPrice     json.RawMessage `json:"noPrice"`
		YesTokenID  string          `json:"yesTokenId"`
		NoTokenID   string          `json:"noTokenId"`
		Volume24h   json.RawMessage `json:"volume24h"`
		Liquidity   json.RawMessage `json:"liquidity"`
		EndDate     string          `json:"endDate"`
		Status      string          `json:"status"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return platforms.Market{}, err
	}
	q := m.Question
	if q == "" {
		q = m.Title
	}
	var exp *time.Time
	if m.EndDate != "" {
		if t, err := time.Parse(time.RFC3339, strings.ReplaceAll(m.EndDate, "Z", "+00:00")); err == nil {
			exp = &t
		}
	}
	yes := platforms.DecimalFromJSON(m.YesPrice)
	no := platforms.DecimalFromJSON(m.NoPrice)
	if yes.IsZero() && no.IsZero() {
		yes = decimal.NewFromFloat(0.5)
		no = decimal.NewFromFloat(0.5)
	}
	return platforms.Market{
		ID:          fmt.Sprint(m.ID),
		Slug:        m.Slug,
		Question:    q,
		Description: m.Description,
		Category:    m.Category,
		YesPrice:    yes,
		NoPrice:     no,
		YesTokenID:  m.YesTokenID,
		NoTokenID:   m.NoTokenID,
		Volume24h:   platforms.DecimalFromJSON(m.Volume24h),
		Liquidity:   platforms.DecimalFromJSON(m.Liquidity),
		ExpiresAt:   exp,
		Active:      strings.EqualFold(m.Status, "active"),
		Raw:         raw,
	}, nil
}

func parsePredictFunOrder(raw []byte) (platforms.Order, error) {
	var o struct {
		ID       interface{}     `json:"id"`
		MarketID interface{}     `json:"marketId"`
		Side     string          `json:"side"`
		Size     json.RawMessage `json:"size"`
		Price    json.RawMessage `json:"price"`
		Type     string          `json:"type"`
		Status   string          `json:"status"`
	}
	if err := json.Unmarshal(raw, &o); err != nil {
		return platforms.Order{}, err
	}
	return platforms.Order{
		ID:        fmt.Sprint(o.ID),
		MarketID:  fmt.Sprint(o.MarketID),
		Side:      strings.ToLower(o.Side),
		Size:      platforms.DecimalFromJSON(o.Size),
		Price:     platforms.DecimalFromJSON(o.Price),
		OrderType: strings.ToLower(o.Type),
		Status:    strings.ToLower(o.Status),
		Raw:       raw,
	}, nil
}

func parsePredictFunPosition(raw []byte) (platforms.Position, error) {
	var p struct {
		MarketID     interface{}     `json:"marketId"`
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
		MarketID:     fmt.Sprint(p.MarketID),
		Side:         strings.ToLower(p.Side),
		Size:         platforms.DecimalFromJSON(p.Size),
		AvgPrice:     platforms.DecimalFromJSON(p.AvgPrice),
		CurrentPrice: platforms.DecimalFromJSON(p.CurrentPrice),
		PnL:          platforms.DecimalFromJSON(p.PnL),
	}, nil
}
