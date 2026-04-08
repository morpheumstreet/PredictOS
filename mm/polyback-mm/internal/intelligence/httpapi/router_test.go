package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/intelligence/app"
)

func TestIntelligencePing_OK(t *testing.T) {
	r := chi.NewRouter()
	r.Route("/api/intelligence", func(sub chi.Router) {
		Mount(sub, app.NewDeps(nil))
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	res, err := http.Post(srv.URL+"/api/intelligence/ping", "application/json", bytes.NewReader([]byte("{}")))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("status %d body %s", res.StatusCode, b)
	}
	var out map[string]any
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out["ok"] != true {
		t.Fatalf("expected ok true, got %#v", out)
	}
}

func TestIntelligenceGetEvents_InvalidJSON(t *testing.T) {
	r := chi.NewRouter()
	r.Route("/api/intelligence", func(sub chi.Router) {
		Mount(sub, app.NewDeps(nil))
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	res, err := http.Post(srv.URL+"/api/intelligence/get-events", "application/json", bytes.NewReader([]byte("not-json")))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("want 400, got %d %s", res.StatusCode, b)
	}
}
