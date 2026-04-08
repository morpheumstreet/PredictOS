package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/imdario/mergo"
	"gopkg.in/yaml.v3"
)

type Root struct {
	Hft            Hft          `yaml:"hft"`
	Executor       ExecutorCfg  `yaml:"executor"`
	Kafka          KafkaCfg     `yaml:"kafka"`
	Server         ServerCfg    `yaml:"server"`
	Infrastructure InfraRoot    `yaml:"infrastructure"`
	Ingestor       IngestorCfg  `yaml:"ingestor"`
	Analytics      AnalyticsCfg `yaml:"analytics"`
}

type Hft struct {
	Mode        string         `yaml:"mode"`
	Events      EventsCfg      `yaml:"events"`
	Executor    HftExecutor    `yaml:"executor"`
	Polymarket  PolymarketCfg  `yaml:"polymarket"`
	Limitless   LimitlessCfg   `yaml:"limitless"`
	PredictFun  PredictFunCfg  `yaml:"predict_fun"`
	KalshiDFlow KalshiDFlowCfg `yaml:"kalshi_dflow"`
	Strategy    StrategyCfg    `yaml:"strategy"`
	Risk        RiskCfg        `yaml:"risk"`
}

// LimitlessCfg is optional; secrets should stay empty in committed configs (use .env / private overrides).
// Env fallback when YAML empty: LIMITLESS_API_KEY, LIMITLESS_WALLET_ADDRESS.
type LimitlessCfg struct {
	BaseURL       string `yaml:"base_url"`
	APIKey        string `yaml:"api_key"`
	WalletAddress string `yaml:"wallet_address"`
}

// PredictFunCfg is optional (Predict.fun REST). Env when YAML empty: PREDICT_FUN_API_KEY, PREDICT_FUN_PRIVATE_KEY, PREDICT_FUN_BASE_URL.
type PredictFunCfg struct {
	BaseURL    string `yaml:"base_url"`
	APIKey     string `yaml:"api_key"`
	PrivateKey string `yaml:"private_key"`
}

// KalshiDFlowCfg is optional (Kalshi-shaped markets via DFlow). Env when YAML empty: DFLOW_API_KEY, DFLOW_LIVE_EVENT_TICKER, DFLOW_BASE_URL.
type KalshiDFlowCfg struct {
	BaseURL     string `yaml:"base_url"`
	APIKey      string `yaml:"api_key"`
	EventTicker string `yaml:"event_ticker"`
}

type EventsCfg struct {
	Enabled                       bool   `yaml:"enabled"`
	Topic                         string `yaml:"topic"`
	MarketWsTobMinIntervalMillis  int64  `yaml:"market_ws_tob_min_interval_millis"`
	MarketWsSnapshotPublishMillis int64  `yaml:"market_ws_snapshot_publish_millis"`
	MarketWsCachePublishOnStart   bool   `yaml:"market_ws_cache_publish_on_start"`
}

type HftExecutor struct {
	BaseURL     string `yaml:"base_url"`
	SendLiveAck bool   `yaml:"send_live_ack"`
}

type PolymarketCfg struct {
	GammaURL                       string  `yaml:"gamma_url"`
	ClobRestURL                    string  `yaml:"clob_rest_url"`
	ClobWsURL                      string  `yaml:"clob_ws_url"`
	ChainID                        int     `yaml:"chain_id"`
	MarketWsEnabled                bool    `yaml:"market_ws_enabled"`
	MarketWsStaleTimeoutMillis     int64   `yaml:"market_ws_stale_timeout_millis"`
	MarketWsReconnectBackoffMillis int64   `yaml:"market_ws_reconnect_backoff_millis"`
	MarketWsCachePath              string  `yaml:"market_ws_cache_path"`
	MarketWsCacheFlushMillis       int64   `yaml:"market_ws_cache_flush_millis"`
	Auth                           AuthCfg `yaml:"auth"`
	// Relayer/builder credentials (Polymarket CLOB). Env fallback when YAML empty: POLY_RELAYER_API_KEY, POLY_BUILDER_*.
	RelayerAPIKey     string       `yaml:"relayer_api_key"`
	BuilderAPIKey     string       `yaml:"builder_api_key"`
	BuilderSecret     string       `yaml:"builder_secret"`
	BuilderPassphrase string       `yaml:"builder_passphrase"`
	EventFeed         EventFeedCfg `yaml:"event_feed"`
}

