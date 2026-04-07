package cryptoclient

import (
	"testing"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/domain"
)

func TestSignClobAuthOfficialVector(t *testing.T) {
	pk := "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	sig, err := SignClobAuth(pk, 80002, 10_000_000, 23)
	if err != nil {
		t.Fatal(err)
	}
	want := "0xf62319a987514da40e57e2f4d7529f7bac38f0355bd88bb5adbb3768d80de6c1682518e0af677d5260366425f4361e7b70c25ae232aff0ab2331e2b164a1aedc1b"
	if sig != want {
		t.Fatalf("got %q want %q", sig, want)
	}
}

func TestSignOrderOfficialVector(t *testing.T) {
	pk := "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	// derive address from key - Java uses same key's address
	sig, err := SignOrder(
		pk,
		80002,
		"0xdFE02Eb6733538f8Ea35D585af8DE5958AD99E40",
		"479249096354",
		"0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
		"0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
		"0x0000000000000000000000000000000000000000",
		"1234",
		"100000000",
		"50000000",
		"0",
		"0",
		"100",
		domain.SideBuy.EIP712Value(),
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	want := "0x302cd9abd0b5fcaa202a344437ec0b6660da984e24ae9ad915a592a90facf5a51bb8a873cd8d270f070217fea1986531d5eec66f1162a81f66e026db653bf7ce1c"
	if sig != want {
		t.Fatalf("got %q want %q", sig, want)
	}
}
