package alpharules

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/intelligence/llm"
)

const resultSchemaHint = `Reply with a single JSON object only, no markdown, no extra text. Keys: "answer" (string, exactly "yes" or "no") and "supporting_description" (string, concise evidence from the rules text).`

var tmplPlaceholderLeft = regexp.MustCompile(`\{[a-zA-Z_][a-zA-Z0-9_]*\}`)

// DescriptionAgentConfig is the JSON shape from config/description_agent.json (see strat/alpha-rules).
type DescriptionAgentConfig struct {
	StrategiesSource   string           `json:"strategies_source"`
	Model              string           `json:"model"`
	Temperature        float64          `json:"temperature"`
	ParallelWorkers    int              `json:"parallel_workers"`
	BatchSize          int              `json:"batch_size"`
	RequestTimeoutSec  float64          `json:"request_timeout_sec"`
	MaxRetries         int              `json:"max_retries"`
	JSONResponseFormat bool             `json:"json_response_format"`
	Templates          []map[string]any `json:"templates"`
}

// LoadDescriptionAgentConfig reads and validates agent JSON (templates + strategies_source).
func LoadDescriptionAgentConfig(path string) (*DescriptionAgentConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var root map[string]any
	if err := json.Unmarshal(b, &root); err != nil {
		return nil, err
	}
	cfg := &DescriptionAgentConfig{
		StrategiesSource:   "auto",
		Model:              "gpt-4.1-mini",
		Temperature:        0.2,
		ParallelWorkers:    4,
		BatchSize:          16,
		RequestTimeoutSec:  120,
		MaxRetries:         2,
		JSONResponseFormat: true,
	}
	if v, ok := root["strategies_source"].(string); ok && strings.TrimSpace(v) != "" {
		cfg.StrategiesSource = strings.ToLower(strings.TrimSpace(v))
	}
	if v, ok := root["model"].(string); ok && v != "" {
		cfg.Model = v
	}
	if v, ok := root["temperature"].(float64); ok {
		cfg.Temperature = v
	}
	if v, ok := root["parallel_workers"].(float64); ok {
		cfg.ParallelWorkers = int(v)
	}
	if v, ok := root["batch_size"].(float64); ok {
		cfg.BatchSize = int(v)
	}
	if v, ok := root["request_timeout_sec"].(float64); ok {
		cfg.RequestTimeoutSec = v
	}
	if v, ok := root["max_retries"].(float64); ok {
		cfg.MaxRetries = int(v)
	}
	if v, ok := root["json_response_format"].(bool); ok {
		cfg.JSONResponseFormat = v
	}
	tplRaw, _ := root["templates"].([]any)
	for _, x := range tplRaw {
		m, ok := x.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("templates: each entry must be an object")
		}
		cfg.Templates = append(cfg.Templates, m)
	}
	switch cfg.StrategiesSource {
	case "auto", "file", "db", "both":
	default:
		return nil, fmt.Errorf("strategies_source must be auto, file, db, or both")
	}
	if cfg.StrategiesSource == "file" && len(cfg.Templates) == 0 {
		return nil, fmt.Errorf("strategies_source=file requires non-empty templates")
	}
	for _, t := range cfg.Templates {
		if err := validateTemplateMap(t, "config JSON"); err != nil {
			return nil, err
		}
	}
	return cfg, nil
}

func validateTemplateMap(t map[string]any, ctx string) error {
	id, _ := t["id"].(string)
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%s: each template needs string id", ctx)
	}
	sys, sok := t["system"].(string)
	user, uok := t["user"].(string)
	if !sok || !uok {
		return fmt.Errorf("%s: template %q needs string system and user", ctx, id)
	}
	_ = sys
	_ = user
	if raw, ok := t["targets"]; ok && raw != nil {
		arr, ok := raw.([]any)
		if !ok || len(arr) == 0 {
			return fmt.Errorf("%s: template %q targets must be non-empty array or omitted", ctx, id)
		}
		for _, x := range arr {
			s := strings.ToLower(strings.TrimSpace(fmt.Sprint(x)))
			if s != "events" && s != "markets" {
				return fmt.Errorf("%s: template %q targets must be events and/or markets", ctx, id)
			}
		}
	}
	return nil
}

