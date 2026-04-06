package marketdata

import (
	"context"
	"testing"
	"time"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/domain"
	polyws "github.com/profitlock/PredictOS/mm/polyback-mm/internal/polymarket/ws"
	"github.com/shopspring/decimal"
)

type fakeClob struct {
	tob    *polyws.TopOfBook
	have   bool
	trades []domain.Trade
	emaBid decimal.Decimal
	emaAsk decimal.Decimal
	emaOk  bool
	listeners []func(assetID string)
}

func (f *fakeClob) RegisterBookListener(fn func(assetID string)) {
	if f == nil || fn == nil {
		return
	}
	f.listeners = append(f.listeners, fn)
}

func (f *fakeClob) fireBook(assetID string) {
	for _, fn := range f.listeners {
		fn(assetID)
	}
}

func (f *fakeClob) GetTopOfBook(assetID string) (*polyws.TopOfBook, bool) {
	_ = assetID
	return f.tob, f.have
}

func (f *fakeClob) RecentTrades(assetID string, limit int) []domain.Trade {
	_ = assetID
	if limit <= 0 || len(f.trades) == 0 {
		return nil
	}
	if len(f.trades) <= limit {
		out := make([]domain.Trade, len(f.trades))
		copy(out, f.trades)
		return out
	}
	return append([]domain.Trade(nil), f.trades[len(f.trades)-limit:]...)
}

func (f *fakeClob) LiquidityEMA(assetID string) (decimal.Decimal, decimal.Decimal, bool) {
	_ = assetID
	return f.emaBid, f.emaAsk, f.emaOk
}

func TestWSProvider_Snapshot_mapsTopOfBookAndTrades(t *testing.T) {
	bb := decimal.RequireFromString("0.48")
	ba := decimal.RequireFromString("0.52")
	bs := decimal.RequireFromString("100")
	asz := decimal.RequireFromString("90")
	now := time.Now().UTC()
	tob := &polyws.TopOfBook{
		BestBid: &bb, BestAsk: &ba, BestBidSize: &bs, BestAskSize: &asz,
		UpdatedAt: &now,
	}
	emaB := decimal.RequireFromString("120")
	emaA := decimal.RequireFromString("95")
	f := &fakeClob{
		tob: tob, have: true,
		trades: []domain.Trade{{AssetID: "a1", Price: decimal.RequireFromString("0.5"), Timestamp: now}},
		emaBid: emaB, emaAsk: emaA, emaOk: true,
	}
	p := NewWSProvider(f, 10)
	snap, ok := p.Snapshot(context.Background(), "a1")
	if !ok {
		t.Fatal("expected snapshot")
	}
	if snap.AssetID != "a1" {
		t.Fatalf("asset id: %q", snap.AssetID)
	}
	if snap.Book.BestBid == nil || !snap.Book.BestBid.Equal(bb) {
		t.Fatalf("bid: %v", snap.Book.BestBid)
	}
	if snap.Book.EMABidSize == nil || !snap.Book.EMABidSize.Equal(emaB) {
		t.Fatalf("ema bid: %v", snap.Book.EMABidSize)
	}
	if len(snap.Trades) != 1 {
		t.Fatalf("trades: %d", len(snap.Trades))
	}
}

func TestWSProvider_Snapshot_emptyAssetID(t *testing.T) {
	p := NewWSProvider(&fakeClob{have: true, tob: &polyws.TopOfBook{}}, 10)
	_, ok := p.Snapshot(context.Background(), "  ")
	if ok {
		t.Fatal("expected miss")
	}
}

func TestWSProvider_SubscribeL2_emitsOnBookUpdate(t *testing.T) {
	bb := decimal.RequireFromString("0.48")
	ba := decimal.RequireFromString("0.52")
	bs := decimal.RequireFromString("100")
	asz := decimal.RequireFromString("90")
	now := time.Now().UTC()
	tob := &polyws.TopOfBook{
		BestBid: &bb, BestAsk: &ba, BestBidSize: &bs, BestAskSize: &asz,
		UpdatedAt: &now,
	}
	f := &fakeClob{tob: tob, have: true, trades: []domain.Trade{{AssetID: "tok1", Price: decimal.RequireFromString("0.5"), Timestamp: now}}}
	p := NewWSProvider(f, 10)
	ctx := context.Background()
	ch, err := p.SubscribeL2(ctx)
	if err != nil {
		t.Fatal(err)
	}
	f.fireBook("tok1")
	select {
	case snap := <-ch:
		if snap.AssetID != "tok1" {
			t.Fatalf("asset: %q", snap.AssetID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for snapshot")
	}
}

func TestWSProvider_SubscribeL2_nilClob(t *testing.T) {
	p := &WSProvider{}
	_, err := p.SubscribeL2(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}
