package gabagool

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"sync"
	"time"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/config"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/strategy/executorclient"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/strategy/metrics"
	polyws "github.com/profitlock/PredictOS/mm/polyback-mm/internal/polymarket/ws"
	"github.com/shopspring/decimal"
)

type tickEnt struct {
	until time.Time
	size  decimal.Decimal
}

type Engine struct {
	root     *config.Root
	feed     polyws.MarketFeed
	exec     *executorclient.Client
	disc     *Discovery
	bank     *Bankroll
	pos      *PositionTracker
	qc       *QuoteCalculator
	om       *OrderManager
	mu           sync.RWMutex
	active       []Market
	tickSize map[string]tickEnt
	stop     chan struct{}
	wg       sync.WaitGroup
}

func NewEngine(root *config.Root, feed polyws.MarketFeed, ex *executorclient.Client, disc *Discovery, met *metrics.Service, om *OrderManager) *Engine {
	if feed == nil {
		feed = polyws.NoopMarketFeed{}
	}
	b := NewBankroll(root, ex, met)
	p := NewPositionTracker(ex)
	qc := NewQuoteCalculator(b, root, met)
	return &Engine{
		root: root, feed: feed, exec: ex, disc: disc, bank: b, pos: p, qc: qc, om: om,
		tickSize: map[string]tickEnt{}, stop: make(chan struct{}),
	}
}

