package gabagool

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/config"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/domain"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/hftevents"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/platforms/polymarket/api"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/strategy/executorclient"
	polyws "github.com/profitlock/PredictOS/mm/polyback-mm/internal/platforms/polymarket/ws"
	"github.com/shopspring/decimal"
)

type OrderManager struct {
	exec   *executorclient.Client
	events hftevents.Publisher
	runID  string

	mu     sync.Mutex
	orders map[string]*OrderState
}

func NewOrderManager(ex *executorclient.Client, pub hftevents.Publisher, runID string) *OrderManager {
	return &OrderManager{exec: ex, events: pub, runID: runID, orders: map[string]*OrderState{}}
}

func (o *OrderManager) RunID() string { return o.runID }

func (o *OrderManager) OpenOrders() map[string]*OrderState {
	o.mu.Lock()
	defer o.mu.Unlock()
	out := make(map[string]*OrderState, len(o.orders))
	for k, v := range o.orders {
		out[k] = v
	}
	return out
}

func (o *OrderManager) GetOrder(tokenID string) *OrderState {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.orders[tokenID]
}

func ptrStr(s string) *string { return &s }

func (o *OrderManager) PlaceOrder(m *Market, tokenID string, dir Direction, price, size decimal.Decimal, secondsToEnd int64,
	tickSize *decimal.Decimal, book, otherBook *polyws.TopOfBook, replaced *OrderState, reason PlaceReason) {
	reasonStr := string(PlaceQuote)
	if replaced != nil {
		reasonStr = string(PlaceReplace)
	}
	if reason != "" {
		reasonStr = string(reason)
	}
	var repID *string
	var repPrice, repSize *decimal.Decimal
	var repAge *int64
	if replaced != nil {
		s := replaced.OrderID
		repID = &s
		p := replaced.Price
		sz := replaced.Size
		repPrice, repSize = &p, &sz
		ms := time.Since(replaced.PlacedAt).Milliseconds()
		repAge = &ms
	}
	otherToken := ""
	if m != nil {
		if dir == DirUp {
			otherToken = m.DownTokenID
		} else {
			otherToken = m.UpTokenID
		}
	}
	req := &api.LimitOrderRequest{
		TokenID: tokenID, Side: domain.SideBuy, Price: price, Size: size,
		OrderType: ptrStr("GTC"),
	}
	res, err := o.exec.PlaceLimitOrder(req)
	oid := resolveOrderID(res)
	if err != nil || oid == "" {
		errMsg := ""
		if err != nil {
			errMsg = truncateErr(err)
		} else {
			errMsg = "orderId null"
		}
		o.publishLifecycle("PLACE", reasonStr, m, tokenID, dir, secondsToEnd, tickSize, false, errMsg, nil, &price, &size, repID, repPrice, repSize, repAge, book, otherToken, otherBook)
		return
	}
	o.mu.Lock()
	o.orders[tokenID] = &OrderState{
		OrderID: oid, Market: m, TokenID: tokenID, Direction: dir,
		Price: price, Size: size, PlacedAt: time.Now(), MatchedSize: decimal.Zero,
		SecondsToEndAtEntry: secondsToEnd,
	}
	o.mu.Unlock()
	oidp := oid
	o.publishLifecycle("PLACE", reasonStr, m, tokenID, dir, secondsToEnd, tickSize, true, "", &oidp, &price, &size, repID, repPrice, repSize, repAge, book, otherToken, otherBook)
}

func (o *OrderManager) MaybeReplaceOrder(tokenID string, newPrice, newSize decimal.Decimal, g *config.GabagoolCfg,
	reason CancelReason, secondsToEnd int64, book, otherBook *polyws.TopOfBook) ReplaceDecision {
	o.mu.Lock()
	existing := o.orders[tokenID]
	if existing == nil {
		o.mu.Unlock()
		return ReplacePlace
	}
	ageMs := time.Since(existing.PlacedAt).Milliseconds()
	if ageMs < g.MinReplaceMillis {
		o.mu.Unlock()
		return ReplaceSkip
	}
	if existing.Price.Equal(newPrice) && existing.Size.Equal(newSize) {
		o.mu.Unlock()
		return ReplaceSkip
	}
	delete(o.orders, tokenID)
	o.mu.Unlock()
	o.safeCancelUnlocked(existing, reason, secondsToEnd, book, otherBook)
	return ReplaceDo
}

