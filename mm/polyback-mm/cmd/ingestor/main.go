package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/config"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/hftevents"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/httpserver"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/platforms/polymarket/gamma"
	polyws "github.com/profitlock/PredictOS/mm/polyback-mm/internal/platforms/polymarket/ws"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/wiring"
)

func main() {
	cfgPath := config.DefaultPath()
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}
	if !filepath.IsAbs(cfgPath) {
		if abs, err := filepath.Abs(cfgPath); err == nil {
			cfgPath = abs
		}
	}
	root, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	pub, err := hftevents.NewPublisherFromBrokers(
		config.BrokerList(root),
		root.Hft.Events.Topic,
		"ingestor",
		root.Hft.Events.Enabled,
	)
	if err != nil {
		log.Fatalf("kafka: %v", err)
	}
	defer pub.Close()

	gc := gamma.New(root.Hft.Polymarket.GammaURL)
	wsClient := polyws.NewClobClient(
		root.Hft.Polymarket.ClobWsURL,
		root.Hft.Polymarket.MarketWsEnabled,
		wiring.TOBFromPublisher(pub),
		root.Hft.Events.MarketWsTobMinIntervalMillis,
		root.Hft.Events.MarketWsSnapshotPublishMillis,
	)
	wsClient.StartBackground()
	defer wsClient.Close()

	r := chi.NewRouter()
	r.Use(middleware.Logger, middleware.Recoverer)
	r.Get("/actuator/health", httpserver.ActuatorHealth)
	r.Handle("/metrics", httpserver.MetricsHandler())
	r.Get("/api/ingestor/status", func(w http.ResponseWriter, _ *http.Request) {
		st := map[string]any{
			"app":                "polyback-ingestor",
			"polymarketUsername": root.Ingestor.Polymarket.Username,
			"proxyAddress":       root.Ingestor.Polymarket.ProxyAddress,
			"dataApiBaseUrl":     root.Ingestor.Polymarket.DataAPIBaseURL,
			"pollingEnabled":     root.Ingestor.Polling.Enabled,
			"marketWsStarted":    wsClient.IsConnected(),
			"subscribedAssets":   wsClient.SubscribedAssetCount(),
			"topOfBookCount":     wsClient.TopOfBookCount(),
			"kafkaEnabled":       root.Hft.Events.Enabled,
			"kafkaTopic":         root.Hft.Events.Topic,
			"gammaBaseUrl":       root.Hft.Polymarket.GammaURL,
			"clickhouseBaseUrl":  root.Ingestor.Clickhouse.BaseURL,
		}
		_ = gc
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(st)
	})

	addr := root.Server.IngestorAddr
	if addr == "" {
		addr = ":8083"
	}
	srv := &http.Server{Addr: addr, Handler: r, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		log.Printf("ingestor listening on %s (polling pipelines are stubs; wire Java parity as needed)", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
