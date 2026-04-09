package config

import (
	"os"
	"testing"
)

func TestLoad_missingEnv(t *testing.T) {
	_ = os.Unsetenv("IRYS_CHAIN_ENVIRONMENT")
	_ = os.Unsetenv("IRYS_SOLANA_PRIVATE_KEY")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoad_devnetRequiresRPC(t *testing.T) {
	t.Setenv("IRYS_CHAIN_ENVIRONMENT", "devnet")
	t.Setenv("IRYS_SOLANA_PRIVATE_KEY", "dummy")
	_ = os.Unsetenv("IRYS_SOLANA_RPC_URL")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing RPC on devnet")
	}
}
