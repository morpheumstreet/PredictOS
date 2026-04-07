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
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/httpserver"
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

	r := chi.NewRouter()
	r.Use(middleware.Logger, middleware.Recoverer)
	r.Get("/actuator/health", httpserver.ActuatorHealth)
	r.Handle("/metrics", httpserver.MetricsHandler())
	httpserver.MountClientConfig(r, root)

	r.Route("/api/analytics", func(r chi.Router) {
		r.Get("/status", func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, map[string]any{
				"app":           "polyback-analytics",
				"datasourceUrl": root.Analytics.ClickhouseDSN,
				"eventsTable":   "analytics_events",
			})
		})
		r.Get("/events", func(w http.ResponseWriter, req *http.Request) {
			// Stub: port JdbcAnalyticsEventRepository to clickhouse-go against this DSN when needed.
			_ = req.URL.Query().Get("type")
			writeJSON(w, []any{})
		})
	})

	addr := root.Server.AnalyticsAddr
	if addr == "" {
		addr = ":8082"
	}
	srv := &http.Server{Addr: addr, Handler: r, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		log.Printf("analytics listening on %s (events endpoint returns empty until CH repo is ported)", addr)
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

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
