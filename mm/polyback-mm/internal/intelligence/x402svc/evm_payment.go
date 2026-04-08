package x402svc

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

func randomNonceHex() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "0x" + common.Bytes2Hex(b[:]), nil
}

func chainIDForNetwork(network string) int64 {
	n := strings.ToLower(strings.TrimSpace(network))
	switch n {
	case "base", "eip155:8453":
		return 8453
	default:
		return 8453
	}
}

// BuildEVMPaymentHeaderBase64 builds the base64 PAYMENT-SIGNATURE payload (x402 v2) for EVM USDC EIP-3009.
func BuildEVMPaymentHeaderBase64(
	privateKeyHex string,
	payTo string,
	asset string,
	amountAtomic string,
	network string,
	x402Version int,
	scheme string,
	maxTimeoutSeconds int,
	extra map[string]any,
	resourceURL, resourceDesc, mimeType string,
) (string, error) {
	keyHex := strings.TrimPrefix(strings.TrimSpace(privateKeyHex), "0x")
	if keyHex == "" {
		return "", fmt.Errorf("empty evm private key")
	}
	priv, err := crypto.HexToECDSA(keyHex)
	if err != nil {
		return "", fmt.Errorf("invalid evm private key: %w", err)
	}
	fromAddr := crypto.PubkeyToAddress(priv.PublicKey)
	payToAddr := common.HexToAddress(payTo)
	assetAddr := common.HexToAddress(asset)
	if _, ok := new(big.Int).SetString(strings.TrimSpace(amountAtomic), 10); !ok {
		return "", fmt.Errorf("invalid amount")
	}

	now := time.Now().Unix()
	validAfter := big.NewInt(now - 600)
	validBefore := big.NewInt(now + int64(maxTimeoutSeconds))
	if maxTimeoutSeconds <= 0 {
		validBefore = big.NewInt(now + 60)
	}

	nonce, err := randomNonceHex()
	if err != nil {
		return "", err
	}

	name := "USD Coin"
	version := "2"
	if extra != nil {
		if v, ok := extra["name"].(string); ok && v != "" {
			name = v
		}
		if v, ok := extra["version"].(string); ok && v != "" {
			version = v
		}
	}

	chainID := chainIDForNetwork(network)
	domain := apitypes.TypedDataDomain{
		Name:              name,
		Version:           version,
		ChainId:           (*math.HexOrDecimal256)(big.NewInt(chainID)),
		VerifyingContract: assetAddr.Hex(),
	}

	types := apitypes.Types{
		"EIP712Domain": {
			{Name: "name", Type: "string"},
			{Name: "version", Type: "string"},
			{Name: "chainId", Type: "uint256"},
			{Name: "verifyingContract", Type: "address"},
		},
		"TransferWithAuthorization": {
			{Name: "from", Type: "address"},
			{Name: "to", Type: "address"},
			{Name: "value", Type: "uint256"},
			{Name: "validAfter", Type: "uint256"},
			{Name: "validBefore", Type: "uint256"},
			{Name: "nonce", Type: "bytes32"},
		},
	}

	msg := apitypes.TypedDataMessage{
		"from":        fromAddr.Hex(),
		"to":          payToAddr.Hex(),
		"value":       mustBigInt(amountAtomic),
		"validAfter":  validAfter,
		"validBefore": validBefore,
		"nonce":       common.HexToHash(nonce),
	}

	td := apitypes.TypedData{
		Types:       types,
		PrimaryType: "TransferWithAuthorization",
		Domain:      domain,
		Message:     msg,
	}

	digest, _, err := apitypes.TypedDataAndHash(td)
	if err != nil {
		return "", err
	}
	sig, err := crypto.Sign(digest, priv)
	if err != nil {
		return "", err
	}
	if sig[64] < 27 {
		sig[64] += 27
	}
	signature := "0x" + common.Bytes2Hex(sig)

	authorization := map[string]any{
		"from":        fromAddr.Hex(),
		"to":          payToAddr.Hex(),
		"value":       amountAtomic,
		"validAfter":  fmt.Sprintf("%d", validAfter.Int64()),
		"validBefore": fmt.Sprintf("%d", validBefore.Int64()),
		"nonce":       nonce,
	}

	if scheme == "" {
		scheme = "exact"
	}

	payload := map[string]any{
		"x402Version": x402Version,
		"resource": map[string]any{
			"url":         resourceURL,
			"description": resourceDesc,
			"mimeType":    mimeType,
		},
		"accepted": map[string]any{
			"scheme":            scheme,
			"network":           network,
			"asset":             asset,
			"amount":            amountAtomic,
			"payTo":             payTo,
			"maxTimeoutSeconds": maxTimeoutSeconds,
			"extra":             extra,
		},
		"payload": map[string]any{
			"authorization": authorization,
			"signature":     signature,
		},
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(raw), nil
}

func mustBigInt(s string) *big.Int {
	x := new(big.Int)
	if _, ok := x.SetString(strings.TrimSpace(s), 10); !ok {
		return big.NewInt(0)
	}
	return x
}
