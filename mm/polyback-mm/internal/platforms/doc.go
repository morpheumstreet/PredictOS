// Package platforms defines the shared Platform interface, DTOs, errors, and small HTTP/JSON helpers.
// Venue implementations live in subpackages: polymarket, limitless, predictfun, kalshidflow.
//
// Live API smoke tests (network): go test -tags=live ./internal/platforms/... -timeout 120s -v
// Optional env: PREDICT_FUN_API_KEY, PREDICT_FUN_PRIVATE_KEY, DFLOW_API_KEY, DFLOW_LIVE_EVENT_TICKER.
package platforms
