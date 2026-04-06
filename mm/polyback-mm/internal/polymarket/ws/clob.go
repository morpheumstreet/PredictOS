package ws

import (
	"encoding/json"
	"log"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/domain"
	"github.com/shopspring/decimal"
)

// eventMarketWsTOB must stay aligned with hftevents.MarketWsTOB / Java HftEventTypes.MARKET_WS_TOB.
const eventMarketWsTOB = "market_ws.tob"

const (
	defaultTradeHistoryCap = 128
	liquidityEmaAlpha      = 0.2
)

type ClobClient struct {
	baseWsURL string
	enabled   bool

	mu             sync.RWMutex
	conn           *websocket.Conn
	topByAsset     map[string]*TopOfBook
	subscribed     map[string]struct{}
	lastMsgAt      time.Time
	tobEmitter     TOBEventEmitter
	tobMinInterval time.Duration
	snapshotEvery  time.Duration
	stop           chan struct{}
	done           sync.WaitGroup

	tradesByAsset   map[string][]domain.Trade
	tradeHistoryCap int
	bidSizeEMA      map[string]decimal.Decimal
	askSizeEMA      map[string]decimal.Decimal

	// optional cache path flush - omitted for brevity; Java persists TOB
}

func NewClobClient(baseWsURL string, enabled bool, emit TOBEventEmitter, tobMinMs, snapshotMs int64) *ClobClient {
	if emit == nil {
		emit = noopTOB{}
	}
	c := &ClobClient{
		baseWsURL:       strings.TrimSuffix(strings.TrimSpace(baseWsURL), "/"),
		enabled:         enabled,
		topByAsset:      make(map[string]*TopOfBook),
		subscribed:      make(map[string]struct{}),
		tobEmitter:      emit,
		tobMinInterval:  time.Duration(tobMinMs) * time.Millisecond,
		snapshotEvery:   time.Duration(snapshotMs) * time.Millisecond,
		stop:            make(chan struct{}),
		tradesByAsset:   make(map[string][]domain.Trade),
		tradeHistoryCap: defaultTradeHistoryCap,
		bidSizeEMA:      make(map[string]decimal.Decimal),
		askSizeEMA:      make(map[string]decimal.Decimal),
	}
	return c
}

func (c *ClobClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn != nil
}

func (c *ClobClient) SubscribedAssetCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.subscribed)
}

func (c *ClobClient) TopOfBookCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.topByAsset)
}

func (c *ClobClient) SubscribeAssets(ids []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" {
			c.subscribed[id] = struct{}{}
		}
	}
	c.sendSubscribeLocked()
}

func (c *ClobClient) GetTopOfBook(assetID string) (*TopOfBook, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	t, ok := c.topByAsset[strings.TrimSpace(assetID)]
	return t, ok
}

// RecentTrades returns the last up to limit trades for an asset (oldest first).
func (c *ClobClient) RecentTrades(assetID string, limit int) []domain.Trade {
	c.mu.RLock()
	defer c.mu.RUnlock()
	assetID = strings.TrimSpace(assetID)
	sl := c.tradesByAsset[assetID]
	if limit <= 0 || len(sl) == 0 {
		return nil
	}
	if len(sl) <= limit {
		out := make([]domain.Trade, len(sl))
		copy(out, sl)
		return out
	}
	return append([]domain.Trade(nil), sl[len(sl)-limit:]...)
}

// LiquidityEMA returns EMA baselines for top-of-book sizes (for liquidity-drop heuristics).
func (c *ClobClient) LiquidityEMA(assetID string) (bidEMA, askEMA decimal.Decimal, ok bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	assetID = strings.TrimSpace(assetID)
	bb, bOk := c.bidSizeEMA[assetID]
	aa, aOk := c.askSizeEMA[assetID]
	return bb, aa, bOk || aOk
}

func (c *ClobClient) recordTradeLocked(assetID string, price *decimal.Decimal, size *decimal.Decimal, side string, ts time.Time) {
	if assetID == "" || price == nil {
		return
	}
	t := domain.Trade{AssetID: assetID, Price: *price, Size: size, Side: side, Timestamp: ts}
	sl := c.tradesByAsset[assetID]
	sl = append(sl, t)
	capN := c.tradeHistoryCap
	if capN <= 0 {
		capN = defaultTradeHistoryCap
	}
	if len(sl) > capN {
		sl = sl[len(sl)-capN:]
	}
	c.tradesByAsset[assetID] = sl
}

