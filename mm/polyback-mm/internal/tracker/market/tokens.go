package market

import (
	"encoding/json"
	"strings"
)

// UpDownClobTokenIDs maps Gamma outcome order to Up (YES) and Down (NO) CLOB token ids.
func UpDownClobTokenIDs(m map[string]any, ids []string) (upTok, downTok string) {
	if len(ids) < 2 {
		return "", ""
	}
	outcomes := `["Yes","No"]`
	if o, ok := m["outcomes"].(string); ok && o != "" {
		outcomes = o
	}
	var outs []string
	_ = json.Unmarshal([]byte(outcomes), &outs)
	yesIdx, noIdx := -1, -1
	for i, o := range outs {
		ol := strings.ToLower(strings.TrimSpace(o))
		if ol == "yes" || ol == "up" {
			yesIdx = i
		}
		if ol == "no" || ol == "down" {
			noIdx = i
		}
	}
	if yesIdx >= 0 && yesIdx < len(ids) && noIdx >= 0 && noIdx < len(ids) {
		return ids[yesIdx], ids[noIdx]
	}
	return ids[0], ids[1]
}
