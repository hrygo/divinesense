// Package dingtalk provides cryptographic utilities for DingTalk.
package dingtalk

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"sort"
	"strings"
)

// SignURL generates a signature for DingTalk API requests.
func SignURL(appSecret, url string, params map[string]string) string {
	// Sort parameters by key
	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build parameter string
	var paramParts []string
	for _, k := range keys {
		paramParts = append(paramParts, fmt.Sprintf("%s=%s", k, params[k]))
	}
	paramString := strings.Join(paramParts, "&")

	// Compute signature
	stringToSign := url + "?" + paramString
	h := hmac.New(sha256.New, []byte(appSecret))
	h.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return signature
}

// VerifyWebhookSignature verifies the DingTalk webhook signature.
func VerifyWebhookSignature(timestamp, sign, secret, body string) bool {
	expected := computeWebhookSignature(timestamp, secret, body)
	return hmac.Equal([]byte(sign), []byte(expected))
}

func computeWebhookSignature(timestamp, secret, body string) string {
	stringToSign := timestamp + "\n" + body
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// DecryptCallback decrypts a callback from DingTalk (for enterprise robots).
func DecryptCallback(encryptKey, ciphertext, iv string) ([]byte, error) {
	// Base64 decode
	key, err := base64.StdEncoding.DecodeString(encryptKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode key: %w", err)
	}

	decoded, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	decodedIV, err := base64.StdEncoding.DecodeString(iv)
	if err != nil {
		return nil, fmt.Errorf("failed to decode IV: %w", err)
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Decrypt using CBC mode
	mode := cipher.NewCBCDecrypter(block, decodedIV)
	mode.CryptBlocks(decoded, decoded)

	// Remove PKCS7 padding
	padding := int(decoded[len(decoded)-1])
	if padding < 1 || padding > block.BlockSize() {
		return nil, fmt.Errorf("invalid padding")
	}

	return decoded[:len(decoded)-padding], nil
}
