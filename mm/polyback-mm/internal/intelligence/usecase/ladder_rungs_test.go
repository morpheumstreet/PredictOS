package usecase

import (
	"testing"
)

// Golden vectors from terminal calculateLadderRungs (BettingBotTerminalLadder.tsx), via Node:
// node -e '...same algorithm...'

func TestCalculateLadderRungs_GoldenParity(t *testing.T) {
	tests := []struct {
		name     string
		bankroll float64
		maxP     int
		minP     int
		taper    float64
		want     []LadderRung
	}{
		{
			name: "case1_defaultish", bankroll: 25, maxP: 49, minP: 35, taper: 1.5,
			want: []LadderRung{
				{49, 8.34}, {48, 6.18}, {47, 4.58}, {46, 3.39}, {45, 2.51},
			},
		},
		{
			name: "case2_narrow_range", bankroll: 100, maxP: 48, minP: 40, taper: 2.0,
			want: []LadderRung{
				{48, 23.05}, {47, 18.45}, {46, 14.78}, {45, 11.83}, {44, 9.47},
				{43, 7.59}, {42, 6.07}, {41, 4.86}, {40, 3.89},
			},
		},
		{
			name: "case3_heavy_truncation", bankroll: 10, maxP: 49, minP: 35, taper: 1.5,
			want: []LadderRung{
				{49, 6.79}, {48, 3.21},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalculateLadderRungs(tt.bankroll, tt.maxP, tt.minP, tt.taper)
			if err != nil {
				t.Fatalf("CalculateLadderRungs: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("len got %d want %d: %+v", len(got), len(tt.want), got)
			}
			for i := range tt.want {
				if got[i].PricePercent != tt.want[i].PricePercent {
					t.Errorf("[%d] price got %d want %d", i, got[i].PricePercent, tt.want[i].PricePercent)
				}
				if got[i].SizeUsd != tt.want[i].SizeUsd {
					t.Errorf("[%d] sizeUsd got %v want %v", i, got[i].SizeUsd, tt.want[i].SizeUsd)
				}
			}
		})
	}
}

func TestCalculateLadderRungs_Validation(t *testing.T) {
	if _, err := CalculateLadderRungs(0, 49, 35, 1.5); err == nil {
		t.Fatal("expected error for zero bankroll")
	}
	if _, err := CalculateLadderRungs(25, 35, 49, 1.5); err == nil {
		t.Fatal("expected error when max <= min")
	}
	if _, err := CalculateLadderRungs(25, 49, 35, 0); err == nil {
		t.Fatal("expected error for taper 0")
	}
}
