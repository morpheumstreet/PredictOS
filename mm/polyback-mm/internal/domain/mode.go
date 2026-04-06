package domain

import "strings"

// TradingModeFromConfig maps YAML hft.mode to a typed mode (single place, avoids drift).
func TradingModeFromConfig(mode string) TradingMode {
	if strings.EqualFold(strings.TrimSpace(mode), "LIVE") {
		return ModeLive
	}
	return ModePaper
}
