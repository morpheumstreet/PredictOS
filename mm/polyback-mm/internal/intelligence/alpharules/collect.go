package alpharules

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"strings"
	"time"
)

// PageFetcher fetches one Gamma /events page (see gamma.Client.FetchEventsPage).
type PageFetcher interface {
	FetchEventsPage(ctx context.Context, limit, offset int, active, closed, archived bool) (json.RawMessage, error)
}

// CollectOptions configures a catalog scan (mirrors collect.py CLI flags).
type CollectOptions struct {
	PageLimit         int
	MaxEvents         *int
	SleepBetweenPages time.Duration
	SourcesConfig     SourcesConfig
	RecordScanRun     bool
}

// UTCNowISO returns RFC3339 timestamp with sub-second stripped (like Python utc_now_iso).
func UTCNowISO() string {
	return time.Now().UTC().Truncate(time.Second).Format(time.RFC3339)
}

func fetchEventsPageWithRetry(ctx context.Context, g PageFetcher, limit, offset int) (json.RawMessage, error) {
	var last error
	for attempt := 0; attempt < 5; attempt++ {
		raw, err := g.FetchEventsPage(ctx, limit, offset, true, false, false)
		if err == nil {
			return raw, nil
		}
		last = err
		if attempt < 4 {
			d := time.Duration(500*(1<<attempt)) * time.Millisecond
			select {
			case <-time.After(d):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}
	return nil, last
}

// RunCollect pages Gamma active events, upserts eligible rows into the SQLite catalog.
func RunCollect(ctx context.Context, db *sql.DB, g PageFetcher, opt CollectOptions) (total int, err error) {
	if err = InitDB(ctx, db); err != nil {
		return 0, err
	}
	apiPageLimit := opt.PageLimit
	if apiPageLimit < 1 {
		apiPageLimit = 500
	}
	if apiPageLimit > 500 {
		apiPageLimit = 500
	}

	var runID int64
	if opt.RecordScanRun {
		runID, err = BeginScanRun(ctx, db, UTCNowISO())
		if err != nil {
			return 0, err
		}
	}

	offset := 0
	finishRun := func(ok bool, errMsg *string) {
		if !opt.RecordScanRun || runID == 0 {
			return
		}
		_ = FinishScanRun(ctx, db, runID, UTCNowISO(), ok, total, errMsg)
	}

	defer func() {
		if err != nil {
			msg := err.Error()
			if len(msg) > 8000 {
				msg = msg[len(msg)-8000:]
			}
			finishRun(false, &msg)
		}
	}()

	for {
		if opt.MaxEvents != nil && total >= *opt.MaxEvents {
			break
		}
		nowUTC := time.Now().UTC()
		var raw json.RawMessage
		raw, err = fetchEventsPageWithRetry(ctx, g, apiPageLimit, offset)
		if err != nil {
			return total, err
		}
		var batch []map[string]any
		if err = json.Unmarshal(raw, &batch); err != nil {
			return total, err
		}
		if len(batch) == 0 {
			break
		}
		fetchedAt := UTCNowISO()

		var tx *sql.Tx
		tx, err = db.BeginTx(ctx, nil)
		if err != nil {
			return total, err
		}
		for _, ev := range batch {
			if opt.MaxEvents != nil && total >= *opt.MaxEvents {
				break
			}
			if ev == nil {
				continue
			}
			if !EventEligibleForCatalog(ev, nowUTC) {
				continue
			}
			if err = StoreEventBundle(ctx, tx, ev, fetchedAt, opt.SourcesConfig); err != nil {
				_ = tx.Rollback()
				return total, err
			}
			total++
		}
		if err = tx.Commit(); err != nil {
			return total, err
		}

		if len(batch) < apiPageLimit {
			break
		}
		offset += apiPageLimit
		if opt.SleepBetweenPages > 0 {
			select {
			case <-time.After(opt.SleepBetweenPages):
			case <-ctx.Done():
				err = ctx.Err()
				return total, err
			}
		}
	}

	finishRun(true, nil)
	err = nil
	return total, nil
}

// DefaultSQLitePath returns ALPHA_RULES_SQLITE when set and non-empty.
func DefaultSQLitePath() string {
	return strings.TrimSpace(os.Getenv("ALPHA_RULES_SQLITE"))
}
