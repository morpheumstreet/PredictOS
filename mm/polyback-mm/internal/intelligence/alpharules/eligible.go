package alpharules

import (
	"time"
)

// EventEligibleForCatalog mirrors strat/alpha-rules/collect.py event_eligible_for_catalog.
func EventEligibleForCatalog(ev map[string]any, nowUTC time.Time) bool {
	active, _ := ev["active"].(bool)
	if !active {
		return false
	}
	closed, _ := ev["closed"].(bool)
	if closed {
		return false
	}
	archived, _ := ev["archived"].(bool)
	if archived {
		return false
	}
	end := ParseAPIDateUTC(ev["endDate"])
	if end != nil && !end.After(nowUTC) {
		return false
	}
	return true
}
