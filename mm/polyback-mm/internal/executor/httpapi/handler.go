package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/config"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/domain"
	execevents "github.com/profitlock/PredictOS/mm/polyback-mm/internal/executor/events"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/executor/ports"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/hftevents"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/platforms/polymarket/api"
	polyws "github.com/profitlock/PredictOS/mm/polyback-mm/internal/platforms/polymarket/ws"
	"github.com/shopspring/decimal"
)

// Polymarket is the /api/polymarket HTTP adapter (SRP: HTTP only; delegates to ports).
type Polymarket struct {
	cfg     *config.Root
	sim     ports.OrderSimulator
	events  hftevents.Publisher
	feed    polyws.MarketFeed
	metrics *OrderMetrics
	mode    domain.TradingMode
	notify  *orderNotifier
}

// NewPolymarket wires executor HTTP handlers. feed may be nil only in tests; production should pass a real MarketFeed.
func NewPolymarket(cfg *config.Root, sim ports.OrderSimulator, events hftevents.Publisher, feed polyws.MarketFeed, metrics *OrderMetrics) *Polymarket {
	if feed == nil {
		feed = polyws.NoopMarketFeed{}
	}
	if metrics == nil {
		metrics = NewOrderMetrics()
	}
	mode := domain.TradingModeFromConfig(cfg.Hft.Mode)
	return &Polymarket{
		cfg: cfg, sim: sim, events: events, feed: feed, metrics: metrics, mode: mode,
		notify: &orderNotifier{pub: events, mode: mode},
	}
}

// RegisterRoutes mounts REST routes on r.
func (h *Polymarket) RegisterRoutes(r chi.Router) {
	r.Route("/api/polymarket", func(r chi.Router) {
		r.Get("/health", h.health)
		r.Get("/account", h.account)
		r.Get("/bankroll", h.bankroll)
		r.Get("/positions", h.positions)
		r.Get("/tick-size/{tokenId}", h.tickSize)
		r.Get("/marketdata/top/{tokenId}", h.marketTop)
		r.Post("/orders/limit", h.placeLimit)
		r.Post("/orders/market", h.placeMarket)
		r.Delete("/orders/{orderId}", h.cancelOrder)
		r.Get("/orders/{orderId}", h.getOrder)
		r.Get("/orders", h.notImplemented("list orders"))
		r.Get("/trades", h.notImplemented("trades"))
	})
}

func (h *Polymarket) health(w http.ResponseWriter, req *http.Request) {
	deep := req.URL.Query().Get("deep") == "true"
	tokenID := req.URL.Query().Get("tokenId")
	resp := api.PolymarketHealthResponse{
		Mode:            string(h.mode),
		ClobRestURL:     h.cfg.Hft.Polymarket.ClobRestURL,
		ClobWsURL:       h.cfg.Hft.Polymarket.ClobWsURL,
		ChainID:         h.cfg.Hft.Polymarket.ChainID,
		MarketWsEnabled: h.cfg.Hft.Polymarket.MarketWsEnabled,
		Deep:            deep,
		TokenID:         tokenID,
	}
	if deep && tokenID != "" {
		if tob, ok := h.feed.GetTopOfBook(tokenID); ok {
			b, _ := json.Marshal(tob)
			resp.OrderBook = b
		} else {
			resp.DeepError = "no top of book"
		}
	}
	writeJSON(w, resp)
}

func (h *Polymarket) account(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, api.PolymarketAccountResponse{Mode: string(h.mode)})
}

func (h *Polymarket) bankroll(w http.ResponseWriter, _ *http.Request) {
	br := h.cfg.Hft.Strategy.Gabagool.BankrollUsd
	writeJSON(w, api.PolymarketBankrollResponse{
		Mode:                     string(h.mode),
		MakerAddress:             h.cfg.Executor.Sim.ProxyAddress,
		USDCBalance:              decimal.NewFromFloat(br),
		PositionsCurrentValueUsd: decimal.Zero,
		PositionsInitialValueUsd: decimal.Zero,
		TotalEquityUsd:           decimal.NewFromFloat(br),
		AsOfMillis:               time.Now().UnixMilli(),
	})
}

func (h *Polymarket) positions(w http.ResponseWriter, req *http.Request) {
	limit, _ := strconv.Atoi(req.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(req.URL.Query().Get("offset"))
	if h.sim != nil && h.sim.Enabled() {
		writeJSON(w, h.sim.GetPositions(limit, offset))
		return
	}
	http.Error(w, "live positions not implemented in Go port yet", http.StatusNotImplemented)
}

func (h *Polymarket) tickSize(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, decimal.NewFromFloat(0.01))
}

func (h *Polymarket) marketTop(w http.ResponseWriter, req *http.Request) {
	tid := chi.URLParam(req, "tokenId")
	tob, ok := h.feed.GetTopOfBook(tid)
	if !ok {
		http.NotFound(w, req)
		return
	}
	writeJSON(w, tob)
}

