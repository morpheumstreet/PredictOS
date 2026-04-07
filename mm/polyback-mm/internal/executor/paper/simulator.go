package paper

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/config"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/executor/ports"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/domain"
	execevents "github.com/profitlock/PredictOS/mm/polyback-mm/internal/executor/events"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/hftevents"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/platforms/polymarket/api"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/platforms/polymarket/gamma"
	polyws "github.com/profitlock/PredictOS/mm/polyback-mm/internal/platforms/polymarket/ws"
	"github.com/shopspring/decimal"
)

const userTradeEventType = "polymarket.user.trade"

type Simulator struct {
	mode   domain.TradingMode
	hft    *config.Root
	sim    config.SimCfg
	events hftevents.Publisher
	feed   polyws.MarketFeed
	gamma  *gamma.Client

	mu              sync.Mutex
	orders          map[string]*simOrder
	positions       map[string]*position
	metaByToken     map[string]*tokenMeta
	fillStop        chan struct{}
	fillWG          sync.WaitGroup
}

type simOrder struct {
	OrderID        string
	TokenID        string
	Side           domain.OrderSide
	RequestedPrice *decimal.Decimal
	RequestedSize  *decimal.Decimal
	CreatedAt      time.Time
	Status         string
	MatchedSize    *decimal.Decimal
	RemainingSize  *decimal.Decimal

	lastPubStatus   string
	lastPubMatched  *decimal.Decimal
	lastPubRemaining *decimal.Decimal
}

type position struct {
	Shares  decimal.Decimal
	CostUSD decimal.Decimal
}

func (p *position) AvgPrice() *decimal.Decimal {
	if p.Shares.IsZero() {
		return nil
	}
	d := p.CostUSD.Div(p.Shares).Round(6)
	return &d
}

type tokenMeta struct {
	MarketSlug    string
	Title         string
	ConditionID   string
	Outcome       string
	OutcomeIndex  int
}

func NewSimulator(r *config.Root, pub hftevents.Publisher, feed polyws.MarketFeed, g *gamma.Client) *Simulator {
	if feed == nil {
		feed = polyws.NoopMarketFeed{}
	}
	mode := domain.ModePaper
	if strings.EqualFold(r.Hft.Mode, "LIVE") {
		mode = domain.ModeLive
	}
	s := &Simulator{
		mode:      mode,
		hft:       r,
		sim:       r.Executor.Sim,
		events:    pub,
		feed:      feed,
		gamma:     g,
		orders:    make(map[string]*simOrder),
		positions: make(map[string]*position),
		metaByToken: make(map[string]*tokenMeta),
		fillStop:  make(chan struct{}),
	}
	if s.Enabled() && s.sim.FillsEnabled && s.sim.FillPollMillis > 0 {
		s.fillWG.Add(1)
		go s.fillLoop()
	}
	return s
}

func (s *Simulator) Enabled() bool {
	return s.sim.Enabled
}

func (s *Simulator) Close() {
	close(s.fillStop)
	s.fillWG.Wait()
}

func randomSimID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "sim-" + hex.EncodeToString(b)
}

func (s *Simulator) PlaceLimitOrder(req *api.LimitOrderRequest) *api.OrderSubmissionResult {
	s.feed.SubscribeAssets([]string{req.TokenID})
	id := randomSimID()
	size := req.Size
	if size.IsNegative() {
		size = decimal.Zero
	}
	rem := size
	m0 := decimal.Zero
	o := &simOrder{
		OrderID:        id,
		TokenID:        req.TokenID,
		Side:           req.Side,
		RequestedPrice: &req.Price,
		RequestedSize:  &size,
		CreatedAt:      time.Now().UTC(),
		Status:         "OPEN",
		MatchedSize:    &m0,
		RemainingSize:  &rem,
	}
	s.mu.Lock()
	s.orders[id] = o
	s.mu.Unlock()
	s.publishOrderStatus(o, "")
	resp, _ := json.Marshal(map[string]any{
		"mode":    "SIM",
		"orderID": id,
		"orderId": id,
		"status":  "OPEN",
	})
	return &api.OrderSubmissionResult{Mode: s.mode, ClobResponse: resp}
}

