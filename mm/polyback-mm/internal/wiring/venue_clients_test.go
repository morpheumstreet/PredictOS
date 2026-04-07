package wiring

import (
	"testing"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/config"
)

func TestVenueClientsFromHft_smoke(t *testing.T) {
	h := &config.Hft{
		PredictFun: config.PredictFunCfg{
			BaseURL:    "",
			APIKey:     "k",
			PrivateKey: "0x1",
		},
		KalshiDFlow: config.KalshiDFlowCfg{
			APIKey:      "d",
			EventTicker: "KX",
		},
		Limitless: config.LimitlessCfg{
			APIKey:        "l",
			WalletAddress: "0xabc",
		},
	}
	_ = PredictFunFromHft(h)
	_ = KalshiDFlowFromHft(h)
	_ = LimitlessFromHft(h)
}
