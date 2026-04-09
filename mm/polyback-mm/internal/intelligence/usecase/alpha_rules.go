package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/config"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/intelligence/alpharules"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/intelligence/llm"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/platforms/polymarket/gamma"
)

// AlphaRules implements Gamma → SQLite catalog scan and description-agent runs (strat/alpha-rules in Go).
type AlphaRules struct {
	root *config.Root
	hc   *http.Client
	llm  *llm.Facade
}

func NewAlphaRules(root *config.Root, hc *http.Client, lf *llm.Facade) *AlphaRules {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &AlphaRules{root: root, hc: hc, llm: lf}
}

type alphaRulesCollectReq struct {
	SQLitePath        string `json:"sqlite_path"`
	SourcesConfigPath string `json:"sources_config_path"`
	Limit             int    `json:"limit"`
	MaxEvents         *int   `json:"max_events"`
	SleepMs           int    `json:"sleep_ms"`
	RecordScanRun     *bool  `json:"record_scan_run"`
}

// RunCollect POST body: optional sqlite_path (else ALPHA_RULES_SQLITE), sources_config_path, limit, max_events, sleep_ms, record_scan_run.
func (a *AlphaRules) RunCollect(ctx context.Context, body []byte) (status int, out map[string]any) {
	start := time.Now()
	meta := map[string]any{
		"requestId":        fmt.Sprintf("%d", time.Now().UnixNano()),
		"timestamp":        time.Now().UTC().Format(time.RFC3339Nano),
		"processingTimeMs": time.Since(start).Milliseconds(),
	}
	var req alphaRulesCollectReq
	if err := json.Unmarshal(body, &req); err != nil {
		return http.StatusBadRequest, map[string]any{"success": false, "error": "Invalid JSON", "metadata": meta}
	}
	dbPath := strings.TrimSpace(req.SQLitePath)
	if dbPath == "" {
		dbPath = alpharules.DefaultSQLitePath()
	}
	if dbPath == "" {
		return http.StatusBadRequest, map[string]any{
			"success": false,
			"error":   "sqlite_path is required (or set ALPHA_RULES_SQLITE)",
			"metadata": meta,
		}
	}
	srcPath := strings.TrimSpace(req.SourcesConfigPath)
	if srcPath == "" {
		srcPath = strings.TrimSpace(os.Getenv("ALPHA_RULES_EXTERNAL_TRUTH_JSON"))
	}
	cfg, err := alpharules.LoadSourcesConfig(srcPath)
	if err != nil {
		return http.StatusBadRequest, map[string]any{"success": false, "error": fmt.Sprintf("sources config: %v", err), "metadata": meta}
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 500
	}
	sleep := time.Duration(req.SleepMs) * time.Millisecond
	if req.SleepMs <= 0 {
		sleep = 150 * time.Millisecond
	}
	record := true
	if req.RecordScanRun != nil {
		record = *req.RecordScanRun
	}

	gammaURL := ""
	if a.root != nil {
		gammaURL = strings.TrimSpace(a.root.Hft.Polymarket.GammaURL)
	}
	if gammaURL == "" {
		gammaURL = "https://gamma-api.polymarket.com"
	}
	gc := gamma.NewWithHTTP(gammaURL, &http.Client{Timeout: 60 * time.Second})

	db, err := alpharules.OpenSQLite(dbPath)
	if err != nil {
		return http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error(), "metadata": meta}
	}
	defer db.Close()

	n, err := alpharules.RunCollect(ctx, db, gc, alpharules.CollectOptions{
		PageLimit:         limit,
		MaxEvents:         req.MaxEvents,
		SleepBetweenPages: sleep,
		SourcesConfig:     cfg,
		RecordScanRun:     record,
	})
	meta["processingTimeMs"] = time.Since(start).Milliseconds()
	if err != nil {
		return http.StatusInternalServerError, map[string]any{
			"success": false, "error": err.Error(),
			"metadata": meta, "events_stored": n,
		}
	}
	return http.StatusOK, map[string]any{
		"success": true, "events_stored": n, "sqlite_path": dbPath,
		"metadata": meta,
	}
}

type alphaRulesAgentReq struct {
	SQLitePath          string `json:"sqlite_path"`
	ConfigPath          string `json:"config_path"`
	Workers             *int   `json:"workers"`
	BatchSize           *int   `json:"batch_size"`
	Targets             string `json:"targets"`
	Reprocess           bool   `json:"reprocess"`
	DryRun              bool   `json:"dry_run"`
	MaxDescriptionChars *int   `json:"max_description_chars"`
	MaxJobs             *int   `json:"max_jobs"`
}

