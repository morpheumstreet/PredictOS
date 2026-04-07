package platforms

import (
	"encoding/json"
	"strings"

	"github.com/shopspring/decimal"
)

// DecimalFromJSON parses a JSON number or string field into decimal (for loose API payloads).
func DecimalFromJSON(raw json.RawMessage) decimal.Decimal {
	return decFromFlexible(raw)
}

// DecimalFromInterface coerces string, float64, or json.Number to decimal.
func DecimalFromInterface(v interface{}) decimal.Decimal {
	switch t := v.(type) {
	case string:
		d, err := decimal.NewFromString(t)
		if err != nil {
			return decimal.Zero
		}
		return d
	case float64:
		return decimal.NewFromFloat(t)
	case json.Number:
		d, err := decimal.NewFromString(string(t))
		if err != nil {
			return decimal.Zero
		}
		return d
	default:
		return decimal.Zero
	}
}

func decFromFlexible(raw json.RawMessage) decimal.Decimal {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" {
		return decimal.Zero
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err == nil {
		return decimal.NewFromFloat(f)
	}
	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		d, err := decimal.NewFromString(str)
		if err != nil {
			return decimal.Zero
		}
		return d
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero
	}
	return d
}
