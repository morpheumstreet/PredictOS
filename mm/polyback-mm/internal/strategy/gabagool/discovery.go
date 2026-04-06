package gabagool

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/config"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/polymarket/gamma"
)

type Discovery struct {
	cfg   *config.Root
	gamma *gamma.Client
}

func NewDiscovery(root *config.Root, g *gamma.Client) *Discovery {
	return &Discovery{cfg: root, gamma: g}
}

func (d *Discovery) ActiveMarkets() []Market {
	now := time.Now().UTC()
	maxEnd := now.Add(2 * time.Hour)
	discovered := d.fetchActiveUpDownEvents()
	var active []Market
	for _, dm := range discovered {
		if dm.EndTime.IsZero() || !dm.EndTime.After(now) || !dm.EndTime.Before(maxEnd) {
			continue
		}
		dur := time.Hour
		if dm.MarketType == "updown-15m" {
			dur = 15 * time.Minute
		}
		start := dm.EndTime.Add(-dur)
		if now.Before(start) {
			continue
		}
		if dm.Closed {
			continue
		}
		active = append(active, Market{
			Slug:        dm.Slug,
			UpTokenID:   dm.UpTokenID,
			DownTokenID: dm.DownTokenID,
			EndTime:     dm.EndTime,
			MarketType:  dm.MarketType,
		})
	}
	return active
}

type discoveredRaw struct {
	Slug        string
	MarketID    string
	UpTokenID   string
	DownTokenID string
	EndTime     time.Time
	Closed      bool
	MarketType  string
}

func (d *Discovery) fetchActiveUpDownEvents() []discoveredRaw {
	now := time.Now().UTC()
	var candidates []string
	candidates = append(candidates, candidateUpDown15m("btc", now)...)
	candidates = append(candidates, candidateUpDown15m("eth", now)...)

	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		loc = time.UTC
	}
	nowEt := now.In(loc)
	candidates = append(candidates, candidateUpOrDown1h("bitcoin", nowEt)...)
	candidates = append(candidates, candidateUpOrDown1h("ethereum", nowEt)...)

	seen := map[string]struct{}{}
	var out []discoveredRaw
	for _, slug := range candidates {
		if _, ok := seen[slug]; ok {
			continue
		}
		seen[slug] = struct{}{}
		if dm := d.fetchMarketBySlug(slug); dm != nil {
			out = append(out, *dm)
		}
	}
	return out
}

func candidateUpDown15m(prefix string, now time.Time) []string {
	from := now.Add(-30 * time.Minute).Unix()
	to := now.Add(15 * time.Minute).Unix()
	startFrom := (from / 900) * 900
	startTo := (to / 900) * 900
	var slugs []string
	for s := startFrom; s <= startTo; s += 900 {
		slugs = append(slugs, fmt.Sprintf("%s-updown-15m-%d", prefix, s))
	}
	return slugs
}

func candidateUpOrDown1h(assetPrefix string, nowEt time.Time) []string {
	hourStart := nowEt.Truncate(time.Hour)
	times := []time.Time{
		hourStart.Add(-2 * time.Hour),
		hourStart.Add(-time.Hour),
		hourStart,
		hourStart.Add(time.Hour),
	}
	var out []string
	for _, t := range times {
		out = append(out, buildUpOrDown1hSlug(assetPrefix, t))
	}
	return out
}

func buildUpOrDown1hSlug(assetPrefix string, hourStartEt time.Time) string {
	month := strings.ToLower(hourStartEt.Month().String())
	day := hourStartEt.Day()
	h24 := hourStartEt.Hour()
	h12 := h24 % 12
	if h12 == 0 {
		h12 = 12
	}
	ampm := "pm"
	if h24 < 12 {
		ampm = "am"
	}
	return fmt.Sprintf("%s-up-or-down-%s-%d-%d%s-et", assetPrefix, month, day, h12, ampm)
}

func (d *Discovery) fetchMarketBySlug(slug string) *discoveredRaw {
	raw, err := d.gamma.EventsBySlug(slug)
	if err != nil || len(raw) == 0 {
		return nil
	}
	var arr []json.RawMessage
	if json.Unmarshal(raw, &arr) != nil || len(arr) == 0 {
		return nil
	}
	var event map[string]any
	if json.Unmarshal(arr[0], &event) != nil {
		return nil
	}
	return parseEventWithFullDetails(event, slug)
}

func parseEventWithFullDetails(eventNode map[string]any, slug string) *discoveredRaw {
	eventSlug := str(eventNode["slug"])
	if eventSlug == "" {
		return nil
	}
	closed, _ := eventNode["closed"].(bool)
	if closed {
		return nil
	}
	var marketType string
	switch {
	case strings.Contains(eventSlug, "updown-15m"):
		marketType = "updown-15m"
	case strings.Contains(eventSlug, "up-or-down"):
		marketType = "up-or-down"
	default:
		return nil
	}
	var endTime time.Time
	if eds, ok := eventNode["endDate"].(string); ok && eds != "" {
		var err error
		endTime, err = time.Parse(time.RFC3339, eds)
		if err != nil {
			endTime = parseEndTimeFromSlug(eventSlug, marketType)
		}
	} else {
		endTime = parseEndTimeFromSlug(eventSlug, marketType)
	}
	markets, _ := eventNode["markets"].([]any)
	if len(markets) == 0 {
		log.Printf("gabagool discovery: event %s has no markets", eventSlug)
		return nil
	}
	first, _ := markets[0].(map[string]any)
	marketID := str(first["id"])
	tokenIds := parseStringArray(first["clobTokenIds"])
	outcomes := parseStringArray(first["outcomes"])
	var upID, downID string
	for i := 0; i < len(outcomes) && i < len(tokenIds); i++ {
		o := strings.ToLower(strings.TrimSpace(outcomes[i]))
		tid := strings.TrimSpace(tokenIds[i])
		if tid == "" {
			continue
		}
		if o == "up" {
			upID = tid
		}
		if o == "down" {
			downID = tid
		}
	}
	if upID == "" || downID == "" {
		return nil
	}
	return &discoveredRaw{
		Slug: eventSlug, MarketID: marketID, UpTokenID: upID, DownTokenID: downID,
		EndTime: endTime, Closed: closed, MarketType: marketType,
	}
}

func parseStringArray(v any) []string {
	if v == nil {
		return nil
	}
	if s, ok := v.(string); ok {
		var inner []any
		if json.Unmarshal([]byte(s), &inner) == nil {
			return parseStringArray(inner)
		}
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	var out []string
	for _, x := range arr {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func parseEndTimeFromSlug(slug, marketType string) time.Time {
	if marketType == "updown-15m" {
		parts := strings.Split(slug, "-")
		if len(parts) >= 4 {
			var epoch int64
			fmt.Sscanf(parts[len(parts)-1], "%d", &epoch)
			if epoch > 0 {
				return time.Unix(epoch+900, 0).UTC()
			}
		}
	}
	return time.Time{}
}

func str(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	default:
		return ""
	}
}
