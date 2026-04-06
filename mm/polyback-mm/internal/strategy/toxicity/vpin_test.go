package toxicity

import (
	"testing"
	"time"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/domain"
	"github.com/shopspring/decimal"
)

func TestVPINUnsafe_noData(t *testing.T) {
	_, u := VPINUnsafe(nil, 3, 0.7)
	if u {
		t.Fatal("expected safe")
	}
}

func TestVPINUnsafe_imbalanced(t *testing.T) {
	sz := decimal.NewFromFloat(10)
	trades := []domain.Trade{
		{Price: decimal.RequireFromString("0.5"), Size: &sz, Side: "BUY", Timestamp: time.Now()},
		{Price: decimal.RequireFromString("0.5"), Size: &sz, Side: "BUY", Timestamp: time.Now()},
		{Price: decimal.RequireFromString("0.5"), Size: &sz, Side: "BUY", Timestamp: time.Now()},
	}
	imb, u := VPINUnsafe(trades, 3, 0.5)
	if imb < 0.5 {
		t.Fatalf("imbalance %v", imb)
	}
	if !u {
		t.Fatal("expected unsafe")
	}
}
