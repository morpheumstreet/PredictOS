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
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/executor/httpapi"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/executor/paper"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/hftevents"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/httpserver"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/polymarket/gamma"
	polyws "github.com/profitlock/PredictOS/mm/polyback-mm/internal/polymarket/ws"
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
		"executor",
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

	sim := paper.NewSimulator(root, pub, wsClient, gc)
	defer sim.Close()

	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Logger, middleware.Recoverer)
	r.Get("/actuator/health", httpserver.ActuatorHealth)
	r.Handle("/metrics", httpserver.MetricsHandler())

	h := httpapi.NewPolymarket(root, sim, pub, wsClient, httpapi.NewOrderMetrics())
	h.RegisterRoutes(r)

	addr := root.Server.ExecutorAddr
	if addr == "" {
		addr = ":8080"
	}
	srv := &http.Server{Addr: addr, Handler: r, ReadHeaderTimeout: 10 * time.Second}

	go func() {
		log.Printf("executor listening on %s", addr)
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