// StrategyTemplate is a resolved prompt template (file and/or DB).
type StrategyTemplate struct {
	ID              string
	System          string
	User            string
	Targets         []string // nil or empty = all targets
	Model           string
	Temperature     *float64
	JSONResponseFmt *bool
}

func mapToStrategyTemplate(t map[string]any) (StrategyTemplate, error) {
	if err := validateTemplateMap(t, "template"); err != nil {
		return StrategyTemplate{}, err
	}
	id, _ := t["id"].(string)
	sys, _ := t["system"].(string)
	user, _ := t["user"].(string)
	var targets []string
	if raw, ok := t["targets"]; ok && raw != nil {
		if arr, ok := raw.([]any); ok {
			for _, x := range arr {
				targets = append(targets, strings.ToLower(strings.TrimSpace(fmt.Sprint(x))))
			}
		}
	}
	st := StrategyTemplate{ID: id, System: sys, User: user, Targets: targets}
	if m, ok := t["model"].(string); ok && strings.TrimSpace(m) != "" {
		st.Model = strings.TrimSpace(m)
	}
	if v, ok := t["temperature"].(float64); ok {
		st.Temperature = &v
	}
	if v, ok := t["json_response_format"].(bool); ok {
		st.JSONResponseFmt = &v
	}
	return st, nil
}

func loadEnabledStrategies(ctx context.Context, db *sql.DB) ([]StrategyTemplate, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, targets_json, system_prompt, user_prompt_template,
		       model, temperature, json_response_format
		FROM description_agent_strategies
		WHERE enabled = 1
		ORDER BY sort_order ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StrategyTemplate
	for rows.Next() {
		var id, targetsJSON, sys, user string
		var model sql.NullString
		var temp sql.NullFloat64
		var jf sql.NullInt64
		if err := rows.Scan(&id, &targetsJSON, &sys, &user, &model, &temp, &jf); err != nil {
			return nil, err
		}
		m := map[string]any{"id": id, "system": sys, "user": user}
		raw := strings.TrimSpace(targetsJSON)
		if raw != "" && raw != "[]" && raw != "null" {
			var arr []any
			if json.Unmarshal([]byte(raw), &arr) == nil && len(arr) > 0 {
				m["targets"] = arr
			}
		}
		if model.Valid && strings.TrimSpace(model.String) != "" {
			m["model"] = model.String
		}
		if temp.Valid {
			v := temp.Float64
			m["temperature"] = v
		}
		if jf.Valid {
			m["json_response_format"] = jf.Int64 != 0
		}
		st, err := mapToStrategyTemplate(m)
		if err != nil {
			continue
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

// ResolveStrategies merges file templates and DB strategies per strategies_source.
func ResolveStrategies(ctx context.Context, db *sql.DB, cfg *DescriptionAgentConfig) ([]StrategyTemplate, error) {
	fileTpls := make([]StrategyTemplate, 0, len(cfg.Templates))
	for _, t := range cfg.Templates {
		st, err := mapToStrategyTemplate(t)
		if err != nil {
			return nil, err
		}
		fileTpls = append(fileTpls, st)
	}
	dbTpls, err := loadEnabledStrategies(ctx, db)
	if err != nil {
		return nil, err
	}
	src := strings.ToLower(strings.TrimSpace(cfg.StrategiesSource))
	switch src {
	case "db":
		return dbTpls, nil
	case "file":
		return fileTpls, nil
	case "both":
		byID := make(map[string]StrategyTemplate, len(fileTpls)+len(dbTpls))
		for _, t := range fileTpls {
			byID[t.ID] = t
		}
		for _, t := range dbTpls {
			byID[t.ID] = t
		}
		var ordered []StrategyTemplate
		seen := make(map[string]struct{})
		for _, t := range fileTpls {
			ordered = append(ordered, byID[t.ID])
			seen[t.ID] = struct{}{}
		}
		for _, t := range dbTpls {
			if _, ok := seen[t.ID]; !ok {
				ordered = append(ordered, t)
				seen[t.ID] = struct{}{}
			}
		}
		return ordered, nil
	default: // auto
		if len(dbTpls) > 0 {
			return dbTpls, nil
		}
		return fileTpls, nil
	}
}

// DescriptionJob is one LLM evaluation task.
type DescriptionJob struct {
	TargetType      string
	TargetID        string
	TemplateID      string
	System          string
	User            string
	InputHash       string
	ModelOverride   string
	Temperature     *float64
	JSONResponseFmt *bool
}

func textHash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func rowContextString(row map[string]any) map[string]string {
	out := make(map[string]string, len(row))
	for k, v := range row {
		if v == nil {
			out[k] = ""
			continue
		}
		out[k] = strings.TrimSpace(fmt.Sprint(v))
	}
	return out
}

func renderTemplate(tpl string, ctx map[string]string) (string, error) {
	out := tpl
	for k, v := range ctx {
		out = strings.ReplaceAll(out, "{"+k+"}", v)
	}
	if tmplPlaceholderLeft.MatchString(out) {
		return "", fmt.Errorf("unknown placeholder remains in template")
	}
	return out, nil
}

func fetchDescriptionEventRows(ctx context.Context, db *sql.DB) ([]map[string]any, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, slug, title, description, resolution_source
		FROM events
		WHERE description IS NOT NULL AND TRIM(description) != ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id, slug, title, desc, res sql.NullString
		if err := rows.Scan(&id, &slug, &title, &desc, &res); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"target_type":       "event",
			"target_id":         id.String,
			"event_id":          id.String,
			"event_slug":        slug.String,
			"event_title":       title.String,
			"description":       desc.String,
			"resolution_source": res.String,
		})
	}
	return out, rows.Err()
}

