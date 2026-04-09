package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds env for the Irys upload service (Solana-funded uploads).
type Config struct {
	Environment string // mainnet | devnet
	PrivateKey  string // base58 Solana keypair
	RPCURL      string // required for devnet
	Port        int
}

func Load() (*Config, error) {
	env := os.Getenv("IRYS_CHAIN_ENVIRONMENT")
	if env == "" {
		return nil, fmt.Errorf("IRYS_CHAIN_ENVIRONMENT is not set")
	}
	if env != "mainnet" && env != "devnet" {
		return nil, fmt.Errorf("IRYS_CHAIN_ENVIRONMENT must be mainnet or devnet")
	}
	key := os.Getenv("IRYS_SOLANA_PRIVATE_KEY")
	if key == "" {
		return nil, fmt.Errorf("IRYS_SOLANA_PRIVATE_KEY is not set")
	}
	rpc := os.Getenv("IRYS_SOLANA_RPC_URL")
	if env == "devnet" && rpc == "" {
		return nil, fmt.Errorf("IRYS_SOLANA_RPC_URL is required for devnet")
	}
	port := 8091
	if p := os.Getenv("PORT"); p != "" {
		var err error
		port, err = strconv.Atoi(p)
		if err != nil || port <= 0 {
			return nil, fmt.Errorf("invalid PORT: %q", p)
		}
	}
	return &Config{
		Environment: env,
		PrivateKey:  key,
		RPCURL:      rpc,
		Port:        port,
	}, nil
}

func (c *Config) MainnetRPCDefault() string {
	if c.RPCURL != "" {
		return c.RPCURL
	}
	return "https://api.mainnet-beta.solana.com/"
}
