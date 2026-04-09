package alpharules

import (
	"context"
	"database/sql"
	"encoding/json"
	"math"
	"strings"
)

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func parseURLListFromDB(raw sql.NullString) []string {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return nil
	}
	arr := ParseJSONListField(raw.String)
	var out []string
	for _, x := range arr {
		if x == nil {
			continue
		}
		s := strings.TrimSpace(anyToString(x))
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func configURLsForEvent(cfg SourcesConfig, eid, slug string) []string {
	var out []string
	if cfg.ByEventID != nil {
		out = append(out, cfg.ByEventID[eid]...)
	}
	if slug != "" && cfg.BySlug != nil {
		out = append(out, cfg.BySlug[slug]...)
	}
	return out
}

func dedupePreserveOrder(urls []string) []string {
	seen := make(map[string]struct{}, len(urls))
	var out []string
	for _, u := range urls {
		if u == "" {
			continue
		}
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		out = append(out, u)
	}
	return out
}

// MergeExternalTruthURLs merges DB URLs with config (same rules as collect.py).
func MergeExternalTruthURLs(ctx context.Context, tx *sql.Tx, eid, slug string, cfg SourcesConfig) (*string, error) {
	var raw sql.NullString
	err := tx.QueryRowContext(ctx, `SELECT external_truth_source_urls FROM events WHERE id = ?`, eid).Scan(&raw)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	existing := parseURLListFromDB(raw)
	fromCfg := configURLsForEvent(cfg, eid, slug)
	merged := dedupePreserveOrder(append(append([]string{}, existing...), fromCfg...))
	if len(merged) == 0 {
		return nil, nil
	}
	b, err := json.Marshal(merged)
	if err != nil {
		return nil, err
	}
	s := string(b)
	return &s, nil
}

func tagsToJSON(tags any) (sql.NullString, error) {
	arr, ok := tags.([]any)
	if !ok || len(arr) == 0 {
		return sql.NullString{}, nil
	}
	var out []map[string]any
	for _, t := range arr {
		m, ok := t.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, map[string]any{
			"id":    m["id"],
			"label": m["label"],
			"slug":  m["slug"],
		})
	}
	if len(out) == 0 {
		return sql.NullString{}, nil
	}
	b, err := json.Marshal(out)
	if err != nil {
		return sql.NullString{}, err
	}
	return sql.NullString{String: string(b), Valid: true}, nil
}

func optionalStringCol(v any) interface{} {
	if v == nil {
		return nil
	}
	if s, ok := v.(string); ok {
		if strings.TrimSpace(s) == "" {
			return nil
		}
		return s
	}
	t := strings.TrimSpace(AnyStringID(v))
	if t == "" {
		return nil
	}
	return t
}

func optionalFloatCol(v any) interface{} {
	p := FloatOrNil(v)
	if p == nil {
		return nil
	}
	return *p
}

func nullSQLString(ns sql.NullString) interface{} {
	if !ns.Valid {
		return nil
	}
	return ns.String
}