func fetchDescriptionMarketRows(ctx context.Context, db *sql.DB) ([]map[string]any, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT m.id, m.slug, m.question, m.description, m.resolution_source,
		       e.id, e.slug, e.title
		FROM markets m
		JOIN events e ON e.id = m.event_id
		WHERE m.description IS NOT NULL AND TRIM(m.description) != ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var mid, mslug, mq, md, mres, eid, eslug, etitle sql.NullString
		if err := rows.Scan(&mid, &mslug, &mq, &md, &mres, &eid, &eslug, &etitle); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"target_type":       "market",
			"target_id":         mid.String,
			"market_id":         mid.String,
			"market_slug":       mslug.String,
			"market_question":   mq.String,
			"description":       md.String,
			"resolution_source": mres.String,
			"event_id":          eid.String,
			"event_slug":        eslug.String,
			"event_title":       etitle.String,
		})
	}
	return out, rows.Err()
}

func existingInputHash(ctx context.Context, db *sql.DB, targetType, targetID, templateID string) (string, error) {
	var h sql.NullString
	err := db.QueryRowContext(ctx, `
		SELECT input_hash FROM description_agent_results
		WHERE target_type = ? AND target_id = ? AND template_id = ?`,
		targetType, targetID, templateID).Scan(&h)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if !h.Valid {
		return "", nil
	}
	return h.String, nil
}