func (s *Simulator) PlaceMarketOrder(req *api.MarketOrderRequest) *api.OrderSubmissionResult {
	s.feed.SubscribeAssets([]string{req.TokenID})
	id := randomSimID()
	tob, ok := s.feed.GetTopOfBook(req.TokenID)
	if !ok || tob == nil || tob.BestBid == nil || tob.BestAsk == nil {
		resp, _ := json.Marshal(map[string]any{"mode": "SIM", "orderID": id, "orderId": id, "status": "REJECTED", "reason": "no_tob"})
		return &api.OrderSubmissionResult{Mode: s.mode, ClobResponse: resp}
	}
	limitPrice := req.Price
	if limitPrice.IsZero() {
		limitPrice = decimal.NewFromInt(1)
	}
	if req.Side == domain.SideBuy {
		bestAsk := *tob.BestAsk
		if bestAsk.GreaterThan(limitPrice) {
			resp, _ := json.Marshal(map[string]any{"mode": "SIM", "orderID": id, "orderId": id, "status": "REJECTED", "reason": "ask_above_limit"})
			return &api.OrderSubmissionResult{Mode: s.mode, ClobResponse: resp}
		}
		notional := req.Amount
		if !notional.IsPositive() {
			resp, _ := json.Marshal(map[string]any{"mode": "SIM", "orderID": id, "orderId": id, "status": "REJECTED", "reason": "amount_invalid"})
			return &api.OrderSubmissionResult{Mode: s.mode, ClobResponse: resp}
		}
		shares := notional.Div(bestAsk).RoundDown(2)
		if shares.LessThan(decimal.NewFromFloat(0.01)) {
			resp, _ := json.Marshal(map[string]any{"mode": "SIM", "orderID": id, "orderId": id, "status": "REJECTED", "reason": "shares_too_small"})
			return &api.OrderSubmissionResult{Mode: s.mode, ClobResponse: resp}
		}
		s.applyBuyFill(id, req.TokenID, bestAsk, shares)
		resp, _ := json.Marshal(map[string]any{"mode": "SIM", "orderID": id, "orderId": id, "status": "FILLED"})
		return &api.OrderSubmissionResult{Mode: s.mode, ClobResponse: resp}
	}
	if req.Side == domain.SideSell {
		bestBid := *tob.BestBid
		if bestBid.LessThan(limitPrice) {
			resp, _ := json.Marshal(map[string]any{"mode": "SIM", "orderID": id, "orderId": id, "status": "REJECTED", "reason": "bid_below_limit"})
			return &api.OrderSubmissionResult{Mode: s.mode, ClobResponse: resp}
		}
		shares := req.Amount
		if shares.LessThan(decimal.NewFromFloat(0.01)) {
			resp, _ := json.Marshal(map[string]any{"mode": "SIM", "orderID": id, "orderId": id, "status": "REJECTED", "reason": "amount_invalid"})
			return &api.OrderSubmissionResult{Mode: s.mode, ClobResponse: resp}
		}
		s.applySellFill(id, req.TokenID, bestBid, shares)
		resp, _ := json.Marshal(map[string]any{"mode": "SIM", "orderID": id, "orderId": id, "status": "FILLED"})
		return &api.OrderSubmissionResult{Mode: s.mode, ClobResponse: resp}
	}
	resp, _ := json.Marshal(map[string]any{"mode": "SIM", "orderID": id, "orderId": id, "status": "REJECTED", "reason": "unsupported_side"})
	return &api.OrderSubmissionResult{Mode: s.mode, ClobResponse: resp}
}