func updateSizeEMA(m map[string]decimal.Decimal, id string, val *decimal.Decimal, alpha float64) {
	if val == nil || id == "" {
		return
	}
	prev, ok := m[id]
	if !ok {
		m[id] = *val
		return
	}
	a := decimal.NewFromFloat(alpha)
	one := decimal.NewFromInt(1)
	m[id] = a.Mul(*val).Add(one.Sub(a).Mul(prev))
}

func (c *ClobClient) StartBackground() {
	if !c.enabled {
		return
	}
	c.done.Add(1)
	go c.maintainLoop()
	if c.snapshotEvery > 0 && c.tobEmitter.Enabled() {
		c.done.Add(1)
		go c.snapshotLoop()
	}
}

func (c *ClobClient) Close() {
	close(c.stop)
	c.done.Wait()
	c.mu.Lock()
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
	c.mu.Unlock()
}

func (c *ClobClient) wsURL() string {
	b := c.baseWsURL
	switch {
	case strings.HasPrefix(b, "wss://"), strings.HasPrefix(b, "ws://"):
		return b + "/ws/market"
	case strings.HasPrefix(b, "https://"):
		return "wss://" + strings.TrimPrefix(b, "https://") + "/ws/market"
	case strings.HasPrefix(b, "http://"):
		return "ws://" + strings.TrimPrefix(b, "http://") + "/ws/market"
	default:
		return "wss://" + strings.TrimPrefix(b, "/") + "/ws/market"
	}
}

func (c *ClobClient) connectLocked() error {
	u := c.wsURL()
	parsed, err := url.Parse(u)
	if err != nil {
		return err
	}
	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, _, err := dialer.Dial(parsed.String(), nil)
	if err != nil {
		return err
	}
	c.conn = conn
	c.sendSubscribeLocked()
	go c.readLoop(conn)
	return nil
}

func (c *ClobClient) sendSubscribeLocked() {
	if c.conn == nil || len(c.subscribed) == 0 {
		return
	}
	ids := make([]string, 0, len(c.subscribed))
	for id := range c.subscribed {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	msg, _ := json.Marshal(map[string]any{
		"assets_ids": ids,
		"type":       "market",
	})
	_ = c.conn.WriteMessage(websocket.TextMessage, msg)
}

func (c *ClobClient) maintainLoop() {
	defer c.done.Done()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.stop:
			return
		case <-ticker.C:
			c.mu.Lock()
			need := len(c.subscribed) > 0 && (!c.IsConnected() || time.Since(c.lastMsgAt) > 60*time.Second)
			if need && c.conn == nil {
				if err := c.connectLocked(); err != nil {
					log.Printf("clob ws connect: %v", err)
				}
			} else if need && c.conn != nil {
				_ = c.conn.Close()
				c.conn = nil
				if err := c.connectLocked(); err != nil {
					log.Printf("clob ws reconnect: %v", err)
				}
			}
			c.mu.Unlock()
		}
	}
}

func (c *ClobClient) readLoop(conn *websocket.Conn) {
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			c.mu.Lock()
			if c.conn == conn {
				c.conn = nil
			}
			c.mu.Unlock()
			return
		}
		s := string(data)
		if strings.EqualFold(s, "PING") || strings.EqualFold(s, "PONG") {
			c.mu.Lock()
			c.lastMsgAt = time.Now()
			c.mu.Unlock()
			continue
		}
		c.mu.Lock()
		c.lastMsgAt = time.Now()
		c.mu.Unlock()
		var node any
		if err := json.Unmarshal(data, &node); err != nil {
			continue
		}
		c.handleAny(node)
	}
}

func (c *ClobClient) handleAny(node any) {
	switch v := node.(type) {
	case []any:
		for _, x := range v {
			c.handleAny(x)
		}
	case map[string]any:
		et, _ := v["event_type"].(string)
		switch et {
		case "book":
			c.handleBook(v)
		case "price_change":
			c.handlePriceChange(v)
		case "last_trade_price":
			c.handleLastTrade(v)
		}
	}
}