// EventFeedCfg optional HTTP poll; body hash change triggers OnAlert in strategy (cancel-all).
type EventFeedCfg struct {
	Enabled            bool   `yaml:"enabled"`
	PollURL            string `yaml:"poll_url"`
	PollIntervalMillis int    `yaml:"poll_interval_millis"`
}

type AuthCfg struct {
	PrivateKey string `yaml:"private_key"`
}

type StrategyCfg struct {
	Gabagool    GabagoolCfg    `yaml:"gabagool"`
	MarketMaker MarketMakerCfg `yaml:"market_maker"`
}

// MarketMakerCfg enables study.md-style quoting + toxicity (see internal/strategy/quoting, toxicity).
type MarketMakerCfg struct {
	Enabled bool `yaml:"enabled"`

	TradeWindowMillis int `yaml:"trade_window_millis"`
	BurstTradeCount   int `yaml:"burst_trade_count"`

	ImpactSpreadMultiple float64 `yaml:"impact_spread_multiple"`

	LiquidityDropRatio float64 `yaml:"liquidity_drop_ratio"`

	ToxicityPenaltyMax  float64 `yaml:"toxicity_penalty_max"`
	ToxicityUnsafeBurst int     `yaml:"toxicity_unsafe_burst"`

	BaseSpread     float64 `yaml:"base_spread"`
	VolSpreadBonus float64 `yaml:"vol_spread_bonus"`

	// EWMA dynamic spread (0 scale = disabled). Addon is added to base+vol_spread_bonus before half-spread.
	EwmaVolLambda      float64 `yaml:"ewma_vol_lambda"`
	EwmaVolSpreadScale float64 `yaml:"ewma_vol_spread_scale"`
	EwmaVolSpreadMax   float64 `yaml:"ewma_vol_spread_max"`

	ImbalanceSkewScale float64 `yaml:"imbalance_skew_scale"`

	NoiseSigma    float64 `yaml:"noise_sigma"`
	NoiseMaxTicks int     `yaml:"noise_max_ticks"`

	// Depth pause: stop quoting a side when top size vs EMA indicates sudden thinning.
	DepthPauseEnabled   bool    `yaml:"depth_pause_enabled"`
	DepthPauseDropRatio float64 `yaml:"depth_pause_drop_ratio"` // 0 = reuse liquidity_drop_ratio
	DepthPauseFallback  bool    `yaml:"depth_pause_fallback"`   // true = fall back to legacy when bid paused

	// VPIN-style imbalance on recent trades (requires size + side on trades).
	VpinEnabled            bool    `yaml:"vpin_enabled"`
	VpinMinTrades          int     `yaml:"vpin_min_trades"`
	VpinImbalanceThreshold float64 `yaml:"vpin_imbalance_threshold"` // 0–1, mark unsafe if exceeded

	// Push-driven evaluate (book listener); debounce per asset to avoid thrash.
	PushRefreshEnabled        bool `yaml:"push_refresh_enabled"`
	PushRefreshDebounceMillis int  `yaml:"push_refresh_debounce_millis"`

	// TWAP: cap each quote size; remainder on later ticks/push (no in-process queue).
	TwapEnabled        bool    `yaml:"twap_enabled"`
	TwapMaxChunkShares float64 `yaml:"twap_max_chunk_shares"`
}

