package cryptoclient

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func keccak256(b []byte) []byte {
	return crypto.Keccak256(b)
}

func hashString(s string) []byte {
	return keccak256([]byte(s))
}

func uint256Bytes(v *big.Int) []byte {
	if v.Sign() < 0 {
		panic("uint256 cannot be negative")
	}
	out := make([]byte, 32)
	b := v.Bytes()
	copy(out[32-len(b):], b)
	return out
}

func addressBytes(addr string) []byte {
	a := common.HexToAddress(strings.TrimSpace(addr))
	out := make([]byte, 32)
	copy(out[12:], a.Bytes())
	return out
}

func hashStruct(typeHash []byte, fields ...[]byte) []byte {
	total := 32 + 32*len(fields)
	buf := make([]byte, 0, total)
	buf = append(buf, typeHash...)
	for _, f := range fields {
		if len(f) != 32 {
			panic(fmt.Sprintf("expected 32-byte field, got %d", len(f)))
		}
		buf = append(buf, f...)
	}
	return keccak256(buf)
}

func eip712Digest(domainSeparator, messageHash []byte) []byte {
	buf := make([]byte, 2+32+32)
	buf[0] = 0x19
	buf[1] = 0x01
	copy(buf[2:], domainSeparator)
	copy(buf[34:], messageHash)
	return keccak256(buf)
}

// SignClobAuth matches Eip712Signer.signClobAuth (chainId, timestampSeconds, nonce).
func SignClobAuth(privateKeyHex string, chainID int, timestampSeconds int64, nonce int64) (string, error) {
	key, err := crypto.HexToECDSA(strings.TrimPrefix(strings.TrimSpace(privateKeyHex), "0x"))
	if err != nil {
		return "", err
	}
	addr := crypto.PubkeyToAddress(key.PublicKey)

	domainType := keccak256([]byte("EIP712Domain(string name,string version,uint256 chainId)"))
	domainSep := hashStruct(domainType,
		hashString("ClobAuthDomain"),
		hashString("1"),
		uint256Bytes(big.NewInt(int64(chainID))),
	)

	clobType := keccak256([]byte("ClobAuth(address address,string timestamp,uint256 nonce,string message)"))
	msgHash := hashStruct(clobType,
		addressBytes(addr.Hex()),
		hashString(formatInt(timestampSeconds)),
		uint256Bytes(big.NewInt(nonce)),
		hashString("This message attests that I control the given wallet"),
	)

	digest := eip712Digest(domainSep, msgHash)
	sig, err := crypto.Sign(digest, key)
	if err != nil {
		return "", err
	}
	if len(sig) != 65 {
		return "", fmt.Errorf("unexpected sig len %d", len(sig))
	}
	v := sig[64]
	if v < 27 {
		sig[64] = v + 27
	}
	return "0x" + hex.EncodeToString(sig), nil
}

// SignOrder matches Eip712Signer.signOrder for Polymarket CTF Exchange orders.
func SignOrder(privateKeyHex string, chainID int, verifyingContract, salt, maker, signer, taker, tokenID, makerAmount, takerAmount, expiration, nonce, feeRateBps string, side, signatureType int) (string, error) {
	key, err := crypto.HexToECDSA(strings.TrimPrefix(strings.TrimSpace(privateKeyHex), "0x"))
	if err != nil {
		return "", err
	}
	domainType := keccak256([]byte("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"))
	domainSep := hashStruct(domainType,
		hashString("Polymarket CTF Exchange"),
		hashString("1"),
		uint256Bytes(big.NewInt(int64(chainID))),
		addressBytes(verifyingContract),
	)

	orderTypeStr := "Order(uint256 salt,address maker,address signer,address taker,uint256 tokenId,uint256 makerAmount," +
		"uint256 takerAmount,uint256 expiration,uint256 nonce,uint256 feeRateBps,uint8 side,uint8 signatureType)"
	orderType := keccak256([]byte(orderTypeStr))

	saltBI := mustBigInt(salt)
	tokenBI := mustBigInt(tokenID)
	makerAmt := mustBigInt(makerAmount)
	takerAmt := mustBigInt(takerAmount)
	expBI := mustBigInt(expiration)
	nonceBI := mustBigInt(nonce)
	feeBI := mustBigInt(feeRateBps)

	msgHash := hashStruct(orderType,
		uint256Bytes(saltBI),
		addressBytes(maker),
		addressBytes(signer),
		addressBytes(taker),
		uint256Bytes(tokenBI),
		uint256Bytes(makerAmt),
		uint256Bytes(takerAmt),
		uint256Bytes(expBI),
		uint256Bytes(nonceBI),
		uint256Bytes(feeBI),
		uint256Bytes(big.NewInt(int64(side))),
		uint256Bytes(big.NewInt(int64(signatureType))),
	)

	digest := eip712Digest(domainSep, msgHash)
	sig, err := crypto.Sign(digest, key)
	if err != nil {
		return "", err
	}
	v := sig[64]
	if v < 27 {
		sig[64] = v + 27
	}
	return "0x" + hex.EncodeToString(sig), nil
}

func mustBigInt(s string) *big.Int {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		s = s[2:]
	}
	n := new(big.Int)
	if _, ok := n.SetString(s, 10); !ok {
		if _, ok := n.SetString(s, 16); !ok {
			panic("invalid int: " + s)
		}
	}
	return n
}
