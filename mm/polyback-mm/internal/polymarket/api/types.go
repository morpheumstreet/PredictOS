package api

import (
	"encoding/json"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/domain"
	"github.com/shopspring/decimal"
)

type LimitOrderRequest struct {
	TokenID           string          `json:"tokenId"`
	Side              domain.OrderSide `json:"side"`
	Price             decimal.Decimal `json:"price"`
	Size              decimal.Decimal `json:"size"`
	OrderType         *string         `json:"orderType,omitempty"`
	TickSize          *decimal.Decimal `json:"tickSize,omitempty"`
	NegRisk           *bool           `json:"negRisk,omitempty"`
	FeeRateBps        *int            `json:"feeRateBps,omitempty"`
	Nonce             *int64          `json:"nonce,omitempty"`
	ExpirationSeconds *int64          `json:"expirationSeconds,omitempty"`
	Taker             *string         `json:"taker,omitempty"`
	DeferExec         *bool           `json:"deferExec,omitempty"`
}

type MarketOrderRequest struct {
	TokenID    string           `json:"tokenId"`
	Side       domain.OrderSide `json:"side"`
	Amount     decimal.Decimal  `json:"amount"`
	Price      decimal.Decimal  `json:"price"`
	OrderType  *string          `json:"orderType,omitempty"`
	TickSize   *decimal.Decimal `json:"tickSize,omitempty"`
	NegRisk    *bool            `json:"negRisk,omitempty"`
	FeeRateBps *int             `json:"feeRateBps,omitempty"`
	Nonce      *int64           `json:"nonce,omitempty"`
	Taker      *string          `json:"taker,omitempty"`
	DeferExec  *bool            `json:"deferExec,omitempty"`
}

type OrderSubmissionResult struct {
	Mode         domain.TradingMode `json:"mode"`
	SignedOrder  any                `json:"signedOrder,omitempty"`
	ClobResponse json.RawMessage    `json:"clobResponse"`
}

type PolymarketHealthResponse struct {
	Mode             string          `json:"mode"`
	ClobRestURL      string          `json:"clobRestUrl"`
	ClobWsURL        string          `json:"clobWsUrl"`
	ChainID          int             `json:"chainId"`
	UseServerTime    bool            `json:"useServerTime"`
	MarketWsEnabled  bool            `json:"marketWsEnabled"`
	UserWsEnabled    bool            `json:"userWsEnabled"`
	Deep             bool            `json:"deep"`
	TokenID          string          `json:"tokenId,omitempty"`
	ServerTimeSeconds *int64         `json:"serverTimeSeconds,omitempty"`
	OrderBook        json.RawMessage `json:"orderBook,omitempty"`
	DeepError        string          `json:"deepError,omitempty"`
}

type PolymarketAccountResponse struct {
	Mode          string `json:"mode"`
	SignerAddress string `json:"signerAddress,omitempty"`
	MakerAddress  string `json:"makerAddress,omitempty"`
	FunderAddress string `json:"funderAddress,omitempty"`
}

type PolymarketBankrollResponse struct {
	Mode                     string          `json:"mode"`
	MakerAddress             string          `json:"makerAddress,omitempty"`
	USDCBalance              decimal.Decimal `json:"usdcBalance"`
	PositionsCurrentValueUsd decimal.Decimal `json:"positionsCurrentValueUsd"`
	PositionsInitialValueUsd decimal.Decimal `json:"positionsInitialValueUsd"`
	TotalEquityUsd           decimal.Decimal `json:"totalEquityUsd"`
	PositionsCount           int             `json:"positionsCount"`
	RedeemablePositionsCount int             `json:"redeemablePositionsCount"`
	MergeablePositionsCount  int             `json:"mergeablePositionsCount"`
	AsOfMillis               int64           `json:"asOfMillis"`
}

type PolymarketPosition struct {
	ProxyWallet   string          `json:"proxyWallet,omitempty"`
	Asset         string          `json:"asset,omitempty"`
	ConditionID   string          `json:"conditionId,omitempty"`
	Size          decimal.Decimal `json:"size"`
	AvgPrice      decimal.Decimal `json:"avgPrice,omitempty"`
	InitialValue  decimal.Decimal `json:"initialValue,omitempty"`
	CurrentValue  decimal.Decimal `json:"currentValue,omitempty"`
	Title         string          `json:"title,omitempty"`
	Slug          string          `json:"slug,omitempty"`
	Outcome       string          `json:"outcome,omitempty"`
	OutcomeIndex  *int            `json:"outcomeIndex,omitempty"`
	CurPrice      decimal.Decimal `json:"curPrice,omitempty"`
	Redeemable    *bool           `json:"redeemable,omitempty"`
	Mergeable     *bool           `json:"mergeable,omitempty"`
	NegativeRisk  *bool           `json:"negativeRisk,omitempty"`
}
