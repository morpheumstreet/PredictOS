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
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/intelligence/app"
	intelhttp "github.com/profitlock/PredictOS/mm/polyback-mm/internal/intelligence/httpapi"
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

	deps := app.NewDeps(root)

	r := chi.NewRouter()
	httpserver.UseCORSIfConfigured(r, root.Server.CorsAllowedOrigins)
	r.Use(middleware.Logger, middleware.Recoverer)
	r.Get("/actuator/health", httpserver.ActuatorHealth)
	r.Handle("/metrics", httpserver.MetricsHandler())
	httpserver.MountClientConfig(r, root)

	r.Route("/api/intelligence", func(sub chi.Router) {
		intelhttp.UseIntelligenceCORS(sub, root.Server.CorsAllowedOrigins)
		intelhttp.Mount(sub, deps)
	})

	addr := root.Server.IntelligenceAddr
	if addr == "" {
		addr = ":8085"
	}
	srv := &http.Server{Addr: addr, Handler: r, ReadHeaderTimeout: 30 * time.Second}
	go func() {
		log.Printf("intelligence listening on %s", addr)
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
