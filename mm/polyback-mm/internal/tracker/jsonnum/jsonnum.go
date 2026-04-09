package jsonnum

import (
	"encoding/json"
	"strings"
)

// AsFloat64 coerces JSON-decoded numeric values (float64, json.Number, numeric string, int).
func AsFloat64(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case json.Number:
		f, err := t.Float64()
		if err != nil {
			return 0
		}
		return f
	case string:
		f, err := json.Number(strings.TrimSpace(t)).Float64()
		if err != nil {
			return 0
		}
		return f
	default:
		return 0
	}
}