type GabagoolCfg struct {
	Enabled                              bool    `yaml:"enabled"`
	RefreshMillis                        int64   `yaml:"refresh_millis"`
	MinReplaceMillis                     int64   `yaml:"min_replace_millis"`
	MinSecondsToEnd                      int64   `yaml:"min_seconds_to_end"`
	MaxSecondsToEnd                      int64   `yaml:"max_seconds_to_end"`
	ImproveTicks                         int     `yaml:"improve_ticks"`
	QuoteSize                            float64 `yaml:"quote_size"`
	QuoteSizeBankrollFraction            float64 `yaml:"quote_size_bankroll_fraction"`
	BankrollMode                         string  `yaml:"bankroll_mode"`
	BankrollUsd                          float64 `yaml:"bankroll_usd"`
	BankrollRefreshMillis                int64   `yaml:"bankroll_refresh_millis"`
	BankrollSmoothingAlpha               float64 `yaml:"bankroll_smoothing_alpha"`
	BankrollMinThreshold                 float64 `yaml:"bankroll_min_threshold"`
	BankrollTradingFraction              float64 `yaml:"bankroll_trading_fraction"`
	MaxOrderBankrollFraction             float64 `yaml:"max_order_bankroll_fraction"`
	MaxTotalBankrollFraction             float64 `yaml:"max_total_bankroll_fraction"`
	CompleteSetMinEdge                   float64 `yaml:"complete_set_min_edge"`
	CompleteSetMaxSkewTicks              int     `yaml:"complete_set_max_skew_ticks"`
	CompleteSetImbalanceSharesForMaxSkew float64 `yaml:"complete_set_imbalance_shares_for_max_skew"`
	CompleteSetTopUpEnabled              bool    `yaml:"complete_set_top_up_enabled"`
	CompleteSetTopUpSecondsToEnd         int64   `yaml:"complete_set_top_up_seconds_to_end"`
	CompleteSetTopUpMinShares            float64 `yaml:"complete_set_top_up_min_shares"`
	CompleteSetFastTopUpEnabled          bool    `yaml:"complete_set_fast_top_up_enabled"`
	CompleteSetFastTopUpMinShares        float64 `yaml:"complete_set_fast_top_up_min_shares"`
	CompleteSetFastTopUpMinSecAfterFill  int64   `yaml:"complete_set_fast_top_up_min_seconds_after_fill"`
	CompleteSetFastTopUpMaxSecAfterFill  int64   `yaml:"complete_set_fast_top_up_max_seconds_after_fill"`
	CompleteSetFastTopUpCooldownMillis   int64   `yaml:"complete_set_fast_top_up_cooldown_millis"`
	CompleteSetFastTopUpMinEdge          float64 `yaml:"complete_set_fast_top_up_min_edge"`
	TakerModeEnabled                     bool    `yaml:"taker_mode_enabled"`
	TakerModeMaxEdge                     float64 `yaml:"taker_mode_max_edge"`
	TakerModeMaxSpread                   float64 `yaml:"taker_mode_max_spread"`
}

type RiskCfg struct {
	KillSwitch          bool    `yaml:"kill_switch"`
	MaxOrderNotionalUsd float64 `yaml:"max_order_notional_usd"`
}

type ExecutorCfg struct {
	Sim SimCfg `yaml:"sim"`
}

type SimCfg struct {
	Enabled                               bool    `yaml:"enabled"`
	FillsEnabled                          bool    `yaml:"fills_enabled"`
	FillPollMillis                        int64   `yaml:"fill_poll_millis"`
	MakerFillProbabilityPerPoll           float64 `yaml:"maker_fill_probability_per_poll"`
	MakerFillProbabilityMultiplierPerTick float64 `yaml:"maker_fill_probability_multiplier_per_tick"`
	MakerFillProbabilityMaxPerPoll        float64 `yaml:"maker_fill_probability_max_per_poll"`
	MakerFillFractionOfRemaining          float64 `yaml:"maker_fill_fraction_of_remaining"`
	Username                              string  `yaml:"username"`
	ProxyAddress                          string  `yaml:"proxy_address"`
}

type KafkaCfg struct {
	Brokers []string `yaml:"brokers"`
}