// BuildDescriptionJobs expands templates × rows (same skip/reprocess rules as description_agent.py).
func BuildDescriptionJobs(ctx context.Context, db *sql.DB, templates []StrategyTemplate, targets map[string]bool, reprocess bool, maxDesc *int, maxJobs *int) ([]DescriptionJob, error) {
	var rows []map[string]any
	if targets["events"] {
		r, err := fetchDescriptionEventRows(ctx, db)
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	if targets["markets"] {
		r, err := fetchDescriptionMarketRows(ctx, db)
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	var jobs []DescriptionJob
	for _, row := range rows {
		desc, _ := row["description"].(string)
		if maxDesc != nil && len(desc) > *maxDesc {
			desc = desc[:*maxDesc] + "\n\n[truncated]"
			row = shallowCopyRow(row)
			row["description"] = desc
		}
		h := textHash(desc)
		tt := row["target_type"].(string)
		for _, t := range templates {
			if len(t.Targets) > 0 {
				allowed := make(map[string]struct{}, len(t.Targets))
				for _, x := range t.Targets {
					allowed[x] = struct{}{}
				}
				if tt == "event" {
					if _, ok := allowed["events"]; !ok {
						continue
					}
				}
				if tt == "market" {
					if _, ok := allowed["markets"]; !ok {
						continue
					}
				}
			}
			if !reprocess {
				prev, err := existingInputHash(ctx, db, tt, row["target_id"].(string), t.ID)
				if err != nil {
					return nil, err
				}
				if prev == h {
					continue
				}
			}
			ctxStr := rowContextString(row)
			userRendered, err := renderTemplate(t.User, ctxStr)
			if err != nil {
				return nil, fmt.Errorf("template %q: %w", t.ID, err)
			}
			sysBase, err := renderTemplate(t.System, ctxStr)
			if err != nil {
				return nil, fmt.Errorf("template %q system: %w", t.ID, err)
			}
			system := strings.TrimRight(sysBase, "\n") + "\n\n" + resultSchemaHint
			j := DescriptionJob{
				TargetType: tt, TargetID: row["target_id"].(string), TemplateID: t.ID,
				System: system, User: userRendered, InputHash: h,
			}
			if t.Model != "" {
				j.ModelOverride = t.Model
			}
			if t.Temperature != nil {
				j.Temperature = t.Temperature
			}
			if t.JSONResponseFmt != nil {
				j.JSONResponseFmt = t.JSONResponseFmt
			}
			jobs = append(jobs, j)
			if maxJobs != nil && len(jobs) >= *maxJobs {
				return jobs, nil
			}
		}
	}
	return jobs, nil
}

func shallowCopyRow(row map[string]any) map[string]any {
	out := make(map[string]any, len(row))
	for k, v := range row {
		out[k] = v
	}
	return out
}

func stripCodeFence(s string) string {
	t := strings.TrimSpace(s)
	if !strings.HasPrefix(t, "```") {
		return t
	}
	lines := strings.Split(t, "\n")
	if len(lines) > 0 && strings.HasPrefix(strings.TrimSpace(lines[0]), "```") {
		lines = lines[1:]
	}
	if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "```" {
		lines = lines[:len(lines)-1]
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func normalizeYesNo(val any) string {
	switch v := val.(type) {
	case bool:
		if v {
			return "yes"
		}
		return "no"
	case string:
		s := strings.ToLower(strings.TrimSpace(v))
		switch s {
		case "yes", "y", "true", "1":
			return "yes"
		case "no", "n", "false", "0":
			return "no"
		}
	}
	return ""
}

func parseYesNoJSONResponse(raw string) (canonical string, answer string, err error) {
	cleaned := stripCodeFence(raw)
	var obj map[string]any
	if err := json.Unmarshal([]byte(cleaned), &obj); err != nil {
		return "", "", fmt.Errorf("invalid JSON: %w", err)
	}
	ans := normalizeYesNo(obj["answer"])
	if ans == "" {
		return "", "", fmt.Errorf("missing or invalid answer (need yes/no)")
	}
	support, _ := obj["supporting_description"].(string)
	if strings.TrimSpace(support) == "" {
		if s, ok := obj["support"].(string); ok {
			support = s
		} else if s, ok := obj["rationale"].(string); ok {
			support = s
		} else if s, ok := obj["description"].(string); ok {
			support = s
		}
	}
	if strings.TrimSpace(support) == "" {
		return "", "", fmt.Errorf("missing supporting_description")
	}
	norm := map[string]string{"answer": ans, "supporting_description": strings.TrimSpace(support)}
	b, err := json.Marshal(norm)
	if err != nil {
		return "", "", err
	}
	return string(b), ans, nil
}

func sqlOptionalStr(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}

func upsertDescriptionResult(ctx context.Context, db *sql.DB, j DescriptionJob, model, resultJSON, answer, outputText, errStr *string, processedAt string) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO description_agent_results (
			target_type, target_id, template_id, input_hash, model,
			result_json, answer, output_text, error, processed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(target_type, target_id, template_id) DO UPDATE SET
			input_hash = excluded.input_hash,
			model = excluded.model,
			result_json = excluded.result_json,
			answer = excluded.answer,
			output_text = excluded.output_text,
			error = excluded.error,
			processed_at = excluded.processed_at`,
		j.TargetType, j.TargetID, j.TemplateID, j.InputHash,
		sqlOptionalStr(model), sqlOptionalStr(resultJSON), sqlOptionalStr(answer),
		sqlOptionalStr(outputText), sqlOptionalStr(errStr), processedAt,
	)
	return err
}

// DescriptionAgentOptions controls a description-agent run.
type DescriptionAgentOptions struct {
	Workers             int
	BatchSize           int
	Targets             map[string]bool // "events", "markets"
	Reprocess           bool
	MaxDescriptionChars *int
	MaxJobs             *int
	DryRun              bool
}

// RunDescriptionAgent executes LLM jobs over catalog descriptions (OpenAI/Grok via llm.Facade).
func RunDescriptionAgent(ctx context.Context, f *llm.Facade, db *sql.DB, cfg *DescriptionAgentConfig, opt DescriptionAgentOptions) (queued, okN, failN int, err error) {
	if err := InitAgentTables(ctx, db); err != nil {
		return 0, 0, 0, err
	}
	templates, err := ResolveStrategies(ctx, db, cfg)
	if err != nil {
		return 0, 0, 0, err
	}
	if len(templates) == 0 {
		return 0, 0, 0, fmt.Errorf("no strategies: enable description_agent_strategies or add JSON templates")
	}
	jobs, err := BuildDescriptionJobs(ctx, db, templates, opt.Targets, opt.Reprocess, opt.MaxDescriptionChars, opt.MaxJobs)
	if err != nil {
		return 0, 0, 0, err
	}
	queued = len(jobs)
	if opt.DryRun {
		return queued, 0, 0, nil
	}
	if queued == 0 {
		return 0, 0, 0, nil
	}
	workers := opt.Workers
	if workers < 1 {
		workers = 1
	}
	batch := opt.BatchSize
	if batch < 1 {
		batch = 1
	}
	defaultModel := cfg.Model
	if strings.TrimSpace(defaultModel) == "" {
		defaultModel = "gpt-4.1-mini"
	}
	now := UTCNowISO()

	for i := 0; i < len(jobs); i += batch {
		end := i + batch
		if end > len(jobs) {
			end = len(jobs)
		}
		chunk := jobs[i:end]
		type res struct {
			j          DescriptionJob
			err        *string
			resultJSON *string
			answer     *string
			rawOut     *string
		}
		results := make([]res, len(chunk))

		var wg sync.WaitGroup
		sem := make(chan struct{}, workers)
		for idx := range chunk {
			wg.Add(1)
			go func(ix int) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				j := chunk[ix]
				model := strings.TrimSpace(j.ModelOverride)
				if model == "" {
					model = defaultModel
				}
				// Temperature / response_format per-strategy: llm.Facade uses provider defaults + json_object where supported.

				var lastErr error
				delay := 2 * time.Second
				for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
					txt, _, _, _, callErr := f.CompleteJSON(ctx, model, j.System, j.User, nil)
					if callErr == nil {
						canon, ans, perr := parseYesNoJSONResponse(txt)
						if perr != nil {
							es := fmt.Sprintf("JSONValidationError: %v", perr)
							raw := txt
							if len(raw) > 8000 {
								raw = raw[:8000] + "\n…[truncated]"
							}
							results[ix] = res{j: j, err: &es, rawOut: &raw}
							return
						}
						results[ix] = res{j: j, resultJSON: &canon, answer: &ans}
						return
					}
					lastErr = callErr
					if attempt < cfg.MaxRetries {
						select {
						case <-time.After(delay):
							if delay < 60*time.Second {
								delay *= 2
							}
						case <-ctx.Done():
							es := ctx.Err().Error()
							results[ix] = res{j: j, err: &es}
							return
						}
					}
				}
				es := lastErr.Error()
				results[ix] = res{j: j, err: &es}
			}(idx)
		}
		wg.Wait()

		for _, r := range results {
			model := strings.TrimSpace(r.j.ModelOverride)
			if model == "" {
				model = defaultModel
			}
			if err := upsertDescriptionResult(ctx, db, r.j, &model, r.resultJSON, r.answer, r.rawOut, r.err, now); err != nil {
				return queued, okN, failN, err
			}
			if r.err != nil {
				failN++
			} else {
				okN++
			}
		}
	}

	return queued, okN, failN, nil
}
