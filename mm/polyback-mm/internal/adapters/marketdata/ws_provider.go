package marketdata

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/domain"
	polyws "github.com/profitlock/PredictOS/mm/polyback-mm/internal/platforms/polymarket/ws"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/ports/input"
	"github.com/shopspring/decimal"
)

// ClobLike is the WS client surface needed for snapshots (test doubles implement this).
type ClobLike interface {
	GetTopOfBook(assetID string) (*polyws.TopOfBook, bool)
	RecentTrades(assetID string, limit int) []domain.Trade
	LiquidityEMA(assetID string) (bidEMA, askEMA decimal.Decimal, ok bool)
	RegisterBookListener(fn func(assetID string))
}

// WSProvider implements input.MarketDataProvider from the Polymarket CLOB WebSocket client.
type WSProvider struct {
	clob         ClobLike
	maxTradeLook int
}

var _ input.MarketDataProvider = (*WSProvider)(nil)

// NewWSProvider builds a pull-based snapshot provider. maxTradeLook caps trades copied per snapshot (0 = default 64).
func NewWSProvider(clob ClobLike, maxTradeLook int) *WSProvider {
	if maxTradeLook <= 0 {
		maxTradeLook = 64
	}
	return &WSProvider{clob: clob, maxTradeLook: maxTradeLook}
}

func (p *WSProvider) Snapshot(ctx context.Context, assetID string) (domain.MarketSnapshot, bool) {
	_ = ctx
	assetID = strings.TrimSpace(assetID)
	if assetID == "" || p.clob == nil {
		return domain.MarketSnapshot{}, false
	}
	tob, ok := p.clob.GetTopOfBook(assetID)
	if !ok || tob == nil {
		return domain.MarketSnapshot{}, false
	}
	bidEMA, askEMA, emaOk := p.clob.LiquidityEMA(assetID)
	var emaBidPtr, emaAskPtr *decimal.Decimal
	if emaOk {
		b := bidEMA
		a := askEMA
		emaBidPtr = &b
		emaAskPtr = &a
	}
	book := topToL2(assetID, tob, emaBidPtr, emaAskPtr)
	trades := p.clob.RecentTrades(assetID, p.maxTradeLook)
	return domain.MarketSnapshot{
		AssetID:    assetID,
		Book:       book,
		Trades:     trades,
		ObservedAt: time.Now().UTC(),
	}, true
}

// SubscribeL2 registers for full book updates and emits a snapshot for each update (buffered channel).
// The channel is not closed when ctx is cancelled; stop consuming when ctx is done.
func (p *WSProvider) SubscribeL2(ctx context.Context) (<-chan domain.MarketSnapshot, error) {
	if p == nil || p.clob == nil {
		return nil, errors.New("marketdata: nil WSProvider or CLOB")
	}
	out := make(chan domain.MarketSnapshot, 256)
	p.clob.RegisterBookListener(func(assetID string) {
		if ctx.Err() != nil {
			return
		}
		snap, ok := p.Snapshot(ctx, assetID)
		if !ok {
			return
		}
		select {
		case out <- snap:
		default:
		}
	})
	return out, nil
}

func topToL2(assetID string, t *polyws.TopOfBook, emaBid, emaAsk *decimal.Decimal) domain.OrderBookL2 {
	if t == nil {
		return domain.OrderBookL2{AssetID: assetID}
	}
	return domain.OrderBookL2{
		AssetID:        assetID,
		BestBid:        t.BestBid,
		BestAsk:        t.BestAsk,
		BestBidSize:    t.BestBidSize,
		BestAskSize:    t.BestAskSize,
		UpdatedAt:      t.UpdatedAt,
		LastTradeAt:    t.LastTradeAt,
		LastTradePrice: t.LastTradePrice,
		EMABidSize:     emaBid,
		EMAAskSize:     emaAsk,
		BidLevels:      wsLevelsToDomain(t.BidLevels),
		AskLevels:      wsLevelsToDomain(t.AskLevels),
	}
}

func wsLevelsToDomain(in []polyws.BookLevel) []domain.PriceLevel {
	var out []domain.PriceLevel
	for _, x := range in {
		if x.Price == nil {
			continue
		}
		p := *x.Price
		sz := decimal.Zero
		if x.Size != nil {
			sz = *x.Size
		}
		out = append(out, domain.PriceLevel{Price: p, Size: sz})
	}
	return out
}
