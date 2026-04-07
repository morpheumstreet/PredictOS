package kalshidflow

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

const defaultBaseURL = "https://a.prediction-markets-api.dflow.net/api/v1"

// KalshiDFlow lists Kalshi-shaped markets for one event ticker via DFlow (see supabase _shared/dflow).
type KalshiDFlow struct {
	baseURL     string
	apiKey      string
	eventTicker string
	client      *http.Client
}

var _ platforms.Platform = (*KalshiDFlow)(nil)

// NewKalshiDFlow builds a client. apiKey and eventTicker are required for reads.
func NewKalshiDFlow(baseURL, apiKey, eventTicker string) *KalshiDFlow {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultBaseURL
	}
	return &KalshiDFlow{
		baseURL:     strings.TrimSuffix(strings.TrimSpace(baseURL), "/"),
		apiKey:      strings.TrimSpace(apiKey),
		eventTicker: strings.TrimSpace(eventTicker),
		client:      platforms.DefaultHTTPClient(),
	}
}

func (k *KalshiDFlow) Name() string { return "kalshi_dflow" }

func (k *KalshiDFlow) headers() map[string]string {
	return map[string]string{"x-api-key": k.apiKey}
}

func (k *KalshiDFlow) HealthCheck(ctx context.Context) error {
	if k.apiKey == "" || k.eventTicker == "" {
		return platforms.ErrNotConfigured
	}
	_, err := k.GetAllMarkets(ctx)
	return err
}

func (k *KalshiDFlow) GetAllMarkets(ctx context.Context) ([]platforms.Market, error) {
	if k.apiKey == "" || k.eventTicker == "" {
		return nil, platforms.ErrNotConfigured
	}
	u := fmt.Sprintf("%s/event/%s?withNestedMarkets=true", k.baseURL, url.PathEscape(k.eventTicker))
	b, err := platforms.DoJSONExpect2xx(ctx, k.client, http.MethodGet, u, k.headers(), nil)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Markets []struct {
			Ticker      string  `json:"ticker"`
			EventTicker string  `json:"event_ticker"`
			Title       string  `json:"title"`
			Subtitle    string  `json:"subtitle"`
			Status      string  `json:"status"`
			CloseTime   string  `json:"close_time"`
			YesBid      float64 `json:"yes_bid"`
			YesAsk      float64 `json:"yes_ask"`
			NoBid       float64 `json:"no_bid"`
			NoAsk       float64 `json:"no_ask"`
			Volume24h   float64 `json:"volume_24h"`
			Liquidity   float64 `json:"liquidity"`
		} `json:"markets"`
	}
	if err := json.Unmarshal(b, &resp); err != nil {
		return nil, err
	}
	out := make([]platforms.Market, 0, len(resp.Markets))
	for _, m := range resp.Markets {
		raw, _ := json.Marshal(m)
		yesMid := (m.YesBid + m.YesAsk) / 2
		noMid := (m.NoBid + m.NoAsk) / 2
		yesP := decimal.NewFromFloat(yesMid).Div(decimal.NewFromInt(100))
		noP := decimal.NewFromFloat(noMid).Div(decimal.NewFromInt(100))
		desc := m.Subtitle
		var exp *time.Time
		if m.CloseTime != "" {
			if t, err := time.Parse(time.RFC3339, strings.ReplaceAll(m.CloseTime, "Z", "+00:00")); err == nil {
				exp = &t
			}
		}
		out = append(out, platforms.Market{
			ID:          m.Ticker,
			Slug:        m.Ticker,
			Question:    m.Title,
			Description: desc,
			Category:    m.EventTicker,
			YesPrice:    yesP,
			NoPrice:     noP,
			Volume24h:   decimal.NewFromFloat(m.Volume24h),
			Liquidity:   decimal.NewFromFloat(m.Liquidity),
			ExpiresAt:   exp,
			Active:      strings.EqualFold(m.Status, "open") || strings.EqualFold(m.Status, "active"),
			Raw:         raw,
		})
	}
	return out, nil
}

func (k *KalshiDFlow) GetMarket(ctx context.Context, marketID string) (*platforms.Market, error) {
	all, err := k.GetAllMarkets(ctx)
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].ID == marketID || all[i].Slug == marketID {
			return &all[i], nil
		}
	}
	return nil, fmt.Errorf("kalshidflow: market %q not found under event %q", marketID, k.eventTicker)
}

func (k *KalshiDFlow) GetPrices(ctx context.Context, marketID string) (yes, no decimal.Decimal, err error) {
	m, err := k.GetMarket(ctx, marketID)
	if err != nil {
		return decimal.Zero, decimal.Zero, err
	}
	return m.YesPrice, m.NoPrice, nil
}

func (k *KalshiDFlow) GetOrderbook(ctx context.Context, _ string) (platforms.Orderbook, error) {
	_ = ctx
	return platforms.Orderbook{}, fmt.Errorf("kalshidflow: %w", platforms.ErrTradingNotImplemented)
}

func (k *KalshiDFlow) SendOrder(ctx context.Context, _ platforms.PlaceOrderRequest) (*platforms.OrderResult, error) {
	_ = ctx
	return nil, platforms.ErrTradingNotImplemented
}

func (k *KalshiDFlow) CancelOrder(ctx context.Context, _ string) error {
	_ = ctx
	return platforms.ErrTradingNotImplemented
}

func (k *KalshiDFlow) ListOrders(ctx context.Context, _ *string) ([]platforms.Order, error) {
	_ = ctx
	return nil, platforms.ErrTradingNotImplemented
}

func (k *KalshiDFlow) GetPositions(ctx context.Context) ([]platforms.Position, error) {
	_ = ctx
	return nil, platforms.ErrTradingNotImplemented
}

func (k *KalshiDFlow) GetBalance(ctx context.Context) (map[string]decimal.Decimal, error) {
	_ = ctx
	return nil, platforms.ErrTradingNotImplemented
}
