package domain

import "github.com/shopspring/decimal"

// MMQuote is the study.md-style two-sided quote. Gabagool maker-buy uses Bid for limit bids.
type MMQuote struct {
	Fair   decimal.Decimal
	Spread decimal.Decimal
	Bid    decimal.Decimal
	Ask    decimal.Decimal
}