func (c *ClobClient) handleBook(v map[string]any) {
	assetID, _ := v["asset_id"].(string)
	if assetID == "" {
		return
	}
	bids := levelArray(v, "bids", "buys")
	asks := levelArray(v, "asks", "sells")
	bb, bbs := bestLevel(bids, true)
	ba, bas := bestLevel(asks, false)
	ltp := parseDec(stringField(v, "last_trade_price"))
	now := time.Now().UTC()
	c.mu.Lock()
	defer c.mu.Unlock()
	prev := c.topByAsset[assetID]
	var prevLT *decimal.Decimal
	var prevLTA *time.Time
	if prev != nil {
		prevLT = prev.LastTradePrice
		prevLTA = prev.LastTradeAt
	}
	nextLT := prevLT
	nextLTA := prevLTA
	if ltp != nil {
		nextLT = ltp
		if prevLT == nil || !ltp.Equal(*prevLT) {
			t := now
			nextLTA = &t
			sz := parseDec(stringField(v, "size"))
			side := strings.TrimSpace(stringField(v, "side"))
			c.recordTradeLocked(assetID, ltp, sz, side, now)
		}
	}
	bidS := bbs
	askS := bas
	if prev != nil {
		if bidS == nil {
			bidS = prev.BestBidSize
		}
		if askS == nil {
			askS = prev.BestAskSize
		}
	}
	updateSizeEMA(c.bidSizeEMA, assetID, bidS, liquidityEmaAlpha)
	updateSizeEMA(c.askSizeEMA, assetID, askS, liquidityEmaAlpha)
	tob := &TopOfBook{BestBid: bb, BestAsk: ba, BestBidSize: bidS, BestAskSize: askS, LastTradePrice: nextLT, UpdatedAt: &now, LastTradeAt: nextLTA}
	c.topByAsset[assetID] = tob
	c.maybePublishTOB(assetID, tob)
}

func (c *ClobClient) handlePriceChange(v map[string]any) {
	ch, ok := v["price_changes"].([]any)
	if !ok {
		return
	}
	now := time.Now().UTC()
	for _, x := range ch {
		m, ok := x.(map[string]any)
		if !ok {
			continue
		}
		assetID, _ := m["asset_id"].(string)
		if assetID == "" {
			continue
		}
		c.mu.Lock()
		prev := c.topByAsset[assetID]
		bb := coalesceDec(parseDec(stringField(m, "best_bid")), prev, func(p *TopOfBook) *decimal.Decimal { return p.BestBid })
		ba := coalesceDec(parseDec(stringField(m, "best_ask")), prev, func(p *TopOfBook) *decimal.Decimal { return p.BestAsk })
		bbs := coalesceDec(parseDec(stringField(m, "best_bid_size")), prev, func(p *TopOfBook) *decimal.Decimal { return p.BestBidSize })
		bas := coalesceDec(parseDec(stringField(m, "best_ask_size")), prev, func(p *TopOfBook) *decimal.Decimal { return p.BestAskSize })
		var ltp *decimal.Decimal
		var lta *time.Time
		if prev != nil {
			ltp = prev.LastTradePrice
			lta = prev.LastTradeAt
		}
		updateSizeEMA(c.bidSizeEMA, assetID, bbs, liquidityEmaAlpha)
		updateSizeEMA(c.askSizeEMA, assetID, bas, liquidityEmaAlpha)
		tob := &TopOfBook{BestBid: bb, BestAsk: ba, BestBidSize: bbs, BestAskSize: bas, LastTradePrice: ltp, UpdatedAt: &now, LastTradeAt: lta}
		c.topByAsset[assetID] = tob
		c.maybePublishTOB(assetID, tob)
		c.mu.Unlock()
	}
}

