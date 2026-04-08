package httpapi

import (
	"encoding/json"
	"net/http"
)

// WriteJSON writes v as JSON with optional status (default 200).
func WriteJSON(w http.ResponseWriter, status int, v any) {
	if status == 0 {
		status = http.StatusOK
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