func (s *Simulator) applyBuyFill(id, token string, price, shares decimal.Decimal) {
	o := &simOrder{
		OrderID: id, TokenID: token, Side: domain.SideBuy, RequestedPrice: &price, RequestedSize: &shares,
		CreatedAt: time.Now().UTC(), Status: "FILLED", MatchedSize: &shares,
	}
	z := decimal.Zero
	o.RemainingSize = &z
	s.mu.Lock()
	s.orders[id] = o
	p := s.positions[token]
	if p == nil {
		p = &position{}
	}
	p.Shares = p.Shares.Add(shares)
	p.CostUSD = p.CostUSD.Add(price.Mul(shares))
	s.positions[token] = p
	s.mu.Unlock()
	s.publishOrderStatus(o, "")
	s.publishUserTrade(o, shares, price, "TAKER")
}

func (s *Simulator) applySellFill(id, token string, price, shares decimal.Decimal) {
	o := &simOrder{
		OrderID: id, TokenID: token, Side: domain.SideSell, RequestedPrice: &price, RequestedSize: &shares,
		CreatedAt: time.Now().UTC(), Status: "FILLED", MatchedSize: &shares,
	}
	z := decimal.Zero
	o.RemainingSize = &z
	s.mu.Lock()
	s.orders[id] = o
	p := s.positions[token]
	if p == nil {
		p = &position{}
	}
	p.Shares = p.Shares.Sub(shares)
	p.CostUSD = p.CostUSD.Sub(price.Mul(shares))
	s.positions[token] = p
	s.mu.Unlock()
	s.publishOrderStatus(o, "")
	s.publishUserTrade(o, shares, price, "TAKER")
}

func (s *Simulator) CancelOrder(orderID string) json.RawMessage {
	if strings.TrimSpace(orderID) == "" {
		b, _ := json.Marshal(map[string]any{"canceled": false})
		return b
	}
	s.mu.Lock()
	o, ok := s.orders[orderID]
	if !ok {
		s.mu.Unlock()
		b, _ := json.Marshal(map[string]any{"mode": "SIM", "canceled": false, "orderId": orderID})
		return b
	}
	if terminalStatus(o.Status) {
		s.mu.Unlock()
		b, _ := json.Marshal(map[string]any{"mode": "SIM", "canceled": false, "orderId": orderID, "status": o.Status})
		return b
	}
	o.Status = "CANCELED"
	s.mu.Unlock()
	s.publishOrderStatus(o, "")
	b, _ := json.Marshal(map[string]any{"mode": "SIM", "canceled": true, "orderId": orderID, "status": "CANCELED"})
	return b
}

func (s *Simulator) GetOrder(orderID string) json.RawMessage {
	if strings.TrimSpace(orderID) == "" {
		b, _ := json.Marshal(map[string]any{"error": "orderId blank"})
		return b
	}
	s.mu.Lock()
	o, ok := s.orders[orderID]
	s.mu.Unlock()
	if !ok {
		b, _ := json.Marshal(map[string]any{"mode": "SIM", "orderId": orderID, "status": "UNKNOWN"})
		return b
	}
	m := map[string]any{
		"mode":    "SIM",
		"orderId": o.OrderID,
		"tokenId": o.TokenID,
		"side":    o.Side,
		"status":  o.Status,
	}
	if o.MatchedSize != nil {
		f, _ := o.MatchedSize.Float64()
		m["matched_size"] = f
	}
	if o.RemainingSize != nil {
		f, _ := o.RemainingSize.Float64()
		m["remaining_size"] = f
	}
	if o.RequestedPrice != nil {
		f, _ := o.RequestedPrice.Float64()
		m["requestedPrice"] = f
	}
	if o.RequestedSize != nil {
		f, _ := o.RequestedSize.Float64()
		m["requestedSize"] = f
	}
	b, _ := json.Marshal(m)
	return b
}

