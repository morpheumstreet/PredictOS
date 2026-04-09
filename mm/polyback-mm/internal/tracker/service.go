package tracker

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/config"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/platforms/polymarket/gamma"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/tracker/dataapi"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/tracker/gammaadapter"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/tracker/market"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/tracker/port"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/tracker/position"
)

// Service wires HTTP-facing position tracking (composition root for cmd/tracker and intelligence).
type Service struct {
	data  port.PolymarketData
	gamma port.GammaMarket
}

// NewService builds defaults from config (Gamma + data-api bases, shared HTTP client).
func NewService(root *config.Root, hc *http.Client) *Service {
	if hc == nil {
		hc = &http.Client{Timeout: 60 * time.Second}
	}
	gc := gamma.NewWithHTTP(gammaBase(root), hc)
	return NewServiceWith(dataapi.New(dataAPIBase(root), hc), &gammaadapter.FromClient{Inner: gc})
}

// NewServiceWith wires concrete ports (use in tests with fakes / mocks).
func NewServiceWith(data port.PolymarketData, gamma port.GammaMarket) *Service {
	return &Service{data: data, gamma: gamma}
}

func gammaBase(r *config.Root) string {
	b := strings.TrimSpace(r.Hft.Polymarket.GammaURL)
	if b == "" {
		return "https://gamma-api.polymarket.com"
	}
	return strings.TrimSuffix(b, "/")
}

func dataAPIBase(r *config.Root) string {
	b := strings.TrimSpace(r.Ingestor.Polymarket.DataAPIBaseURL)
	if b == "" {
		return "https://data-api.polymarket.com"
	}
	return strings.TrimSuffix(b, "/")
}

// PolymarketPositionTracker implements POST polymarket-position-tracker (terminal + intelligence contract).
func (s *Service) PolymarketPositionTracker(ctx context.Context, body []byte) (int, map[string]any) {
	var req map[string]any
	if err := json.Unmarshal(body, &req); err != nil {
		return http.StatusBadRequest, errJSON("Invalid JSON")
	}
	asset, _ := req["asset"].(string)
	asset = strings.ToUpper(strings.TrimSpace(asset))
	if _, ok := Asset15mSlugPrefix[asset]; !ok {
		return http.StatusOK, errJSON("Invalid asset")
	}
	wallets, err := parseWalletAddresses(req)
	if err != nil {
		return http.StatusBadRequest, errJSON(err.Error())
	}

	ts := time.Now().Unix()
	block := int(math.Floor(float64(ts)/900) * 900)
	slug, _ := req["marketSlug"].(string)
	slug = strings.TrimSpace(slug)
	if slug == "" {
		slug = Asset15mSlugPrefix[asset] + fmt.Sprintf("%d", block)
	}

	var upTok, downTok string
	if tid, ok := req["tokenIds"].(map[string]any); ok {
		if u, ok := tid["up"].(string); ok {
			upTok = strings.TrimSpace(u)
		}
		if d, ok := tid["down"].(string); ok {
			downTok = strings.TrimSpace(d)
		}
	}

	m, err := market.ResolveUpDown(s.gamma, market.ResolveRequest{
		Slug:         slug,
		OverrideUp:   upTok,
		OverrideDown: downTok,
	})
	if err != nil {
		st, body := statusFromMarketErr(err)
		return st, body
	}

	if len(wallets) == 1 {
		pos, perr := position.BuildSnapshot(ctx, s.data, wallets[0], m)
		if perr != nil {
			return http.StatusBadGateway, errJSON(perr.Error())
		}
		return http.StatusOK, okJSON(map[string]any{
			"asset":         asset,
			"walletAddress": wallets[0],
			"position":      pos,
		})
	}

	var rows []map[string]any
	for _, w := range wallets {
		pos, perr := position.BuildSnapshot(ctx, s.data, w, m)
		if perr != nil {
			rows = append(rows, map[string]any{"address": w, "success": false, "error": perr.Error()})
			continue
		}
		rows = append(rows, map[string]any{"address": w, "success": true, "position": pos})
	}
	return http.StatusOK, okJSON(map[string]any{"asset": asset, "wallets": rows})
}
