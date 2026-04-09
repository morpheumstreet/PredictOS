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
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/httpserver"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/tracker"
	trackerhttp "github.com/profitlock/PredictOS/mm/polyback-mm/internal/tracker/httpapi"
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

	svc := tracker.NewService(root, nil)

	r := chi.NewRouter()
	httpserver.UseCORSIfConfigured(r, root.Server.CorsAllowedOrigins)
	r.Use(middleware.Logger, middleware.Recoverer)
	r.Get("/actuator/health", httpserver.ActuatorHealth)
	r.Handle("/metrics", httpserver.MetricsHandler())
	httpserver.MountClientConfig(r, root)

	r.Route("/api/tracker", func(sub chi.Router) {
		trackerhttp.UseTrackerCORS(sub, root.Server.CorsAllowedOrigins)
		trackerhttp.Mount(sub, &trackerhttp.Deps{Svc: svc})
	})

	addr := root.Server.TrackerAddr
	if addr == "" {
		addr = ":8086"
	}
	srv := &http.Server{Addr: addr, Handler: r, ReadHeaderTimeout: 30 * time.Second}
	go func() {
		log.Printf("tracker listening on %s", addr)
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