type ServerCfg struct {
	// CorsAllowedOrigins lists browser origins allowed for cross-origin requests (e.g. PredictOS terminal).
	// When empty, CORS middleware is not applied.
	CorsAllowedOrigins []string `yaml:"cors_allowed_origins"`
	// PublicAPIBaseURL is the canonical HTTP base for browsers and the terminal (no trailing path).
	PublicAPIBaseURL string `yaml:"public_api_base_url"`
	// ClientConfigEnabled when false disables GET /api/v1/config/client on this process. Nil/absent means enabled.
	ClientConfigEnabled *bool `yaml:"client_config_enabled"`
	ExecutorAddr        string `yaml:"executor_addr"`
	StrategyAddr        string `yaml:"strategy_addr"`
	AnalyticsAddr       string `yaml:"analytics_addr"`
	IngestorAddr        string `yaml:"ingestor_addr"`
	InfrastructureAddr string `yaml:"infrastructure_addr"`
	// IntelligenceAddr is the listen address for the Polyback Intelligence HTTP service (agents, proxies).
	IntelligenceAddr string `yaml:"intelligence_addr"`
}

type InfraRoot struct {
	StartupTimeoutSeconds      int          `yaml:"startup_timeout_seconds"`
	HealthCheckIntervalSeconds int          `yaml:"health_check_interval_seconds"`
	PolybotHome                string       `yaml:"polybot_home"`
	Stacks                     []InfraStack `yaml:"stacks"`
}

type InfraStack struct {
	Name             string `yaml:"name"`
	FilePath         string `yaml:"file_path"`
	ProjectName      string `yaml:"project_name"`
	ExpectedServices int    `yaml:"expected_services"`
	StartupOrder     int    `yaml:"startup_order"`
}

type IngestorCfg struct {
	Polymarket PolymarketIngestor `yaml:"polymarket"`
	Polling    struct {
		Enabled bool `yaml:"enabled"`
	} `yaml:"polling"`
	Clickhouse ClickhouseHTTP `yaml:"clickhouse"`
}

type PolymarketIngestor struct {
	Username       string `yaml:"username"`
	ProxyAddress   string `yaml:"proxy_address"`
	DataAPIBaseURL string `yaml:"data_api_base_url"`
}

