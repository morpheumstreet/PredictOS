package httpserver

import (
	"encoding/json"
	"net/http"
)

type Health struct {
	Status string `json:"status"`
}

func ActuatorHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(Health{Status: "UP"})
}
