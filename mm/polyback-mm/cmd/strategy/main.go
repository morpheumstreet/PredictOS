package main

import (
	"context"
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
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/polymarket/gamma"
	polyws "github.com/profitlock/PredictOS/mm/polyback-mm/internal/polymarket/ws"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/strategy/executorclient"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/wiring"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/strategy/gabagool"
	strategyhttp "github.com/profitlock/PredictOS/mm/polyback-mm/internal/strategy/httpapi"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/strategy/metrics"
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
		"strategy",
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

	ex := executorclient.New(root)
	met := metrics.New()
	runID := gabagool.RandomRunID()
	om := gabagool.NewOrderManager(ex, pub, runID)
	disc := gabagool.NewDiscovery(root, gc)
	eng := gabagool.NewEngine(root, wsClient, ex, disc, met, om)
	gcfg := &root.Hft.Strategy.Gabagool
	eng.Start()
	defer eng.Stop()

	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Logger, middleware.Recoverer)
	r.Get("/actuator/health", httpserver.ActuatorHealth)
	r.Handle("/metrics", httpserver.MetricsHandler())
	strategyhttp.Mount(r, eng, gcfg.Enabled)

	addr := root.Server.StrategyAddr
	if addr == "" {
		addr = ":8081"
	}
	srv := &http.Server{Addr: addr, Handler: r, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		log.Printf("strategy listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
