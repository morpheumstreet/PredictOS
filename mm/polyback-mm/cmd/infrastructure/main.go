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
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/infra"
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
	mgr, err := infra.NewManager(root)
	if err != nil {
		log.Fatalf("infra: %v", err)
	}
	log.Printf("infrastructure: polybot home %q", mgr.PolybotHome())
	log.Println("infrastructure: launching compose stacks...")
	if err := mgr.StartAll(); err != nil {
		log.Fatalf("infrastructure start: %v", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger, middleware.Recoverer)
	r.Get("/actuator/health", httpserver.ActuatorHealth)
	r.Handle("/metrics", httpserver.MetricsHandler())

	r.Route("/api/infrastructure", func(r chi.Router) {
		r.Get("/status", func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, mgr.Status())
		})
		r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
			st := mgr.Status()
			code := http.StatusOK
			if st.OverallHealth != "HEALTHY" {
				code = http.StatusServiceUnavailable
			}
			w.WriteHeader(code)
			writeJSON(w, map[string]any{
				"status":        map[bool]string{true: "UP", false: "DOWN"}[st.OverallHealth == "HEALTHY"],
				"overallHealth": st.OverallHealth,
				"managed":       st.Managed,
				"stacks":        st.Stacks,
			})
		})
		r.Get("/links", func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, map[string]any{
				"analytics": map[string]any{
					"clickhouse_http":   "http://localhost:8123",
					"clickhouse_native": "tcp://localhost:9000",
					"redpanda_kafka":    "localhost:9092",
					"redpanda_admin":    "http://localhost:9644",
				},
				"monitoring": map[string]any{
					"grafana":      "http://localhost:3000",
					"prometheus":   "http://localhost:9090",
					"alertmanager": "http://localhost:9093",
				},
			})
		})
		r.Post("/restart", func(w http.ResponseWriter, _ *http.Request) {
			mgr.StopAll()
			time.Sleep(2 * time.Second)
			if err := mgr.StartAll(); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				writeJSON(w, map[string]string{"status": "error", "message": err.Error()})
				return
			}
			writeJSON(w, map[string]string{"status": "success", "message": "Infrastructure stacks restarted"})
		})
	})

	addr := root.Server.InfrastructureAddr
	if addr == "" {
		addr = ":8084"
	}
	srv := &http.Server{Addr: addr, Handler: r, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		log.Printf("infrastructure listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	mgr.StopAll()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