func (o *OrderManager) CancelOrder(tokenID string, reason CancelReason, secondsToEnd int64, book, otherBook *polyws.TopOfBook) {
	o.mu.Lock()
	st := o.orders[tokenID]
	delete(o.orders, tokenID)
	o.mu.Unlock()
	if st == nil {
		return
	}
	o.safeCancelUnlocked(st, reason, secondsToEnd, book, otherBook)
}

func (o *OrderManager) CancelMarketOrders(m *Market, reason CancelReason, secondsToEnd int64) {
	if m == nil {
		return
	}
	o.CancelOrder(m.UpTokenID, reason, secondsToEnd, nil, nil)
	o.CancelOrder(m.DownTokenID, reason, secondsToEnd, nil, nil)
}

func (o *OrderManager) CancelAll(reason CancelReason) {
	o.mu.Lock()
	var all []*OrderState
	for _, st := range o.orders {
		all = append(all, st)
	}
	o.orders = map[string]*OrderState{}
	o.mu.Unlock()
	for _, st := range all {
		o.safeCancelUnlocked(st, reason, 0, nil, nil)
	}
}

func (o *OrderManager) CheckPendingOrders(onFill func(*OrderState, decimal.Decimal)) {
	now := time.Now()
	o.mu.Lock()
	tokens := make([]string, 0, len(o.orders))
	for t := range o.orders {
		tokens = append(tokens, t)
	}
	o.mu.Unlock()
	for _, tokenID := range tokens {
		o.mu.Lock()
		st := o.orders[tokenID]
		o.mu.Unlock()
		if st == nil {
			continue
		}
		o.refreshOrder(tokenID, st, now, onFill)
		o.mu.Lock()
		st = o.orders[tokenID]
		o.mu.Unlock()
		if st == nil {
			continue
		}
		if time.Since(st.PlacedAt) > 5*time.Minute {
			sec := int64(0)
			if st.Market != nil {
				sec = int64(time.Until(st.Market.EndTime).Seconds())
			}
			o.CancelOrder(tokenID, CancelStaleTimeout, sec, nil, nil)
		}
	}
}

func (o *OrderManager) refreshOrder(tokenID string, st *OrderState, now time.Time, onFill func(*OrderState, decimal.Decimal)) {
	if st.LastStatusCheckAt != nil && now.Sub(*st.LastStatusCheckAt) < time.Second {
		return
	}
	raw, err := o.exec.GetOrder(st.OrderID)
	if err != nil {
		t := now
		o.mu.Lock()
		if cur := o.orders[tokenID]; cur != nil && cur.OrderID == st.OrderID {
			cur.LastStatusCheckAt = &t
		}
		o.mu.Unlock()
		return
	}
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	status := firstText(m, "status", "state", "order_status", "orderStatus")
	matched := firstDec(m, "matched_size", "matchedSize", "size_matched", "filled_size", "filledSize")
	remaining := firstDec(m, "remaining_size", "remainingSize", "size_remaining")
	if remaining == nil && matched != nil {
		r := st.Size.Sub(*matched)
		if r.IsNegative() {
			r = decimal.Zero
		}
		remaining = &r
	}
	prev := st.MatchedSize
	if matched != nil && matched.GreaterThan(prev) && onFill != nil {
		delta := matched.Sub(prev)
		onFill(st, delta)
	}
	if terminalStatus(status, matched, remaining, &st.Size) {
		o.mu.Lock()
		delete(o.orders, tokenID)
		o.mu.Unlock()
		return
	}
	t := now
	mat := prev
	if matched != nil {
		mat = *matched
	}
	o.mu.Lock()
	if cur := o.orders[tokenID]; cur != nil && cur.OrderID == st.OrderID {
		cur.MatchedSize = mat
		cur.LastStatusCheckAt = &t
	}
	o.mu.Unlock()
}

func (o *OrderManager) safeCancelUnlocked(st *OrderState, reason CancelReason, sec int64, book, otherBook *polyws.TopOfBook) {
	if st == nil || st.OrderID == "" {
		return
	}
	err := o.exec.CancelOrder(st.OrderID)
	ok := err == nil
	errS := ""
	if err != nil {
		errS = truncateErr(err)
	}
	other := ""
	if st.Market != nil && st.Direction != "" {
		if st.Direction == DirUp {
			other = st.Market.DownTokenID
		} else {
			other = st.Market.UpTokenID
		}
	}
	ms := time.Since(st.PlacedAt).Milliseconds()
	o.publishLifecycle("CANCEL", string(reason), st.Market, st.TokenID, st.Direction, sec, nil, ok, errS, &st.OrderID, &st.Price, &st.Size, nil, nil, nil, &ms, book, other, otherBook)
}

