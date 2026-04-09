package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/profitlock/PredictOS/mm/irys-upload/internal/config"
	"github.com/profitlock/PredictOS/mm/irys-upload/internal/httpserver"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := httpserver.New(cfg)
	log.Printf("irys-upload listening on %s (environment=%s)", addr, cfg.Environment)
	if err := http.ListenAndServe(addr, srv.Router()); err != nil {
		log.Fatal(err)
	}
}