// RunDescriptionAgent POST body: sqlite_path (or env), config_path (or ALPHA_RULES_DESCRIPTION_AGENT_CONFIG), workers, targets, etc.
func (a *AlphaRules) RunDescriptionAgent(ctx context.Context, body []byte) (status int, out map[string]any) {
	start := time.Now()
	meta := map[string]any{
		"requestId":        fmt.Sprintf("%d", time.Now().UnixNano()),
		"timestamp":        time.Now().UTC().Format(time.RFC3339Nano),
		"processingTimeMs": time.Since(start).Milliseconds(),
	}
	if a.llm == nil {
		return http.StatusInternalServerError, map[string]any{"success": false, "error": "LLM not configured", "metadata": meta}
	}
	var req alphaRulesAgentReq
	if err := json.Unmarshal(body, &req); err != nil {
		return http.StatusBadRequest, map[string]any{"success": false, "error": "Invalid JSON", "metadata": meta}
	}
	dbPath := strings.TrimSpace(req.SQLitePath)
	if dbPath == "" {
		dbPath = alpharules.DefaultSQLitePath()
	}
	if dbPath == "" {
		return http.StatusBadRequest, map[string]any{
			"success": false,
			"error":   "sqlite_path is required (or set ALPHA_RULES_SQLITE)",
			"metadata": meta,
		}
	}
	cfgPath := strings.TrimSpace(req.ConfigPath)
	if cfgPath == "" {
		cfgPath = strings.TrimSpace(os.Getenv("ALPHA_RULES_DESCRIPTION_AGENT_CONFIG"))
	}
	if cfgPath == "" {
		return http.StatusBadRequest, map[string]any{
			"success": false,
			"error":   "config_path is required (or set ALPHA_RULES_DESCRIPTION_AGENT_CONFIG)",
			"metadata": meta,
		}
	}
	agentCfg, err := alpharules.LoadDescriptionAgentConfig(cfgPath)
	if err != nil {
		return http.StatusBadRequest, map[string]any{"success": false, "error": err.Error(), "metadata": meta}
	}

	targets := map[string]bool{}
	rawT := strings.TrimSpace(req.Targets)
	if rawT == "" {
		rawT = "events,markets"
	}
	for _, p := range strings.Split(rawT, ",") {
		s := strings.ToLower(strings.TrimSpace(p))
		if s == "" {
			continue
		}
		if s != "events" && s != "markets" {
			return http.StatusBadRequest, map[string]any{
				"success": false,
				"error":   "targets must be events and/or markets (comma-separated)",
				"metadata": meta,
			}
		}
		targets[s] = true
	}
	if len(targets) == 0 {
		return http.StatusBadRequest, map[string]any{"success": false, "error": "targets is empty", "metadata": meta}
	}

	workers := agentCfg.ParallelWorkers
	if req.Workers != nil && *req.Workers > 0 {
		workers = *req.Workers
	}
	batch := agentCfg.BatchSize
	if req.BatchSize != nil && *req.BatchSize > 0 {
		batch = *req.BatchSize
	}

	db, err := alpharules.OpenSQLite(dbPath)
	if err != nil {
		return http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error(), "metadata": meta}
	}
	defer db.Close()

	opt := alpharules.DescriptionAgentOptions{
		Workers:             workers,
		BatchSize:           batch,
		Targets:             targets,
		Reprocess:           req.Reprocess,
		MaxDescriptionChars: req.MaxDescriptionChars,
		MaxJobs:             req.MaxJobs,
		DryRun:              req.DryRun,
	}
	queued, okN, failN, err := alpharules.RunDescriptionAgent(ctx, a.llm, db, agentCfg, opt)
	meta["processingTimeMs"] = time.Since(start).Milliseconds()
	if err != nil {
		return http.StatusInternalServerError, map[string]any{
			"success": false, "error": err.Error(),
			"metadata": meta, "jobs_queued": queued, "ok": okN, "fail": failN,
		}
	}
	return http.StatusOK, map[string]any{
		"success": true, "jobs_queued": queued, "ok": okN, "fail": failN,
		"dry_run": req.DryRun, "sqlite_path": dbPath, "config_path": cfgPath,
		"metadata": meta,
	}
}
