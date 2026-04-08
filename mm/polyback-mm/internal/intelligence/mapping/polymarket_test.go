package mapping

import "testing"

func TestRoundToTick(t *testing.T) {
	if v := roundToTick(0.473, "0.01"); v < 0.469 || v > 0.471 {
		t.Fatalf("got %v", v)
	}
}

func TestValidateBudget(t *testing.T) {
	if ValidateBudget(50) != nil {
		t.Fatal("expected ok")
	}
	if ValidateBudget(0) == nil {
		t.Fatal("expected err")
	}
}