// UpsertEvent inserts or updates one events row (has_profit_opportunity preserved on conflict).
func UpsertEvent(ctx context.Context, tx *sql.Tx, event map[string]any, fetchedAt string, externalTruthJSON *string) error {
	eid := EventStringID(event)
	if eid == "" {
		return nil
	}
	tagsJSON, err := tagsToJSON(event["tags"])
	if err != nil {
		return err
	}
	var ext interface{}
	if externalTruthJSON != nil {
		ext = *externalTruthJSON
	}
	active, _ := event["active"].(bool)
	closed, _ := event["closed"].(bool)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO events (
			id, slug, ticker, title, description, resolution_source,
			start_date, end_date, active, closed, volume, liquidity,
			tags_json, updated_at_api, fetched_at,
			external_truth_source_urls, has_profit_opportunity, last_scanned_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, COALESCE((SELECT has_profit_opportunity FROM events WHERE id = ?), 0), ?)
		ON CONFLICT(id) DO UPDATE SET
			slug = excluded.slug,
			ticker = excluded.ticker,
			title = excluded.title,
			description = excluded.description,
			resolution_source = excluded.resolution_source,
			start_date = excluded.start_date,
			end_date = excluded.end_date,
			active = excluded.active,
			closed = excluded.closed,
			volume = excluded.volume,
			liquidity = excluded.liquidity,
			tags_json = excluded.tags_json,
			updated_at_api = excluded.updated_at_api,
			fetched_at = excluded.fetched_at,
			external_truth_source_urls = excluded.external_truth_source_urls,
			has_profit_opportunity = events.has_profit_opportunity,
			last_scanned_at = excluded.last_scanned_at`,
		eid,
		optionalStringCol(event["slug"]),
		optionalStringCol(event["ticker"]),
		optionalStringCol(event["title"]),
		optionalStringCol(event["description"]),
		optionalStringCol(event["resolutionSource"]),
		optionalStringCol(event["startDate"]),
		optionalStringCol(event["endDate"]),
		boolToInt(active),
		boolToInt(closed),
		optionalFloatCol(event["volume"]),
		optionalFloatCol(event["liquidity"]),
		nullSQLString(tagsJSON),
		optionalStringCol(event["updatedAt"]),
		fetchedAt,
		ext,
		eid,
		fetchedAt,
	)
	return err
}

// StoreEventBundle upserts event, markets, and market_outcomes for one Gamma event payload.
func StoreEventBundle(ctx context.Context, tx *sql.Tx, event map[string]any, fetchedAt string, cfg SourcesConfig) error {
	eid := EventStringID(event)
	if eid == "" {
		return nil
	}
	slugVal := ""
	if s, ok := event["slug"].(string); ok {
		slugVal = s
	}
	extJSON, err := MergeExternalTruthURLs(ctx, tx, eid, slugVal, cfg)
	if err != nil {
		return err
	}
	if err := UpsertEvent(ctx, tx, event, fetchedAt, extJSON); err != nil {
		return err
	}

	markets, _ := event["markets"].([]any)
	for _, raw := range markets {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		mid := AnyStringID(m["id"])
		if mid == "" {
			continue
		}
		outcomesRaw := m["outcomes"]
		pricesRaw := m["outcomePrices"]
		outcomesList := ParseJSONListField(outcomesRaw)
		pricesList := ParseJSONListField(pricesRaw)

		outcomesStr, err := StringifyGammaJSONField(outcomesRaw)
		if err != nil {
			return err
		}
		pricesStr, err := StringifyGammaJSONField(pricesRaw)
		if err != nil {
			return err
		}

		mActive, _ := m["active"].(bool)
		mClosed, _ := m["closed"].(bool)

		_, err = tx.ExecContext(ctx, `
			INSERT OR REPLACE INTO markets (
				id, event_id, question, slug, condition_id, description,
				resolution_source, active, closed, end_date,
				outcomes_json, outcome_prices_json, updated_at_api, fetched_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			mid, eid,
			optionalStringCol(m["question"]),
			optionalStringCol(m["slug"]),
			optionalStringCol(m["conditionId"]),
			optionalStringCol(m["description"]),
			optionalStringCol(m["resolutionSource"]),
			boolToInt(mActive),
			boolToInt(mClosed),
			optionalStringCol(m["endDate"]),
			outcomesStr,
			pricesStr,
			optionalStringCol(m["updatedAt"]),
			fetchedAt,
		)
		if err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx, `DELETE FROM market_outcomes WHERE market_id = ?`, mid); err != nil {
			return err
		}

		n := max(len(outcomesList), len(pricesList))
		for i := 0; i < n; i++ {
			label := ""
			if i < len(outcomesList) && outcomesList[i] != nil {
				label = anyToString(outcomesList[i])
			}
			price := 0.0
			if i < len(pricesList) && pricesList[i] != nil {
				price = anyToFloat(pricesList[i])
			}
			price = ClampPrice01(price)
			pct := math.Round(price*100.0*1e6) / 1e6 // round(price * 100, 6) like Python

			if _, err := tx.ExecContext(ctx, `
				INSERT INTO market_outcomes (
					market_id, outcome_index, outcome_label, price, price_pct, fetched_at
				) VALUES (?, ?, ?, ?, ?, ?)`,
				mid, i, label, price, pct, fetchedAt,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func anyToFloat(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case string:
		var f float64
		_ = json.Unmarshal([]byte(strings.TrimSpace(t)), &f)
		return f
	default:
		return 0
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
