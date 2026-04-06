package domain

type OrderSide string

const (
	SideBuy  OrderSide = "BUY"
	SideSell OrderSide = "SELL"
)

func (s OrderSide) EIP712Value() int {
	if s == SideSell {
		return 1
	}
	return 0
}
