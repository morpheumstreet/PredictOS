package alpharules

import (
	"strings"
	"time"
)

// ParseAPIDateUTC parses Gamma-style ISO timestamps (including trailing Z) to UTC.
func ParseAPIDateUTC(raw any) *time.Time {
	s, ok := raw.(string)
	if !ok {
		return nil
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if strings.HasSuffix(s, "Z") {
		s = s[:len(s)-1] + "+00:00"
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05-07:00", s)
	}
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05", s)
		if err == nil {
			t = t.UTC()
		}
	}
	if err != nil {
		return nil
	}
	u := t.UTC()
	return &u
}
