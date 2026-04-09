package irysupload

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/donutnomad/solana-web3/irys"
	"github.com/donutnomad/solana-web3/web3"
	"github.com/profitlock/PredictOS/mm/irys-upload/internal/config"
)

const gatewayBase = "https://gateway.irys.xyz/"

// UploadPayload mirrors the terminal JSON body (subset used for tags + storage).
type UploadPayload struct {
	RequestID       string `json:"requestId"`
	Timestamp       string `json:"timestamp"`
	PmType          string `json:"pmType"`
	EventIdentifier string `json:"eventIdentifier"`
	EventID         string `json:"eventId,omitempty"`
	AnalysisMode    string `json:"analysisMode"`
	SchemaVersion   string `json:"schemaVersion"`
	AgentsData      []struct {
		Name string `json:"name"`
	} `json:"agentsData"`
}

type UploadResult struct {
	Success       bool   `json:"success"`
	TransactionID string `json:"transactionId,omitempty"`
	GatewayURL    string `json:"gatewayUrl,omitempty"`
	Environment   string `json:"environment,omitempty"`
	Error         string `json:"error,omitempty"`
}

type StatusResponse struct {
	Configured  bool   `json:"configured"`
	Environment string `json:"environment"`
	Error       string `json:"error,omitempty"`
}

// Client performs Irys uploads using Solana (community library: donutnomad/solana-web3/irys).
type Client struct {
	cfg *config.Config
}

func NewClient(cfg *config.Config) *Client {
	return &Client{cfg: cfg}
}

func (c *Client) Status() StatusResponse {
	return StatusResponse{
		Configured:  true,
		Environment: c.cfg.Environment,
	}
}

func (c *Client) Upload(ctx context.Context, rawBody []byte) (*UploadResult, error) {
	var p UploadPayload
	if err := json.Unmarshal(rawBody, &p); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	if p.RequestID == "" {
		return &UploadResult{Success: false, Error: "Missing required field: requestId"}, nil
	}
	if len(p.AgentsData) == 0 {
		return &UploadResult{Success: false, Error: "Missing required field: agentsData"}, nil
	}

	var prettyJSON bytes.Buffer
	var v any
	if err := json.Unmarshal(rawBody, &v); err != nil {
		return nil, err
	}
	enc := json.NewEncoder(&prettyJSON)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	dataToUpload := bytes.TrimSpace(prettyJSON.Bytes())

	signer, err := web3.Keypair.TryFromBase58(c.cfg.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("IRYS_SOLANA_PRIVATE_KEY: %w", err)
	}

	rpcURL := c.cfg.RPCURL
	if rpcURL == "" {
		rpcURL = c.cfg.MainnetRPCDefault()
	}
	timeoutMs := 120_000
	conn, err := web3.NewConnection(rpcURL, &web3.ConnectionConfig{
		Commitment:                       &web3.CommitmentFinalized,
		ConfirmTransactionInitialTimeout: &timeoutMs,
	})
	if err != nil {
		return nil, fmt.Errorf("solana rpc: %w", err)
	}

	endpoint := irys.NODE1
	if c.cfg.Environment == "devnet" {
		endpoint = irys.DEV
	}
	node := irys.NewIrys(endpoint)
	node.HttpClient = &http.Client{Timeout: 120 * time.Second}

	agentNames := ""
	for i, a := range p.AgentsData {
		if i > 0 {
			agentNames += ", "
		}
		agentNames += a.Name
	}
	if len(agentNames) > 100 {
		agentNames = agentNames[:100]
	}

	tags := map[string]string{
		"Content-Type":     "application/json",
		"App-Name":         "PredictOS",
		"App-Version":      "1.0.0",
		"Request-Id":       p.RequestID,
		"PM-Type":          p.PmType,
		"Event-Identifier": p.EventIdentifier,
		"Analysis-Mode":    p.AnalysisMode,
		"Agents-Count":     fmt.Sprintf("%d", len(p.AgentsData)),
		"Agents":           agentNames,
		"Schema-Version":   p.SchemaVersion,
		"Environment":      c.cfg.Environment,
	}

	if err := node.FundByBytes(ctx, conn, signer, len(dataToUpload)); err != nil {
		return nil, fmt.Errorf("irys fund: %w", err)
	}
	receipt, err := node.Upload(dataToUpload, signer, tags)
	if err != nil {
		return nil, fmt.Errorf("irys upload: %w", err)
	}

	return &UploadResult{
		Success:       true,
		TransactionID: receipt.Id,
		GatewayURL:    gatewayBase + receipt.Id,
		Environment:   c.cfg.Environment,
	}, nil
}
