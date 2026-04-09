package alpharules

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// SourcesConfig maps event ids and slugs to external truth URL lists (see strat/alpha-rules config).
type SourcesConfig struct {
	ByEventID map[string][]string
	BySlug    map[string][]string
}

// LoadSourcesConfig reads JSON from path. Missing file yields empty config; other read errors are returned.
func LoadSourcesConfig(path string) (SourcesConfig, error) {
	var out SourcesConfig
	if strings.TrimSpace(path) == "" {
		return out, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return out, err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return out, err
	}
	if v, ok := raw["by_event_id"]; ok {
		out.ByEventID = parseStringListMap(v)
	}
	if v, ok := raw["by_slug"]; ok {
		out.BySlug = parseStringListMap(v)
	}
	return out, nil
}

func parseStringListMap(raw json.RawMessage) map[string][]string {
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil {
		return nil
	}
	out := make(map[string][]string, len(m))
	for k, v := range m {
		out[k] = stringListFromJSON(v)
	}
	return out
}

func stringListFromJSON(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var s string
	if json.Unmarshal(raw, &s) == nil && strings.TrimSpace(s) != "" {
		return []string{strings.TrimSpace(s)}
	}
	var arr []any
	if json.Unmarshal(raw, &arr) != nil {
		return nil
	}
	var out []string
	for _, x := range arr {
		if x == nil {
			continue
		}
		t := strings.TrimSpace(anyToString(x))
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

func anyToString(x any) string {
	switch v := x.(type) {
	case string:
		return v
	default:
		return fmt.Sprint(v)
	}
}