func (s *Simulator) GetPositions(limit, offset int) []api.PolymarketPosition {
	if limit <= 0 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	s.mu.Lock()
	keys := make([]string, 0, len(s.positions))
	for k := range s.positions {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	s.mu.Unlock()

	from := offset
	if from > len(keys) {
		from = len(keys)
	}
	to := from + limit
	if to > len(keys) {
		to = len(keys)
	}
	out := make([]api.PolymarketPosition, 0)
	proxy := s.sim.ProxyAddress
	for i := from; i < to; i++ {
		tid := keys[i]
		s.mu.Lock()
		p := s.positions[tid]
		s.mu.Unlock()
		if p == nil || p.Shares.IsZero() {
			continue
		}
		meta, _ := s.resolveTokenMeta(tid)
		var title, slug, cond, outc string
		var oi *int
		if meta != nil {
			title, slug, cond, outc = meta.Title, meta.MarketSlug, meta.ConditionID, meta.Outcome
			oi = &meta.OutcomeIndex
		}
		cp := s.bestEffortCurPrice(tid)
		out = append(out, api.PolymarketPosition{
			ProxyWallet: proxy, Asset: tid, ConditionID: cond,
			Size: p.Shares, AvgPrice: derefDec(p.AvgPrice()), InitialValue: p.CostUSD,
			Title: title, Slug: slug, Outcome: outc, OutcomeIndex: oi, CurPrice: derefDec(cp),
		})
	}
	return out
}

func derefDec(d *decimal.Decimal) decimal.Decimal {
	if d == nil {
		return decimal.Zero
	}
	return *d
}

func terminalStatus(st string) bool {
	u := strings.ToUpper(strings.TrimSpace(st))
	return strings.Contains(u, "FILLED") || strings.Contains(u, "CANCELED") || strings.Contains(u, "CANCELLED") ||
		strings.Contains(u, "EXPIRED") || strings.Contains(u, "REJECTED") || strings.Contains(u, "FAILED") ||
		strings.Contains(u, "DONE") || strings.Contains(u, "CLOSED")
}

func (s *Simulator) publishOrderStatus(o *simOrder, errStr string) {
	if s.events == nil || !s.events.Enabled() || o == nil {
		return
	}
	st := o.Status
	mat := decimal.Zero
	rem := decimal.Zero
	if o.MatchedSize != nil {
		mat = *o.MatchedSize
	}
	if o.RemainingSize != nil {
		rem = *o.RemainingSize
	}
	changed := o.lastPubStatus != st || !decEqPtr(o.lastPubMatched, mat) || !decEqPtr(o.lastPubRemaining, rem) || errStr != ""
	if !changed {
		return
	}
	o.lastPubStatus = st
	o.lastPubMatched = new(decimal.Decimal)
	*o.lastPubMatched = mat
	o.lastPubRemaining = new(decimal.Decimal)
	*o.lastPubRemaining = rem

	orderJSON := string(s.GetOrder(o.OrderID))
	ev := execevents.ExecutorOrderStatus{
		OrderID: o.OrderID, TokenID: o.TokenID, Side: o.Side,
		RequestedPrice: o.RequestedPrice, RequestedSize: o.RequestedSize,
		Status: st, Matched: &mat, Remaining: &rem, OrderJSON: orderJSON, Error: errStr,
	}
	s.events.Publish(hftevents.ExecutorOrderStatus, o.OrderID, ev)
}

func decEqPtr(p *decimal.Decimal, v decimal.Decimal) bool {
	if p == nil {
		return v.IsZero()
	}
	return p.Equal(v)
}

func (s *Simulator) publishUserTrade(o *simOrder, fillSize, fillPrice decimal.Decimal, kind string) {
	if s.events == nil || !s.events.Enabled() {
		return
	}
	meta, _ := s.resolveTokenMeta(o.TokenID)
	trade := map[string]any{
		"asset": o.TokenID, "side": o.Side, "price": mustFloat(fillPrice), "size": mustFloat(fillSize),
		"timestamp": time.Now().Unix(), "transactionHash": "", "simKind": kind,
	}
	if meta != nil {
		trade["slug"] = meta.MarketSlug
		trade["title"] = meta.Title
		trade["conditionId"] = meta.ConditionID
		trade["outcome"] = meta.Outcome
		trade["outcomeIndex"] = meta.OutcomeIndex
	}
	data := map[string]any{
		"username":      s.sim.Username,
		"proxyAddress":  s.sim.ProxyAddress,
		"trade":         trade,
	}
	s.events.PublishAt(time.Unix(time.Now().Unix(), 0).UTC(), userTradeEventType, "simtrade:"+o.OrderID+":"+randomSimID(), data)
}

func mustFloat(d decimal.Decimal) float64 {
	f, _ := d.Float64()
	return f
}

func (s *Simulator) resolveTokenMeta(tokenID string) (*tokenMeta, bool) {
	tokenID = strings.TrimSpace(tokenID)
	if tokenID == "" {
		return nil, false
	}
	s.mu.Lock()
	cached := s.metaByToken[tokenID]
	s.mu.Unlock()
	if cached != nil {
		return cached, true
	}
	raw, err := s.gamma.Markets(map[string]string{"clob_token_ids": tokenID, "limit": "1"})
	if err != nil || len(raw) == 0 {
		return nil, false
	}
	var arr []map[string]any
	if json.Unmarshal(raw, &arr) != nil || len(arr) == 0 {
		return nil, false
	}
	m := arr[0]
	slug := str(m["slug"])
	title := str(m["question"])
	cond := str(m["conditionId"])
	if title == "" {
		title = slug
	}
	clobRaw := str(m["clobTokenIds"])
	outRaw := str(m["outcomes"])
	outcome := ""
	oi := -1
	if clobRaw != "" && outRaw != "" {
		var tids []string
		var outs []string
		_ = json.Unmarshal([]byte(clobRaw), &tids)
		_ = json.Unmarshal([]byte(outRaw), &outs)
		for i, tid := range tids {
			if strings.TrimSpace(tid) == tokenID && i < len(outs) {
				oi = i
				outcome = outs[i]
				break
			}
		}
	}
	meta := &tokenMeta{MarketSlug: slug, Title: title, ConditionID: cond, Outcome: outcome, OutcomeIndex: oi}
	s.mu.Lock()
	s.metaByToken[tokenID] = meta
	s.mu.Unlock()
	return meta, true
}

func str(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	default:
		b, _ := json.Marshal(t)
		return strings.Trim(string(b), `"`)
	}
}

