package polymarket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/platforms"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/platforms/polymarket/gamma"
	"github.com/shopspring/decimal"
)

const (
	defaultGammaURL = "https://gamma-api.polymarket.com"
	defaultCLOBURL  = "https://clob.polymarket.com"
)

// Polymarket is read-focused Gamma + public CLOB book access. Order placement uses the
// dedicated executor stack; SendOrder returns platforms.ErrTradingNotImplemented.
type Polymarket struct {
	gamma    *gamma.Client
	gammaURL string
	clob     string
	client   *http.Client
}

var _ platforms.Platform = (*Polymarket)(nil)

// NewPolymarket builds a Polymarket platforms.Platform client. Empty gammaURL uses the public Gamma host.
func NewPolymarket(gammaURL, clobURL string) *Polymarket {
	if strings.TrimSpace(gammaURL) == "" {
		gammaURL = defaultGammaURL
	}
	if strings.TrimSpace(clobURL) == "" {
		clobURL = defaultCLOBURL
	}
	gammaURL = strings.TrimSuffix(strings.TrimSpace(gammaURL), "/")
	return &Polymarket{
		gamma:    gamma.New(gammaURL),
		gammaURL: gammaURL,
		clob:     strings.TrimSuffix(strings.TrimSpace(clobURL), "/"),
		client:   newHTTPClient(),
	}
}

func (p *Polymarket) Name() string { return "polymarket" }

func (p *Polymarket) HealthCheck(ctx context.Context) error {
	_, err := p.GetAllMarkets(ctx)
	return err
}

func (p *Polymarket) GetAllMarkets(ctx context.Context) ([]platforms.Market, error) {
	raw, err := p.gamma.Markets(map[string]string{
		"active":       "true",
		"closed":       "false",
		"liquidityMin": "1000",
		"limit":        "100",
	})
	if err != nil {
		return nil, err
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil, fmt.Errorf("polymarket: markets json: %w", err)
	}
	out := make([]platforms.Market, 0, len(arr))
	for _, m := range arr {
		pm, err := parseGammaMarket(m)
		if err != nil {
			continue
		}
		out = append(out, pm)
	}
	return out, nil
}

func (p *Polymarket) GetMarket(ctx context.Context, marketID string) (*platforms.Market, error) {
	u := fmt.Sprintf("%s/markets/%s", p.gammaURL, url.PathEscape(marketID))
	b, err := doJSONExpect2xx(ctx, p.client, http.MethodGet, u, nil, nil)
	if err != nil {
		return nil, err
	}
	m, err := parseGammaMarket(b)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (p *Polymarket) GetPrices(ctx context.Context, marketID string) (yes, no decimal.Decimal, err error) {
	m, err := p.GetMarket(ctx, marketID)
	if err != nil {
		return decimal.Zero, decimal.Zero, err
	}
	return m.YesPrice, m.NoPrice, nil
}

func (p *Polymarket) GetOrderbook(ctx context.Context, marketID string) (platforms.Orderbook, error) {
	m, err := p.GetMarket(ctx, marketID)
	if err != nil {
		return platforms.Orderbook{}, err
	}
	if m.YesTokenID == "" {
		return platforms.Orderbook{}, fmt.Errorf("polymarket: missing yes clob token for market %s", marketID)
	}
	u := fmt.Sprintf("%s/book?token_id=%s", p.clob, url.QueryEscape(m.YesTokenID))
	b, err := doJSONExpect2xx(ctx, p.client, http.MethodGet, u, nil, nil)
	if err != nil {
		return platforms.Orderbook{}, err
	}
	return parseCLOBBook(b)
}

func (p *Polymarket) SendOrder(ctx context.Context, _ platforms.PlaceOrderRequest) (*platforms.OrderResult, error) {
	_ = ctx
	return nil, platforms.ErrTradingNotImplemented
}

func (p *Polymarket) CancelOrder(ctx context.Context, _ string) error {
	_ = ctx
	return platforms.ErrTradingNotImplemented
}

func (p *Polymarket) ListOrders(ctx context.Context, _ *string) ([]platforms.Order, error) {
	_ = ctx
	return nil, platforms.ErrTradingNotImplemented
}

func (p *Polymarket) GetPositions(ctx context.Context) ([]platforms.Position, error) {
	_ = ctx
	return nil, platforms.ErrTradingNotImplemented
}

func (p *Polymarket) GetBalance(ctx context.Context) (map[string]decimal.Decimal, error) {
	_ = ctx
	return nil, platforms.ErrTradingNotImplemented
}

func parseClobTokenIDs(raw json.RawMessage) (yesTok, noTok string) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", ""
	}
	var ids []string
	if err := json.Unmarshal(raw, &ids); err == nil {
		if len(ids) > 0 {
			yesTok = ids[0]
		}
		if len(ids) > 1 {
			noTok = ids[1]
		}
		return yesTok, noTok
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil && asString != "" {
		_ = json.Unmarshal([]byte(asString), &ids)
		if len(ids) > 0 {
			yesTok = ids[0]
		}
		if len(ids) > 1 {
			noTok = ids[1]
		}
	}
	return yesTok, noTok
}

