package tracker

import (
	"errors"
	"net/http"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/tracker/market"
)

func errJSON(msg string) map[string]any {
	return map[string]any{"success": false, "error": msg, "logs": []any{}}
}

func okJSON(data map[string]any) map[string]any {
	return map[string]any{"success": true, "data": data, "logs": []any{}}
}

func statusFromMarketErr(err error) (int, map[string]any) {
	switch {
	case errors.Is(err, market.ErrNotFound):
		return http.StatusOK, errJSON("Market not found - may not be created yet")
	case errors.Is(err, market.ErrTokens):
		return http.StatusOK, errJSON("tokens")
	default:
		return http.StatusInternalServerError, errJSON(err.Error())
	}
}
