package cryptoclient

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strings"
)

// SignPolyHmac matches com.polybot.hft.polymarket.crypto.PolyHmacSigner.sign
func SignPolyHmac(secretBase64 string, timestampSeconds int64, method, requestPath, body string) string {
	var msg strings.Builder
	msg.WriteString(formatInt(timestampSeconds))
	msg.WriteString(method)
	msg.WriteString(requestPath)
	if body != "" {
		msg.WriteString(body)
	}

	key := decodePolymarketSecret(secretBase64)
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(msg.String()))
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	sig = strings.ReplaceAll(sig, "+", "-")
	sig = strings.ReplaceAll(sig, "/", "_")
	return sig
}

func formatInt(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [32]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	s := string(buf[i:])
	if neg {
		return "-" + s
	}
	return s
}

func decodePolymarketSecret(secretBase64 string) []byte {
	s := strings.TrimSpace(secretBase64)
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '+' || c == '/' || c == '=' {
			b.WriteByte(c)
		}
	}
	sanitized := b.String()
	for len(sanitized)%4 != 0 {
		sanitized += "="
	}
	raw, err := base64.StdEncoding.DecodeString(sanitized)
	if err != nil {
		return nil
	}
	return raw
}