func (s *Simulator) bestEffortCurPrice(tokenID string) *decimal.Decimal {
	tob, ok := s.feed.GetTopOfBook(tokenID)
	if !ok || tob == nil || tob.BestBid == nil || tob.BestAsk == nil {
		return nil
	}
	mid := tob.BestBid.Add(*tob.BestAsk).Div(decimal.NewFromInt(2)).Round(6)
	return &mid
}

func (s *Simulator) fillLoop() {
	defer s.fillWG.Done()
	t := time.NewTicker(time.Duration(s.sim.FillPollMillis) * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-s.fillStop:
			return
		case <-t.C:
			if !s.Enabled() || !s.sim.FillsEnabled {
				continue
			}
			s.mu.Lock()
			ids := make([]string, 0, len(s.orders))
			for id := range s.orders {
				ids = append(ids, id)
			}
			s.mu.Unlock()
			for _, id := range ids {
				s.simulateOne(id)
			}
		}
	}
}

func (s *Simulator) simulateOne(id string) {
	s.mu.Lock()
	o, ok := s.orders[id]
	if !ok || o == nil || o.Side != domain.SideBuy {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()

	tob, ok := s.feed.GetTopOfBook(o.TokenID)
	if !ok || tob == nil || tob.BestBid == nil || tob.BestAsk == nil || tob.UpdatedAt == nil {
		return
	}
	if time.Since(*tob.UpdatedAt) > 2*time.Second {
		return
	}
	price := o.RequestedPrice
	if price == nil {
		return
	}
	bestAsk := *tob.BestAsk
	bestBid := *tob.BestBid
	if bestAsk.LessThanOrEqual(*price) {
		s.mu.Lock()
		o2 := s.orders[id]
		s.mu.Unlock()
		if o2 == nil || o2.RemainingSize == nil {
			return
		}
		s.fill(o2, *o2.RemainingSize, bestAsk, "TAKER")
		return
	}
	if bestBid.GreaterThan(*price) {
		return
	}
	p := s.sim.MakerFillProbabilityPerPoll
	if p <= 0 {
		return
	}
	ticksAbove := 0
	tickSize := decimal.NewFromFloat(0.01)
	diff := price.Sub(bestBid)
	if diff.IsPositive() && tickSize.IsPositive() {
		ticksAbove = int(diff.Div(tickSize).IntPart())
	}
	if ticksAbove > 0 && s.sim.MakerFillProbabilityMultiplierPerTick > 0 && s.sim.MakerFillProbabilityMultiplierPerTick != 1 {
		p *= math.Pow(s.sim.MakerFillProbabilityMultiplierPerTick, float64(ticksAbove))
	}
	maxP := s.sim.MakerFillProbabilityMaxPerPoll
	if maxP > 0 && p > maxP {
		p = maxP
	}
	if randFloat() > p {
		return
	}
	s.mu.Lock()
	o = s.orders[id]
	if o == nil || o.RemainingSize == nil || terminalStatus(o.Status) {
		s.mu.Unlock()
		return
	}
	rem := *o.RemainingSize
	s.mu.Unlock()
	if !rem.IsPositive() {
		return
	}
	frac := decimal.NewFromFloat(s.sim.MakerFillFractionOfRemaining)
	fill := rem.Mul(frac).RoundDown(2)
	min01 := decimal.NewFromFloat(0.01)
	if fill.LessThan(min01) {
		if rem.LessThan(min01) {
			fill = rem
		} else {
			fill = min01
		}
	}
	s.fill(o, fill, *price, "MAKER")
}

func randFloat() float64 {
	var b [8]byte
	_, _ = rand.Read(b[:])
	u := uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
		uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56
	return float64(u%10000) / 10000.0
}

func (s *Simulator) fill(o *simOrder, fillSize, fillPrice decimal.Decimal, kind string) {
	if !fillSize.IsPositive() {
		return
	}
	s.mu.Lock()
	cur, ok := s.orders[o.OrderID]
	if !ok || terminalStatus(cur.Status) {
		s.mu.Unlock()
		return
	}
	rem := *cur.RemainingSize
	if !rem.IsPositive() {
		s.mu.Unlock()
		return
	}
	applied := fillSize
	if rem.LessThan(applied) {
		applied = rem
	}
	applied = applied.RoundDown(2)
	if applied.LessThan(decimal.NewFromFloat(0.01)) {
		s.mu.Unlock()
		return
	}
	prevM := decimal.Zero
	if cur.MatchedSize != nil {
		prevM = *cur.MatchedSize
	}
	matched := prevM.Add(applied)
	rem = rem.Sub(applied)
	if rem.IsNegative() {
		rem = decimal.Zero
	}
	st := "PARTIALLY_FILLED"
	if rem.IsZero() {
		st = "FILLED"
	}
	cur.MatchedSize = &matched
	cur.RemainingSize = &rem
	cur.Status = st
	token := cur.TokenID
	s.mu.Unlock()

	s.mu.Lock()
	p := s.positions[token]
	if p == nil {
		p = &position{}
	}
	p.Shares = p.Shares.Add(applied)
	p.CostUSD = p.CostUSD.Add(fillPrice.Mul(applied))
	s.positions[token] = p
	s.mu.Unlock()

	s.publishOrderStatus(cur, "")
	s.publishUserTrade(cur, applied, fillPrice, kind)
}

var _ ports.OrderSimulator = (*Simulator)(nil)
