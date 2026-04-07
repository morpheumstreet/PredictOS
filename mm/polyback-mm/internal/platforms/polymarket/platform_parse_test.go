package polymarket

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestParseGammaMarket_outcomes(t *testing.T) {
	raw := []byte(`{
		"id": "m1",
		"slug": "s",
		"question": "Q?",
		"outcomes": [{"price": 60}, {"price": 40}],
		"clobTokenIds": ["yesTok", "noTok"],
		"volume24hr": 1234.5,
		"liquidity": 99,
		"active": true
	}`)
	m, err := parseGammaMarket(raw)
	if err != nil {
		t.Fatal(err)
	}
	if m.ID != "m1" || m.YesTokenID != "yesTok" || m.NoTokenID != "noTok" {
		t.Fatalf("ids: %+v", m)
	}
	if !m.YesPrice.Equal(decimal.RequireFromString("0.6")) || !m.NoPrice.Equal(decimal.RequireFromString("0.4")) {
		t.Fatalf("prices yes=%s no=%s", m.YesPrice, m.NoPrice)
	}
}

func TestParseGammaMarket_clobTokenIdsJSONString(t *testing.T) {
	// Gamma sometimes returns clobTokenIds as a stringified JSON array.
	raw := []byte(`{
		"id": "x",
		"question": "Q",
		"clobTokenIds": "[\"tokYes\", \"tokNo\"]",
		"active": true
	}`)
	m, err := parseGammaMarket(raw)
	if err != nil {
		t.Fatal(err)
	}
	if m.YesTokenID != "tokYes" || m.NoTokenID != "tokNo" {
		t.Fatalf("tokens yes=%q no=%q", m.YesTokenID, m.NoTokenID)
	}
}

func TestParseCLOBBook_strings(t *testing.T) {
	raw := []byte(`{"bids":[{"price":"0.45","size":"100"}],"asks":[{"price":"0.55","size":"200"}]}`)
	ob, err := parseCLOBBook(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(ob.Bids) != 1 || len(ob.Asks) != 1 {
		t.Fatalf("book %+v", ob)
	}
}
