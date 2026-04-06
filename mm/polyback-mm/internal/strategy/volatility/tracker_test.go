package volatility

import (
	"testing"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/config"
	"github.com/shopspring/decimal"
)

func TestTracker_SpreadAddon_disabledWhenScaleZero(t *testing.T) {
	cfg := config.MarketMakerCfg{EwmaVolSpreadScale: 0}
	tr := NewTracker(cfg)
	a := tr.SpreadAddon("x", decimal.RequireFromString("0.5"))
	if !a.IsZero() {
		t.Fatalf("want 0, got %v", a)
	}
}

func TestTracker_SpreadAddon_coldStartNoAddon(t *testing.T) {
	cfg := config.MarketMakerCfg{
		EwmaVolSpreadScale: 10,
		EwmaVolLambda:      0.5,
	}
	tr := NewTracker(cfg)
	a := tr.SpreadAddon("a", decimal.RequireFromString("0.50"))
	if !a.IsZero() {
		t.Fatalf("first tick should be 0, got %v", a)
	}
}

func TestTracker_SpreadAddon_flatMidStaysLow(t *testing.T) {
	cfg := config.MarketMakerCfg{
		EwmaVolSpreadScale: 100,
		EwmaVolLambda:      0.9,
		EwmaVolSpreadMax:   1,
	}
	tr := NewTracker(cfg)
	m := decimal.RequireFromString("0.50")
	_ = tr.SpreadAddon("a", m)
	for i := 0; i < 5; i++ {
		_ = tr.SpreadAddon("a", m)
	}
	a := tr.SpreadAddon("a", m)
	f, _ := a.Float64()
	if f > 0.01 {
		t.Fatalf("flat mids should give tiny addon, got %v", a)
	}
}

func TestTracker_SpreadAddon_shockIncreasesAddon(t *testing.T) {
	cfg := config.MarketMakerCfg{
		EwmaVolSpreadScale: 2.0,
		EwmaVolLambda:      0.5,
		EwmaVolSpreadMax:   1,
	}
	tr := NewTracker(cfg)
	_ = tr.SpreadAddon("a", decimal.RequireFromString("0.50"))
	var flat decimal.Decimal
	for i := 0; i < 8; i++ {
		flat = tr.SpreadAddon("a", decimal.RequireFromString("0.50"))
	}
	spike := tr.SpreadAddon("a", decimal.RequireFromString("0.85"))
	if !spike.GreaterThan(flat) {
		t.Fatalf("after shock want bigger addon than flat: flat=%v spike=%v", flat, spike)
	}
}

func TestTracker_SpreadAddon_respectsMax(t *testing.T) {
	cfg := config.MarketMakerCfg{
		EwmaVolSpreadScale: 1000,
		EwmaVolLambda:      0.01,
		EwmaVolSpreadMax:   0.05,
	}
	tr := NewTracker(cfg)
	_ = tr.SpreadAddon("a", decimal.RequireFromString("0.50"))
	a := tr.SpreadAddon("a", decimal.RequireFromString("0.10"))
	f, _ := a.Float64()
	if f > 0.0501 {
		t.Fatalf("addon should cap at max: got %v", f)
	}
}
