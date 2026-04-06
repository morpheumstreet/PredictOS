package twap

import "github.com/shopspring/decimal"

// Chunks splits total into segments of at most maxChunk (last segment may be smaller).
// Returns nil if total or maxChunk is not positive.
func Chunks(total, maxChunk decimal.Decimal) []decimal.Decimal {
	if !total.IsPositive() || !maxChunk.IsPositive() {
		return nil
	}
	var out []decimal.Decimal
	rem := total
	for rem.GreaterThan(maxChunk) {
		out = append(out, maxChunk)
		rem = rem.Sub(maxChunk)
	}
	if rem.IsPositive() {
		out = append(out, rem)
	}
	return out
}