func (o *OrderManager) publishLifecycle(action, reason string, m *Market, tokenID string, dir Direction, secToEnd int64,
	tick *decimal.Decimal, success bool, err string, oid *string, price, size *decimal.Decimal,
	repID *string, repPrice, repSize *decimal.Decimal, repAge *int64, book *polyws.TopOfBook, otherToken string, otherBook *polyws.TopOfBook) {
	if o.events == nil || !o.events.Enabled() {
		return
	}
	mslug, mtype := "", ""
	if m != nil {
		mslug, mtype = m.Slug, m.MarketType
	}
	ev := map[string]any{
		"strategy":       "gabagool-directional",
		"runId":          o.runID,
		"action":         action,
		"reason":         reason,
		"marketSlug":     mslug,
		"marketType":     mtype,
		"tokenId":        tokenID,
		"direction":      string(dir),
		"secondsToEnd":   secToEnd,
		"tickSize":       nil,
		"success":        success,
		"error":          nil,
		"orderId":        nil,
		"price":          nil,
		"size":           nil,
		"replacedOrderId": nil,
		"replacedPrice":  nil,
		"replacedSize":   nil,
		"replacedOrderAgeMillis": nil,
		"orderAgeMillis": nil,
		"book":           book,
		"otherTokenId":   otherToken,
		"otherBook":      otherBook,
	}
	if tick != nil {
		ev["tickSize"] = tick.String()
	}
	if err != "" {
		ev["error"] = err
	}
	if oid != nil {
		ev["orderId"] = *oid
	}
	if price != nil {
		ev["price"] = price.String()
	}
	if size != nil {
		ev["size"] = size.String()
	}
	if repID != nil {
		ev["replacedOrderId"] = *repID
	}
	if repPrice != nil {
		ev["replacedPrice"] = repPrice.String()
	}
	if repSize != nil {
		ev["replacedSize"] = repSize.String()
	}
	if repAge != nil {
		ev["replacedOrderAgeMillis"] = *repAge
	}
	key := ""
	if oid != nil && *oid != "" {
		key = *oid
	} else {
		key = "gabagool:" + mslug + ":" + tokenID
	}
	o.events.Publish(hftevents.StrategyGabagoolOrder, key, ev)
}

func resolveOrderID(res *api.OrderSubmissionResult) string {
	if res == nil || len(res.ClobResponse) == 0 {
		return ""
	}
	var m map[string]any
	if json.Unmarshal(res.ClobResponse, &m) != nil {
		return ""
	}
	if v, ok := m["orderID"].(string); ok && v != "" {
		return v
	}
	if v, ok := m["orderId"].(string); ok && v != "" {
		return v
	}
	return ""
}

func firstText(m map[string]any, keys ...string) string {
	for _, k := range keys {
		v, ok := m[k]
		if !ok || v == nil {
			continue
		}
		s, _ := v.(string)
		if strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func firstDec(m map[string]any, keys ...string) *decimal.Decimal {
	for _, k := range keys {
		v, ok := m[k]
		if !ok || v == nil {
			continue
		}
		switch t := v.(type) {
		case float64:
			d := decimal.NewFromFloat(t)
			return &d
		case string:
			d, err := decimal.NewFromString(strings.TrimSpace(t))
			if err == nil {
				return &d
			}
		}
	}
	return nil
}

func terminalStatus(status string, matched, remaining, requested *decimal.Decimal) bool {
	if remaining != nil && remaining.IsZero() {
		return true
	}
	if matched != nil && requested != nil && !requested.IsZero() && matched.GreaterThanOrEqual(*requested) {
		return true
	}
	u := strings.ToUpper(strings.TrimSpace(status))
	if u == "" {
		return false
	}
	for _, s := range []string{"FILLED", "CANCELED", "CANCELLED", "EXPIRED", "REJECTED", "FAILED", "DONE", "CLOSED"} {
		if strings.Contains(u, s) {
			return true
		}
	}
	return false
}

func truncateErr(err error) string {
	s := err.Error()
	if len(s) > 512 {
		return s[:512]
	}
	return s
}