type ClickhouseHTTP struct {
	BaseURL  string `yaml:"base_url"`
	Database string `yaml:"database"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

type AnalyticsCfg struct {
	ClickhouseDSN string `yaml:"clickhouse_dsn"`
}

// Load reads the base YAML (e.g. configs/develop.yaml), then merges overlays from the same
// directory in order: real.testing.yml (gitignored; copy from real.testing.template.yml), then
// real.yml (gitignored, optional extra overrides). Missing overlay files are skipped.
func Load(path string) (*Root, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var r Root
	if err := yaml.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	if err := mergeConfigOverlays(&r, path, []string{"real.testing.yml", "real.yml"}); err != nil {
		return nil, err
	}
	if r.Hft.Events.Topic == "" {
		r.Hft.Events.Topic = "polybot.events"
	}
	applyPolymarketEnv(&r.Hft.Polymarket)
	applyLimitlessEnv(&r.Hft.Limitless)
	applyPredictFunEnv(&r.Hft.PredictFun)
	applyKalshiDFlowEnv(&r.Hft.KalshiDFlow)
	normalizeHftExecutorBaseURLFromPublicAPI(&r)
	return &r, nil
}

// normalizeHftExecutorBaseURLFromPublicAPI sets hft.executor.base_url from server.public_api_base_url when the former is empty (DRY).
func normalizeHftExecutorBaseURLFromPublicAPI(r *Root) {
	pub := strings.TrimSpace(r.Server.PublicAPIBaseURL)
	if pub == "" {
		return
	}
	if strings.TrimSpace(r.Hft.Executor.BaseURL) == "" {
		r.Hft.Executor.BaseURL = pub
	}
}

func applyPolymarketEnv(p *PolymarketCfg) {
	if p.RelayerAPIKey == "" {
		p.RelayerAPIKey = os.Getenv("POLY_RELAYER_API_KEY")
	}
	if p.BuilderAPIKey == "" {
		p.BuilderAPIKey = os.Getenv("POLY_BUILDER_API_KEY")
	}
	if p.BuilderSecret == "" {
		p.BuilderSecret = os.Getenv("POLY_BUILDER_SECRET")
	}
	if p.BuilderPassphrase == "" {
		p.BuilderPassphrase = os.Getenv("POLY_BUILDER_PASSPHRASE")
	}
}

func applyLimitlessEnv(l *LimitlessCfg) {
	if l.APIKey == "" {
		l.APIKey = os.Getenv("LIMITLESS_API_KEY")
	}
	if l.WalletAddress == "" {
		l.WalletAddress = os.Getenv("LIMITLESS_WALLET_ADDRESS")
	}
}

func applyPredictFunEnv(p *PredictFunCfg) {
	if p.BaseURL == "" {
		p.BaseURL = os.Getenv("PREDICT_FUN_BASE_URL")
	}
	if p.APIKey == "" {
		p.APIKey = os.Getenv("PREDICT_FUN_API_KEY")
	}
	if p.PrivateKey == "" {
		p.PrivateKey = os.Getenv("PREDICT_FUN_PRIVATE_KEY")
	}
}

func applyKalshiDFlowEnv(k *KalshiDFlowCfg) {
	if k.BaseURL == "" {
		k.BaseURL = os.Getenv("DFLOW_BASE_URL")
	}
	if k.APIKey == "" {
		k.APIKey = os.Getenv("DFLOW_API_KEY")
	}
	if k.EventTicker == "" {
		k.EventTicker = os.Getenv("DFLOW_LIVE_EVENT_TICKER")
	}
}

func mergeConfigOverlays(r *Root, baseConfigPath string, overlayFilenames []string) error {
	absBase, err := filepath.Abs(baseConfigPath)
	if err != nil {
		return err
	}
	dir := filepath.Dir(baseConfigPath)
	for _, name := range overlayFilenames {
		p := filepath.Join(dir, name)
		absOver, err := filepath.Abs(p)
		if err != nil {
			return err
		}
		if absBase == absOver {
			continue
		}
		ob, err := os.ReadFile(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("read %s: %w", p, err)
		}
		var over Root
		if err := yaml.Unmarshal(ob, &over); err != nil {
			return fmt.Errorf("parse %s: %w", p, err)
		}
		if err := mergo.Merge(r, &over, mergo.WithOverride); err != nil {
			return fmt.Errorf("merge %s: %w", p, err)
		}
	}
	return nil
}

// DefaultPath returns POLYBACK_CONFIG, first CLI arg handled in main, or configs/develop.yaml.
func DefaultPath() string {
	if p := os.Getenv("POLYBACK_CONFIG"); p != "" {
		return p
	}
	return "configs/develop.yaml"
}

// ComposeBase is the directory used to resolve relative infrastructure.stack file_path values.
// Order: POLYBOT_HOME, infrastructure.polybot_home, then the repo root (parent of configs/ when the
// loaded config file lives under .../configs/).
func ComposeBase(r *Root, configFilePath string) (string, error) {
	if v := os.Getenv("POLYBOT_HOME"); strings.TrimSpace(v) != "" {
		return filepath.Clean(v), nil
	}
	if r != nil && strings.TrimSpace(r.Infrastructure.PolybotHome) != "" {
		return filepath.Clean(r.Infrastructure.PolybotHome), nil
	}
	if configFilePath == "" {
		return "", fmt.Errorf("infrastructure: set POLYBOT_HOME or load config from configs/*.yaml")
	}
	abs, err := filepath.Abs(configFilePath)
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(abs)
	if filepath.Base(dir) == "configs" {
		return filepath.Clean(filepath.Join(dir, "..")), nil
	}
	return filepath.Clean(dir), nil
}

func (s InfraStack) ResolvedComposePath(composeBase string) (string, error) {
	p := s.FilePath
	if filepath.IsAbs(p) {
		return p, nil
	}
	if composeBase == "" {
		return "", fmt.Errorf("compose base not set for stack %q", s.Name)
	}
	return filepath.Join(composeBase, p), nil
}

func BrokerList(r *Root) []string {
	out := make([]string, 0, len(r.Kafka.Brokers))
	for _, b := range r.Kafka.Brokers {
		if t := strings.TrimSpace(b); t != "" {
			out = append(out, t)
		}
	}
	return out
}
