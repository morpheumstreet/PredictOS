package position

import (
	"math"
	"testing"
)

func TestPairMetrics_profitLocked(t *testing.T) {
	st, pc, minS, gp, tc, prof, rp := PairMetrics(100, 100, 0.45, 0.50, 45, 50)
	if st != "PROFIT_LOCKED" {
		t.Fatalf("status: got %q", st)
	}
	if pc == nil {
		t.Fatal("pairCost nil")
	}
	if math.Abs(pc.(float64)-0.95) > 1e-9 {
		t.Fatalf("pairCost: %v", pc)
	}
	if math.Abs(minS-100) > 1e-9 || math.Abs(gp-100) > 1e-9 {
		t.Fatalf("min/gp: %v %v", minS, gp)
	}
	if math.Abs(tc-95) > 1e-9 || math.Abs(prof-5) > 1e-9 {
		t.Fatalf("cost/profit: %v %v", tc, prof)
	}
	if math.Abs(rp-100.0*5.0/95.0) > 1e-6 {
		t.Fatalf("return %%: %v", rp)
	}
}

func TestPairMetrics_noPosition(t *testing.T) {
	st, pc, _, _, _, _, _ := PairMetrics(0, 0, 0, 0, 0, 0)
	if st != "NO_POSITION" || pc != nil {
		t.Fatalf("got %q pc=%v", st, pc)
	}
}
