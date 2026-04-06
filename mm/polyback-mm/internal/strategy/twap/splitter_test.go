package twap

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestChunks(t *testing.T) {
	max := decimal.RequireFromString("10")
	total := decimal.RequireFromString("25")
	got := Chunks(total, max)
	if len(got) != 3 {
		t.Fatalf("len=%d %v", len(got), got)
	}
	if !got[0].Equal(max) || !got[1].Equal(max) || !got[2].Equal(decimal.RequireFromString("5")) {
		t.Fatalf("chunks: %v", got)
	}
}

func TestChunks_exact(t *testing.T) {
	max := decimal.RequireFromString("7")
	total := decimal.RequireFromString("14")
	got := Chunks(total, max)
	if len(got) != 2 || !got[0].Equal(max) || !got[1].Equal(max) {
		t.Fatalf("%v", got)
	}
}

func TestChunks_nil(t *testing.T) {
	if Chunks(decimal.Zero, decimal.NewFromInt(1)) != nil {
		t.Fatal("expected nil")
	}
}