func (h *Polymarket) placeLimit(w http.ResponseWriter, req *http.Request) {
	var body api.LimitOrderRequest
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	h.metrics.Placed.Inc()
	if h.sim != nil && h.sim.Enabled() {
		res := h.sim.PlaceLimitOrder(&body)
		h.notify.limit(&body, res, nil)
		writeJSON(w, res)
		return
	}
	h.notify.limit(&body, nil, errLive("live limit orders not wired"))
	http.Error(w, "live limit orders not implemented in Go port yet", http.StatusNotImplemented)
}

func (h *Polymarket) placeMarket(w http.ResponseWriter, req *http.Request) {
	var body api.MarketOrderRequest
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	h.metrics.Placed.Inc()
	if h.sim != nil && h.sim.Enabled() {
		res := h.sim.PlaceMarketOrder(&body)
		h.notify.market(&body, res, nil)
		writeJSON(w, res)
		return
	}
	h.notify.market(&body, nil, errLive("live market orders not wired"))
	http.Error(w, "live market orders not implemented in Go port yet", http.StatusNotImplemented)
}

func (h *Polymarket) cancelOrder(w http.ResponseWriter, req *http.Request) {
	id := chi.URLParam(req, "orderId")
	if h.sim != nil && h.sim.Enabled() {
		raw := h.sim.CancelOrder(id)
		h.notify.cancel(id, raw, nil)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(raw)
		return
	}
	h.notify.cancel(id, nil, errLive("live cancel not wired"))
	http.Error(w, "live cancel not implemented in Go port yet", http.StatusNotImplemented)
}

func (h *Polymarket) getOrder(w http.ResponseWriter, req *http.Request) {
	id := chi.URLParam(req, "orderId")
	if h.sim != nil && h.sim.Enabled() {
		raw := h.sim.GetOrder(id)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(raw)
		return
	}
	http.Error(w, "live get order not implemented", http.StatusNotImplemented)
}

func (h *Polymarket) notImplemented(feature string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, feature+" not implemented in Go port yet", http.StatusNotImplemented)
	}
}

// orderNotifier maps domain events to Kafka (SRP: notification only).
type orderNotifier struct {
	pub  hftevents.Publisher
	mode domain.TradingMode
}

func (n *orderNotifier) limit(req *api.LimitOrderRequest, res *api.OrderSubmissionResult, err error) {
	if n.pub == nil || !n.pub.Enabled() {
		return
	}
	ev := execevents.ExecutorLimitOrder{
		TokenID: req.TokenID, Side: req.Side, Price: req.Price, Size: req.Size,
		Mode: string(n.mode),
	}
	if res != nil {
		ev.OrderID = orderIDFromResult(res)
	}
	if err != nil {
		ev.Error = err.Error()
	}
	n.pub.Publish(hftevents.ExecutorOrderLimit, req.TokenID, ev)
}

func (n *orderNotifier) market(req *api.MarketOrderRequest, res *api.OrderSubmissionResult, err error) {
	if n.pub == nil || !n.pub.Enabled() {
		return
	}
	m := map[string]any{
		"tokenId": req.TokenID, "side": req.Side, "amount": req.Amount.String(), "price": req.Price.String(),
		"mode": string(n.mode), "error": nil,
	}
	if res != nil {
		m["orderId"] = orderIDFromResult(res)
	}
	if err != nil {
		m["error"] = err.Error()
	}
	n.pub.Publish(hftevents.ExecutorOrderMarket, req.TokenID, m)
}

func (n *orderNotifier) cancel(id string, raw json.RawMessage, err error) {
	if n.pub == nil || !n.pub.Enabled() {
		return
	}
	m := map[string]any{"orderId": id, "result": json.RawMessage(nil), "error": nil}
	if len(raw) > 0 {
		m["result"] = json.RawMessage(raw)
	}
	if err != nil {
		m["error"] = err.Error()
	}
	n.pub.Publish(hftevents.ExecutorOrderCancel, id, m)
}

func orderIDFromResult(res *api.OrderSubmissionResult) string {
	if res == nil || len(res.ClobResponse) == 0 {
		return ""
	}
	var m map[string]any
	if json.Unmarshal(res.ClobResponse, &m) != nil {
		return ""
	}
	if v, ok := m["orderID"].(string); ok {
		return v
	}
	if v, ok := m["orderId"].(string); ok {
		return v
	}
	return ""
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

type liveErr string

func (e liveErr) Error() string { return string(e) }

func errLive(msg string) error { return liveErr(msg) }

// DiscardBody drains the request body for connection reuse.
func DiscardBody(r *http.Request) {
	if r.Body != nil {
		_, _ = io.Copy(io.Discard, r.Body)
		_ = r.Body.Close()
	}
}
