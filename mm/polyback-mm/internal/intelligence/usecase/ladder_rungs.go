package usecase

import (
	"fmt"
	"math"
)

// LadderRung is one price level with its USD allocation (total for both legs before split).
type LadderRung struct {
	PricePercent int
	SizeUsd      float64
}

const (
	defaultLadderMaxPrice   = 49
	defaultLadderMinPrice   = 35
	defaultLadderTaper      = 1.5
	ladderMinSharesPerOrder = 5
)

// CalculateLadderRungs matches terminal/src/components/BettingBotTerminalLadder.tsx calculateLadderRungs.
func CalculateLadderRungs(totalBankroll float64, maxPrice, minPrice int, taperFactor float64) ([]LadderRung, error) {
	if totalBankroll <= 0 {
		return nil, fmt.Errorf("sizeUsd must be positive")
	}
	if taperFactor <= 0 {
		return nil, fmt.Errorf("taperFactor must be positive")
	}
	if maxPrice <= minPrice {
		return nil, fmt.Errorf("maxPrice must be greater than minPrice")
	}
	if maxPrice < 1 || maxPrice > 99 || minPrice < 1 || minPrice > 99 {
		return nil, fmt.Errorf("price levels must be between 1 and 99")
	}

	minRungUSD := math.Ceil(float64(ladderMinSharesPerOrder)*float64(maxPrice)/100*100) / 100

	var allPriceLevels []int
	for p := maxPrice; p >= minPrice; p-- {
		allPriceLevels = append(allPriceLevels, p)
	}

	priceLevels := append([]int(nil), allPriceLevels...)
	numRungs := len(priceLevels)

	for numRungs > 1 {
		raw := make([]float64, numRungs)
		for i := 0; i < numRungs; i++ {
			raw[i] = math.Exp(-taperFactor * float64(i) / float64(numRungs))
		}
		sumW := 0.0
		for _, w := range raw {
			sumW += w
		}
		norm := make([]float64, numRungs)
		for i := range raw {
			norm[i] = raw[i] / sumW
		}
		smallestAlloc := totalBankroll * norm[numRungs-1]
		if smallestAlloc >= minRungUSD {
			break
		}
		numRungs--
		priceLevels = allPriceLevels[:numRungs]
	}

	raw := make([]float64, numRungs)
	for i := 0; i < numRungs; i++ {
		raw[i] = math.Exp(-taperFactor * float64(i) / float64(numRungs))
	}
	sumW := 0.0
	for _, w := range raw {
		sumW += w
	}
	norm := make([]float64, numRungs)
	for i := range raw {
		norm[i] = raw[i] / sumW
	}

	out := make([]LadderRung, numRungs)
	for i := 0; i < numRungs; i++ {
		sizeUsd := math.Round(totalBankroll*norm[i]*100) / 100
		out[i] = LadderRung{PricePercent: priceLevels[i], SizeUsd: sizeUsd}
	}
	return out, nil
}
