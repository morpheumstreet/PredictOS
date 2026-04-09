package alpharules

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ParseJSONListField mirrors collect.py parse_json_list_field for outcomes / outcomePrices.
func ParseJSONListField(raw any) []any {
	if raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case []any:
		return v
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return nil
		}
		var arr []any
		if json.Unmarshal([]byte(s), &arr) == nil {
			return arr
		}
		return nil
	default:
		return nil
	}
}

// StringifyGammaJSONField stores Gamma field as string: keep JSON strings verbatim, else marshal.
func StringifyGammaJSONField(raw any) (string, error) {
	if raw == nil {
		return "[]", nil
	}
	if s, ok := raw.(string); ok {
		return s, nil
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ClampPrice01 clamps to [0,1].
func ClampPrice01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

// FloatOrNil parses optional float from Gamma map values (JSON numbers decode as float64).
func FloatOrNil(v any) *float64 {
	if v == nil {
		return nil
	}
	f, ok := v.(float64)
	if !ok {
		return nil
	}
	return &f
}

// AnyStringID stringifies Gamma ids (string or JSON number).
func AnyStringID(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return fmt.Sprintf("%.0f", t)
	case json.Number:
		return t.String()
	default:
		return fmt.Sprint(t)
	}
}

// EventStringID returns str(ev["id"]) style id.
func EventStringID(ev map[string]any) string {
	return AnyStringID(ev["id"])
}
