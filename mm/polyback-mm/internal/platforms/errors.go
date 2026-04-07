package platforms

import "errors"

var (
	// ErrTradingNotImplemented means the unified client does not expose signed CLOB flow here;
	// use mm/polyback-mm executor / polymarket packages for Polymarket order placement.
	ErrTradingNotImplemented = errors.New("platforms: polymarket trading not implemented in this client; use dedicated CLOB signer")

	// ErrNotConfigured is returned when required credentials or endpoints are missing.
	ErrNotConfigured = errors.New("platforms: client not configured")

	// ErrAuthRequired is returned when a Predict.fun call needs JWT but Authenticate was not run.
	ErrAuthRequired = errors.New("platforms: predict.fun JWT required; call Authenticate first")
)