func parseGammaMarket(raw []byte) (platforms.Market, error) {
	var gm struct {
		ID            string          `json:"id"`
		Slug          string          `json:"slug"`
		Question      string          `json:"question"`
		Title         string          `json:"title"`
		Description   string          `json:"description"`
		Category      string          `json:"category"`
		Outcomes      json.RawMessage `json:"outcomes"`
		YesPrice      json.RawMessage `json:"yesPrice"`
		NoPrice       json.RawMessage `json:"noPrice"`
		ClobTokenIds  json.RawMessage `json:"clobTokenIds"`
		Volume24hr    json.RawMessage `json:"volume24hr"`
		Liquidity     json.RawMessage `json:"liquidity"`
		EndDate       string          `json:"endDate"`
		Active        bool            `json:"active"`
		Closed        bool            `json:"closed"`
	}
	if err := json.Unmarshal(raw, &gm); err != nil {
		return platforms.Market{}, err
	}
	yes := decimal.NewFromFloat(0.5)
	no := decimal.NewFromFloat(0.5)

	var outcomes []struct {
		Price interface{} `json:"price"`
	}
	if len(gm.Outcomes) > 0 && string(gm.Outcomes) != "null" {
		_ = json.Unmarshal(gm.Outcomes, &outcomes)
	}
	if len(outcomes) >= 2 {
		yes = decFromFraction100(outcomes[0].Price)
		no = decFromFraction100(outcomes[1].Price)
	} else {
		if len(gm.YesPrice) > 0 && string(gm.YesPrice) != "null" {
			yes = decFromFlexible(gm.YesPrice)
		}
		if len(gm.NoPrice) > 0 && string(gm.NoPrice) != "null" {
			no = decFromFlexible(gm.NoPrice)
		}
	}

	q := gm.Question
	if q == "" {
		q = gm.Title
	}
	var exp *time.Time
	if gm.EndDate != "" {
		if t, err := time.Parse(time.RFC3339, strings.ReplaceAll(gm.EndDate, "Z", "+00:00")); err == nil {
			exp = &t
		}
	}
	yesTok, noTok := parseClobTokenIDs(gm.ClobTokenIds)
	vol := decFromFlexible(gm.Volume24hr)
	liq := decFromFlexible(gm.Liquidity)
	active := gm.Active && !gm.Closed

	return platforms.Market{
		ID:          gm.ID,
		Slug:        gm.Slug,
		Question:    q,
		Description: gm.Description,
		Category:    gm.Category,
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

func decFromFraction100(v interface{}) decimal.Decimal {
	switch t := v.(type) {
	case float64:
		return decimal.NewFromFloat(t).Div(decimal.NewFromInt(100))
	case string:
		d, err := decimal.NewFromString(strings.TrimSpace(t))
		if err != nil {
			return decimal.NewFromFloat(0.5)
		}
		return d.Div(decimal.NewFromInt(100))
	default:
		return decimal.NewFromFloat(0.5)
	}
}

func decFromFlexible(raw json.RawMessage) decimal.Decimal {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" {
		return decimal.Zero
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err == nil {
		return decimal.NewFromFloat(f)
	}
	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		d, err := decimal.NewFromString(str)
		if err != nil {
			return decimal.Zero
		}
		return d
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero
	}
	return d
}

func parseCLOBBook(raw []byte) (platforms.Orderbook, error) {
	var ob struct {
		Bids []struct {
			Price string `json:"price"`
			Size  string `json:"size"`
		} `json:"bids"`
		Asks []struct {
			Price string `json:"price"`
			Size  string `json:"size"`
		} `json:"asks"`
	}
	if err := json.Unmarshal(raw, &ob); err != nil {
		return platforms.Orderbook{}, err
	}
	out := platforms.Orderbook{}
	for _, b := range ob.Bids {
		p, e1 := decimal.NewFromString(b.Price)
		sz, e2 := decimal.NewFromString(b.Size)
		if e1 != nil || e2 != nil {
			continue
		}
		out.Bids = append(out.Bids, platforms.BookLevel{Price: p, Size: sz})
	}
	for _, a := range ob.Asks {
		p, e1 := decimal.NewFromString(a.Price)
		sz, e2 := decimal.NewFromString(a.Size)
		if e1 != nil || e2 != nil {
			continue
		}
		out.Asks = append(out.Asks, platforms.BookLevel{Price: p, Size: sz})
	}
	return out, nil
}
