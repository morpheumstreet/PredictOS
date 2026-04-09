package market

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/tracker/port"
)

var (
	// ErrNotFound is returned when Gamma has no market for the slug and token overrides are incomplete.
	ErrNotFound = errors.New("market not found")
	// ErrTokens is returned when clob token ids cannot be resolved.
	ErrTokens = errors.New("tokens")
)

// UpDown describes a binary market's identity for position tracking.
type UpDown struct {
	Slug        string
	ConditionID string
	Title       string
	UpToken     string
	DownToken   string
}

// ResolveRequest is everything needed to locate CLOB legs (optional body overrides + time).
type ResolveRequest struct {
	Slug         string
	OverrideUp   string
	OverrideDown string
}

// ResolveUpDown loads Gamma market by slug and resolves YES/NO token ids.
func ResolveUpDown(g port.GammaMarket, req ResolveRequest) (UpDown, error) {
	slug := strings.TrimSpace(req.Slug)
	up := strings.TrimSpace(req.OverrideUp)
	down := strings.TrimSpace(req.OverrideDown)

	var market map[string]any
	raw, err := g.MarketBySlug(slug)
	if err != nil {
		if up == "" || down == "" {
			return UpDown{}, fmt.Errorf("%w: %w", ErrNotFound, err)
		}
		return UpDown{Slug: slug, UpToken: up, DownToken: down}, nil
	}
	if err := json.Unmarshal(raw, &market); err != nil {
		return UpDown{}, err
	}

	var title string
	for _, k := range []string{"question", "title"} {
		if s, ok := market[k].(string); ok && strings.TrimSpace(s) != "" {
			title = strings.TrimSpace(s)
			break
		}
	}
	conditionID := strings.TrimSpace(str(market["conditionId"]))

	if up != "" && down != "" {
		return UpDown{Slug: slug, ConditionID: conditionID, Title: title, UpToken: up, DownToken: down}, nil
	}

	clob, _ := market["clobTokenIds"].(string)
	var ids []string
	_ = json.Unmarshal([]byte(clob), &ids)
	if len(ids) < 2 {
		return UpDown{}, ErrTokens
	}
	upTok, downTok := UpDownClobTokenIDs(market, ids)
	if upTok == "" || downTok == "" {
		return UpDown{}, ErrTokens
	}
	return UpDown{
		Slug:        slug,
		ConditionID: conditionID,
		Title:       title,
		UpToken:     upTok,
		DownToken:   downTok,
	}, nil
}

func str(v any) string {
	s, _ := v.(string)
	return s
}
