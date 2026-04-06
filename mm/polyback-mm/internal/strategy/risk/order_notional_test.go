package risk

import (
	"testing"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/config"
	"github.com/shopspring/decimal"
)

func TestOrderNotionalAllowed_unsetCap(t *testing.T) {
	root := &config.Root{}
	if !OrderNotionalAllowed(root, decimal.NewFromInt(1_000_000)) {
		t.Fatal("want allowed when cap unset")
	}
}

func TestOrderNotionalAllowed_exceeds(t *testing.T) {
	root := &config.Root{}
	root.Hft.Risk.MaxOrderNotionalUsd = 100
	if OrderNotionalAllowed(root, decimal.NewFromInt(200)) {
		t.Fatal("want reject")
	}
	if !OrderNotionalAllowed(root, decimal.NewFromInt(50)) {
		t.Fatal("want allow")
	}
}