func (c *ClobClient) handleLastTrade(v map[string]any) {
	assetID, _ := v["asset_id"].(string)
	if assetID == "" {
		return
	}
	price := parseDec(stringField(v, "price"))
	sz := parseDec(stringField(v, "size"))
	side := strings.TrimSpace(stringField(v, "side"))
	now := time.Now().UTC()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.recordTradeLocked(assetID, price, sz, side, now)
	prev := c.topByAsset[assetID]
	var bb, ba, bbs, bas *decimal.Decimal
	if prev != nil {
		bb, ba, bbs, bas = prev.BestBid, prev.BestAsk, prev.BestBidSize, prev.BestAskSize
	}
	if bbs != nil {
		updateSizeEMA(c.bidSizeEMA, assetID, bbs, liquidityEmaAlpha)
	}
	if bas != nil {
		updateSizeEMA(c.askSizeEMA, assetID, bas, liquidityEmaAlpha)
	}
	tob := &TopOfBook{
		BestBid: bb, BestAsk: ba, BestBidSize: bbs, BestAskSize: bas,
		LastTradePrice: price, UpdatedAt: &now, LastTradeAt: &now,
	}
	c.topByAsset[assetID] = tob
	c.maybePublishTOB(assetID, tob)
}

var lastPubByAsset sync.Map

func (c *ClobClient) maybePublishTOB(assetID string, tob *TopOfBook) {
	if !c.tobEmitter.Enabled() || tob == nil {
		return
	}
	nowMs := tob.UpdatedAt.UnixMilli()
	if c.tobMinInterval > 0 {
		if v, ok := lastPubByAsset.Load(assetID); ok {
			if nowMs-v.(int64) < c.tobMinInterval.Milliseconds() {
				return
			}
		}
	}
	lastPubByAsset.Store(assetID, nowMs)
	data := map[string]any{
		"assetId":        assetID,
		"bestBid":        decStr(tob.BestBid),
		"bestBidSize":    decStr(tob.BestBidSize),
		"bestAsk":        decStr(tob.BestAsk),
		"bestAskSize":    decStr(tob.BestAskSize),
		"lastTradePrice": decStr(tob.LastTradePrice),
		"updatedAt":      tob.UpdatedAt,
		"lastTradeAt":    tob.LastTradeAt,
	}
	c.tobEmitter.PublishAt(*tob.UpdatedAt, eventMarketWsTOB, assetID, data)
}

func (c *ClobClient) snapshotLoop() {
	defer c.done.Done()
	t := time.NewTicker(c.snapshotEvery)
	defer t.Stop()
	for {
		select {
		case <-c.stop:
			return
		case now := <-t.C:
			if !c.tobEmitter.Enabled() {
				continue
			}
			c.mu.Lock()
			for aid := range c.subscribed {
				tob, ok := c.topByAsset[aid]
				if !ok || tob == nil || tob.BestBid == nil || tob.BestAsk == nil {
					continue
				}
				snap := *tob
				u := now.UTC()
				snap.UpdatedAt = &u
				c.topByAsset[aid] = &snap
				c.maybePublishTOB(aid, &snap)
			}
			c.mu.Unlock()
		}
	}
}

func decStr(d *decimal.Decimal) any {
	if d == nil {
		return nil
	}
	return d.String()
}

func stringField(m map[string]any, k string) string {
	v, ok := m[k]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return decimal.NewFromFloat(t).String()
	default:
		b, _ := json.Marshal(t)
		return strings.Trim(string(b), `"`)
	}
}

func levelArray(v map[string]any, keys ...string) []any {
	for _, k := range keys {
		if a, ok := v[k].([]any); ok {
			return a
		}
	}
	return nil
}

func parseDec(s string) *decimal.Decimal {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return nil
	}
	return &d
}

func bestLevel(levels []any, bestIsMax bool) (*decimal.Decimal, *decimal.Decimal) {
	var bestP, bestS *decimal.Decimal
	for _, lv := range levels {
		m, ok := lv.(map[string]any)
		if !ok {
			continue
		}
		p := parseDec(stringField(m, "price"))
		if p == nil {
			continue
		}
		sz := parseDec(stringField(m, "size"))
		if bestP == nil {
			bestP, bestS = p, sz
			continue
		}
		cmp := p.Cmp(*bestP)
		if (bestIsMax && cmp > 0) || (!bestIsMax && cmp < 0) {
			bestP, bestS = p, sz
		}
	}
	return bestP, bestS
}

func coalesceDec(n *decimal.Decimal, prev *TopOfBook, pick func(*TopOfBook) *decimal.Decimal) *decimal.Decimal {
	if n != nil {
		return n
	}
	if prev == nil {
		return nil
	}
	return pick(prev)
}
