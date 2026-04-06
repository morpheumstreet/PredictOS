package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Root struct {
	Hft             Hft             `yaml:"hft"`
	Executor        ExecutorCfg     `yaml:"executor"`
	Kafka           KafkaCfg        `yaml:"kafka"`
	Server          ServerCfg       `yaml:"server"`
	Infrastructure  InfraRoot       `yaml:"infrastructure"`
	Ingestor        IngestorCfg     `yaml:"ingestor"`
	Analytics       AnalyticsCfg    `yaml:"analytics"`
}

type Hft struct {
	Mode       string           `yaml:"mode"`
	Events     EventsCfg        `yaml:"events"`
	Executor   HftExecutor      `yaml:"executor"`
	Polymarket PolymarketCfg    `yaml:"polymarket"`
	Strategy   StrategyCfg      `yaml:"strategy"`
	Risk       RiskCfg          `yaml:"risk"`
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
	GammaURL                      string    `yaml:"gamma_url"`
	ClobRestURL                   string    `yaml:"clob_rest_url"`
	ClobWsURL                     string    `yaml:"clob_ws_url"`
	ChainID                       int       `yaml:"chain_id"`
	MarketWsEnabled               bool      `yaml:"market_ws_enabled"`
	MarketWsStaleTimeoutMillis    int64     `yaml:"market_ws_stale_timeout_millis"`
	MarketWsReconnectBackoffMillis int64    `yaml:"market_ws_reconnect_backoff_millis"`
	MarketWsCachePath             string    `yaml:"market_ws_cache_path"`
	MarketWsCacheFlushMillis      int64     `yaml:"market_ws_cache_flush_millis"`
	Auth                          AuthCfg   `yaml:"auth"`
}

type AuthCfg struct {
	PrivateKey string `yaml:"private_key"`
}

type StrategyCfg struct {
	Gabagool GabagoolCfg `yaml:"gabagool"`
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
	Enabled                           bool    `yaml:"enabled"`
	FillsEnabled                      bool    `yaml:"fills_enabled"`
	FillPollMillis                    int64   `yaml:"fill_poll_millis"`
	MakerFillProbabilityPerPoll       float64 `yaml:"maker_fill_probability_per_poll"`
	MakerFillProbabilityMultiplierPerTick float64 `yaml:"maker_fill_probability_multiplier_per_tick"`
	MakerFillProbabilityMaxPerPoll  float64 `yaml:"maker_fill_probability_max_per_poll"`
	MakerFillFractionOfRemaining      float64 `yaml:"maker_fill_fraction_of_remaining"`
	Username                          string  `yaml:"username"`
	ProxyAddress                      string  `yaml:"proxy_address"`
}

type KafkaCfg struct {
	Brokers []string `yaml:"brokers"`
}

type ServerCfg struct {
	ExecutorAddr        string `yaml:"executor_addr"`
	StrategyAddr        string `yaml:"strategy_addr"`
	AnalyticsAddr       string `yaml:"analytics_addr"`
	IngestorAddr        string `yaml:"ingestor_addr"`
	InfrastructureAddr  string `yaml:"infrastructure_addr"`
}

type InfraRoot struct {
	StartupTimeoutSeconds       int           `yaml:"startup_timeout_seconds"`
	HealthCheckIntervalSeconds  int           `yaml:"health_check_interval_seconds"`
	PolybotHome                 string        `yaml:"polybot_home"`
	Stacks                      []InfraStack  `yaml:"stacks"`
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
	Username        string `yaml:"username"`
	ProxyAddress    string `yaml:"proxy_address"`
	DataAPIBaseURL  string `yaml:"data_api_base_url"`
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

func Load(path string) (*Root, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var r Root
	if err := yaml.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	if r.Hft.Events.Topic == "" {
		r.Hft.Events.Topic = "polybot.events"
	}
	return &r, nil
}

// DefaultPath returns POLYBACK_CONFIG, first CLI arg handled in main, or configs/develop.yaml.
func DefaultPath() string {
	if p := os.Getenv("POLYBACK_CONFIG"); p != "" {
		return p
	}
	return "configs/develop.yaml"
}

// ResolvePolybotHome sets infrastructure paths relative to polybot-main checkout.
func (r *Root) ResolvePolybotHome() string {
	home := r.Infrastructure.PolybotHome
	if home != "" {
		return home
	}
	if v := os.Getenv("POLYBOT_HOME"); v != "" {
		return v
	}
	// sibling mm/polybot-main from mm/polyback-mm
	if abs, err := filepath.Abs(".."); err == nil {
		candidate := filepath.Join(abs, "polybot-main")
		if st, err := os.Stat(filepath.Join(candidate, "docker-compose.analytics.yaml")); err == nil && !st.IsDir() {
			return candidate
		}
	}
	return ""
}

func (s InfraStack) ResolvedComposePath(polybotHome string) (string, error) {
	p := s.FilePath
	if filepath.IsAbs(p) {
		return p, nil
	}
	if polybotHome == "" {
		return "", fmt.Errorf("polybot home not set for stack %q", s.Name)
	}
	return filepath.Join(polybotHome, p), nil
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
