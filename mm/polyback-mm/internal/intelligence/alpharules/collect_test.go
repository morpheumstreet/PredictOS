package alpharules

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type stubPages struct {
	pages [][]byte
	i     int
}

func (s *stubPages) FetchEventsPage(ctx context.Context, limit, offset int, active, closed, archived bool) (json.RawMessage, error) {
	if s.i >= len(s.pages) {
		return json.RawMessage(`[]`), nil
	}
	b := s.pages[s.i]
	s.i++
	return json.RawMessage(b), nil
}

func TestRunCollect_stubGamma(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "alpha_rules.sqlite")
	db, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	future := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339Nano)
	ev := map[string]any{
		"id": "evt-test-1", "slug": "s1", "active": true, "closed": false, "archived": false,
		"endDate": future, "title": "T", "description": "rules",
		"markets": []any{
			map[string]any{
				"id": "m1", "question": "Q?", "active": true, "closed": false,
				"outcomes": []any{"Yes", "No"}, "outcomePrices": []any{0.4, 0.6},
			},
		},
	}
	batch, _ := json.Marshal([]map[string]any{ev})
	stub := &stubPages{pages: [][]byte{batch}}

	ctx := context.Background()
	n, err := RunCollect(ctx, db, stub, CollectOptions{
		PageLimit:     500,
		RecordScanRun: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("events stored: got %d want 1", n)
	}

	var cnt int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM market_outcomes WHERE market_id = 'm1'`).Scan(&cnt); err != nil {
		t.Fatal(err)
	}
	if cnt != 2 {
		t.Fatalf("outcomes: got %d want 2", cnt)
	}
}

func TestDefaultSQLitePath_env(t *testing.T) {
	const k = "ALPHA_RULES_SQLITE"
	prev, had := os.LookupEnv(k)
	defer func() {
		if had {
			_ = os.Setenv(k, prev)
		} else {
			_ = os.Unsetenv(k)
		}
	}()
	_ = os.Setenv(k, "/tmp/x.sqlite")
	if p := DefaultSQLitePath(); p != "/tmp/x.sqlite" {
		t.Fatalf("got %q", p)
	}
}
