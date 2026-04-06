package cryptoclient

import "testing"

func TestPolyHmacSignerOfficialVector(t *testing.T) {
	sig := SignPolyHmac(
		"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		1_000_000,
		"test-sign",
		"/orders",
		`{"hash": "0x123"}`,
	)
	want := "ZwAdJKvoYRlEKDkNMwd5BuwNNtg93kNaR_oU2HrfVvc="
	if sig != want {
		t.Fatalf("got %q want %q", sig, want)
	}
}