func RandomRunID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (e *Engine) Start() {
	g := &e.root.Hft.Strategy.Gabagool
	if !g.Enabled || !e.root.Hft.Polymarket.MarketWsEnabled {
		log.Printf("gabagool engine: disabled (enabled=%v market_ws=%v)", g.Enabled, e.root.Hft.Polymarket.MarketWsEnabled)
		return
	}
	period := time.Duration(max64(100, g.RefreshMillis)) * time.Millisecond
	e.wg.Add(2)
	go e.loopDiscover(30 * time.Second)
	go e.loopTick(period)
	log.Printf("gabagool engine started runId=%s refresh=%v", e.om.RunID(), period)
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func (e *Engine) Stop() {
	close(e.stop)
	e.wg.Wait()
	e.om.CancelAll(CancelShutdown)
}

func (e *Engine) loopDiscover(interval time.Duration) {
	defer e.wg.Done()
	t := time.NewTicker(interval)
	defer t.Stop()
	e.refreshMarkets()
	for {
		select {
		case <-e.stop:
			return
		case <-t.C:
			e.refreshMarkets()
		}
	}
}

func (e *Engine) refreshMarkets() {
	mkts := e.disc.ActiveMarkets()
	e.mu.Lock()
	e.active = mkts
	e.mu.Unlock()
	for _, m := range mkts {
		e.feed.SubscribeAssets([]string{m.UpTokenID, m.DownTokenID})
	}
}

func (e *Engine) loopTick(period time.Duration) {
	defer e.wg.Done()
	t := time.NewTicker(period)
	defer t.Stop()
	time.Sleep(time.Second)
	for {
		select {
		case <-e.stop:
			return
		case <-t.C:
			e.tick()
		}
	}
}

func (e *Engine) tick() {
	g := &e.root.Hft.Strategy.Gabagool
	e.pos.RefreshIfStale()
	e.bank.RefreshIfStale(g)
	e.mu.RLock()
	markets := append([]Market(nil), e.active...)
	e.mu.RUnlock()
	e.pos.SyncInventory(markets)

	if e.bank.IsBelowThreshold(g) {
		log.Printf("gabagool: circuit breaker bankroll below threshold")
		e.om.CheckPendingOrders(e.onFill)
		return
	}
	now := time.Now()
	for _, m := range markets {
		e.evaluateMarket(&m, g, now)
	}
	e.om.CheckPendingOrders(e.onFill)
}

func (e *Engine) onFill(st *OrderState, filled decimal.Decimal) {
	if st == nil || st.Market == nil {
		return
	}
	e.pos.RecordFill(st.Market.Slug, st.Direction == DirUp, filled, st.Price)
}

func (e *Engine) evaluateMarket(m *Market, g *config.GabagoolCfg, now time.Time) {
	sec := int64(time.Until(m.EndTime).Seconds())
	maxLife := int64(3600)
	if m.MarketType == "updown-15m" {
		maxLife = 900
	}
	if sec < 0 || sec > maxLife {
		e.om.CancelMarketOrders(m, CancelOutsideLifetime, sec)
		return
	}
	minS := max64(0, g.MinSecondsToEnd)
	maxS := min64(maxLife, max64(minS, g.MaxSecondsToEnd))
	if sec < minS || sec > maxS {
		e.om.CancelMarketOrders(m, CancelOutsideWindow, sec)
		return
	}
	upB, ok1 := e.feed.GetTopOfBook(m.UpTokenID)
	downB, ok2 := e.feed.GetTopOfBook(m.DownTokenID)
	if !ok1 || !ok2 || isStale(upB) {
		e.om.CancelOrder(m.UpTokenID, CancelBookStale, sec, upB, downB)
	}
	if !ok2 || isStale(downB) {
		e.om.CancelOrder(m.DownTokenID, CancelBookStale, sec, downB, upB)
		return
	}
	if !ok1 || isStale(upB) {
		return
	}
	inv := e.pos.GetInventory(m.Slug)
	skewU, skewD := e.qc.CalculateSkewTicks(inv, g)
	e.maybeFastTopUp(m, inv, upB, downB, g, sec)
	if g.CompleteSetTopUpEnabled && sec <= g.CompleteSetTopUpSecondsToEnd {
		absImb := inv.Imbalance().Abs()
		if absImb.GreaterThanOrEqual(decimal.NewFromFloat(g.CompleteSetTopUpMinShares)) {
			var lagDir Direction
			if inv.Imbalance().IsPositive() {
				lagDir = DirDown
			} else {
				lagDir = DirUp
			}
			lagTok := m.DownTokenID
			lagBook := downB
			othBook := upB
			if lagDir == DirUp {
				lagTok = m.UpTokenID
				lagBook = upB
				othBook = downB
			}
			e.maybeTopUpLaggingLeg(m, lagTok, lagDir, lagBook, othBook, g, sec, absImb, PlaceTopUp)
		}
	}
	upTick := e.getTickSize(m.UpTokenID)
	downTick := e.getTickSize(m.DownTokenID)
	if upTick == nil || downTick == nil {
		e.om.CancelMarketOrders(m, CancelBookStale, sec)
		return
	}
	upEntry := e.qc.CalculateEntryPrice(upB, *upTick, g, skewU)
	downEntry := e.qc.CalculateEntryPrice(downB, *downTick, g, skewD)
	if upEntry == nil || downEntry == nil {
		e.om.CancelMarketOrders(m, CancelBookStale, sec)
		return
	}
	if !e.qc.HasMinimumEdge(*upEntry, *downEntry, g) {
		e.om.CancelMarketOrders(m, CancelInsufficientEdge, sec)
		return
	}
	plannedEdge := decimal.NewFromInt(1).Sub(upEntry.Add(*downEntry))
	if e.shouldTake(plannedEdge, upB, downB, g) {
		leg := e.decideTakerLeg(inv, upB, downB, g)
		if leg == DirUp {
			e.maybeTakeToken(m, m.UpTokenID, DirUp, upB, downB, g, sec)
			e.maybeQuoteToken(m, m.DownTokenID, DirDown, downB, upB, g, sec, skewD, downTick)
			return
		}
		if leg == DirDown {
			e.maybeTakeToken(m, m.DownTokenID, DirDown, downB, upB, g, sec)
			e.maybeQuoteToken(m, m.UpTokenID, DirUp, upB, downB, g, sec, skewU, upTick)
			return
		}
	}
	e.maybeQuoteToken(m, m.UpTokenID, DirUp, upB, downB, g, sec, skewU, upTick)
	e.maybeQuoteToken(m, m.DownTokenID, DirDown, downB, upB, g, sec, skewD, downTick)
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func isStale(t *polyws.TopOfBook) bool {
	if t == nil || t.UpdatedAt == nil {
		return true
	}
	return time.Since(*t.UpdatedAt) > 2*time.Second
}

func (e *Engine) getTickSize(tokenID string) *decimal.Decimal {
	now := time.Now()
	if ent, ok := e.tickSize[tokenID]; ok && now.Before(ent.until) {
		s := ent.size
		return &s
	}
	d, err := e.exec.GetTickSize(tokenID)
	if err != nil {
		d = decimal.NewFromFloat(0.01)
	}
	e.tickSize[tokenID] = tickEnt{until: now.Add(10 * time.Minute), size: d}
	return &d
}

func (e *Engine) maybeQuoteToken(m *Market, tokenID string, dir Direction, book, other *polyws.TopOfBook, g *config.GabagoolCfg, sec int64, skew int, tick *decimal.Decimal) {
	if tokenID == "" || book == nil {
		return
	}
	entry := e.qc.CalculateEntryPrice(book, *tick, g, skew)
	if entry == nil {
		return
	}
	exposure := e.qc.CalculateExposure(e.om.OpenOrders(), e.pos.AllInventories())
	shares := e.qc.CalculateShares(m, *entry, g, sec, exposure)
	if shares == nil {
		return
	}
	existing := e.om.GetOrder(tokenID)
	dec := e.om.MaybeReplaceOrder(tokenID, *entry, *shares, g, CancelReplacePrice, sec, book, other)
	if dec == ReplaceSkip {
		return
	}
	reason := PlaceQuote
	if dec == ReplaceDo {
		reason = PlaceReplace
	}
	e.om.PlaceOrder(m, tokenID, dir, *entry, *shares, sec, tick, book, other, existing, reason)
}

func (e *Engine) maybeTakeToken(m *Market, tokenID string, dir Direction, book, other *polyws.TopOfBook, g *config.GabagoolCfg, sec int64) {
	if book == nil || book.BestAsk == nil {
		return
	}
	ba := *book.BestAsk
	if ba.GreaterThan(decimal.NewFromFloat(0.99)) {
		return
	}
	exposure := e.qc.CalculateExposure(e.om.OpenOrders(), e.pos.AllInventories())
	shares := e.qc.CalculateShares(m, ba, g, sec, exposure)
	if shares == nil {
		return
	}
	existing := e.om.GetOrder(tokenID)
	if existing != nil {
		if time.Since(existing.PlacedAt).Milliseconds() < g.MinReplaceMillis {
			return
		}
		e.om.CancelOrder(tokenID, CancelReplacePrice, sec, book, other)
	}
	e.om.PlaceOrder(m, tokenID, dir, ba, *shares, sec, nil, book, other, existing, PlaceTaker)
}

func (e *Engine) maybeTopUpLaggingLeg(m *Market, tokenID string, dir Direction, book, other *polyws.TopOfBook, g *config.GabagoolCfg, sec int64, imb decimal.Decimal, reason PlaceReason) {
	if book == nil || book.BestAsk == nil || book.BestBid == nil {
		return
	}
	ba := *book.BestAsk
	if ba.GreaterThan(decimal.NewFromFloat(0.99)) || imb.LessThan(decimal.NewFromFloat(0.01)) {
		return
	}
	spread := ba.Sub(*book.BestBid)
	if spread.GreaterThan(decimal.NewFromFloat(g.TakerModeMaxSpread)) {
		return
	}
	topUp := imb
	bankUsd := e.bank.ResolveEffective(g)
	if bankUsd.IsPositive() {
		if g.MaxOrderBankrollFraction > 0 {
			cap := bankUsd.Mul(decimal.NewFromFloat(g.MaxOrderBankrollFraction))
			cs := cap.Div(ba).RoundDown(2)
			if cs.LessThan(topUp) {
				topUp = cs
			}
		}
		if g.MaxTotalBankrollFraction > 0 {
			total := bankUsd.Mul(decimal.NewFromFloat(g.MaxTotalBankrollFraction))
			ex := e.qc.CalculateExposure(e.om.OpenOrders(), e.pos.AllInventories())
			rem := total.Sub(ex)
			if !rem.IsPositive() {
				return
			}
			cs := rem.Div(ba).RoundDown(2)
			if cs.LessThan(topUp) {
				topUp = cs
			}
		}
	}
	if mx := e.root.Hft.Risk.MaxOrderNotionalUsd; mx > 0 {
		cs := decimal.NewFromFloat(mx).Div(ba).RoundDown(2)
		if cs.LessThan(topUp) {
			topUp = cs
		}
	}
	topUp = topUp.RoundDown(2)
	if topUp.LessThan(decimal.NewFromFloat(0.01)) {
		return
	}
	existing := e.om.GetOrder(tokenID)
	if existing != nil {
		if time.Since(existing.PlacedAt).Milliseconds() < g.MinReplaceMillis {
			return
		}
		e.om.CancelOrder(tokenID, CancelReplacePrice, sec, book, other)
	}
	e.pos.MarkTopUp(m.Slug)
	e.om.PlaceOrder(m, tokenID, dir, ba, topUp, sec, nil, book, other, existing, reason)
}

func (e *Engine) maybeFastTopUp(m *Market, inv MarketInventory, upB, downB *polyws.TopOfBook, g *config.GabagoolCfg, sec int64) {
	if !g.CompleteSetFastTopUpEnabled {
		return
	}
	imb := inv.Imbalance()
	absImb := imb.Abs()
	if absImb.LessThan(decimal.NewFromFloat(g.CompleteSetFastTopUpMinShares)) {
		return
	}
	now := time.Now()
	if inv.LastTopUpAt != nil && now.Sub(*inv.LastTopUpAt) < time.Duration(g.CompleteSetFastTopUpCooldownMillis)*time.Millisecond {
		return
	}
	var lagDir Direction
	if imb.IsPositive() {
		lagDir = DirDown
	} else {
		lagDir = DirUp
	}
	var leadFill *time.Time
	if lagDir == DirDown {
		leadFill = inv.LastUpFillAt
	} else {
		leadFill = inv.LastDownFillAt
	}
	if leadFill == nil {
		return
	}
	since := int64(now.Sub(*leadFill).Seconds())
	if since < g.CompleteSetFastTopUpMinSecAfterFill || since > g.CompleteSetFastTopUpMaxSecAfterFill {
		return
	}
	var lagFill *time.Time
	if lagDir == DirDown {
		lagFill = inv.LastDownFillAt
	} else {
		lagFill = inv.LastUpFillAt
	}
	if lagFill != nil && !lagFill.Before(*leadFill) {
		return
	}
	lagBook := downB
	othBook := upB
	lagTok := m.DownTokenID
	if lagDir == DirUp {
		lagBook = upB
		othBook = downB
		lagTok = m.UpTokenID
	}
	if lagBook == nil || lagBook.BestBid == nil || lagBook.BestAsk == nil {
		return
	}
	spread := lagBook.BestAsk.Sub(*lagBook.BestBid)
	if spread.GreaterThan(decimal.NewFromFloat(g.TakerModeMaxSpread)) {
		return
	}
	e.pos.MarkTopUp(m.Slug)
	e.maybeTopUpLaggingLeg(m, lagTok, lagDir, lagBook, othBook, g, sec, absImb, PlaceFastTop)
}

func (e *Engine) shouldTake(edge decimal.Decimal, upB, downB *polyws.TopOfBook, g *config.GabagoolCfg) bool {
	if !g.TakerModeEnabled {
		return false
	}
	f, _ := edge.Float64()
	if f > g.TakerModeMaxEdge {
		return false
	}
	if upB == nil || downB == nil || upB.BestBid == nil || upB.BestAsk == nil || downB.BestBid == nil || downB.BestAsk == nil {
		return false
	}
	maxSp := decimal.NewFromFloat(g.TakerModeMaxSpread)
	upSp := upB.BestAsk.Sub(*upB.BestBid)
	downSp := downB.BestAsk.Sub(*downB.BestBid)
	return !upSp.GreaterThan(maxSp) && !downSp.GreaterThan(maxSp)
}

func (e *Engine) decideTakerLeg(inv MarketInventory, upB, downB *polyws.TopOfBook, g *config.GabagoolCfg) Direction {
	minE := decimal.NewFromFloat(g.CompleteSetFastTopUpMinEdge)
	askUp := *upB.BestAsk
	bidDown := *downB.BestBid
	edgeUp := decimal.NewFromInt(1).Sub(askUp.Add(bidDown))
	bidUp := *upB.BestBid
	askDown := *downB.BestAsk
	edgeDown := decimal.NewFromInt(1).Sub(bidUp.Add(askDown))
	upOk := edgeUp.GreaterThanOrEqual(minE)
	downOk := edgeDown.GreaterThanOrEqual(minE)
	if !upOk && !downOk {
		return ""
	}
	if upOk && !downOk {
		return DirUp
	}
	if downOk && !upOk {
		return DirDown
	}
	cmp := edgeUp.Cmp(edgeDown)
	if cmp > 0 {
		return DirUp
	}
	if cmp < 0 {
		return DirDown
	}
	im := inv.Imbalance()
	if im.IsPositive() {
		return DirDown
	}
	if im.IsNegative() {
		return DirUp
	}
	return DirUp
}

func (e *Engine) ActiveMarketCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.active)
}
