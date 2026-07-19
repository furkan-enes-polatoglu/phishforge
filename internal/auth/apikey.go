package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
)

// GenerateAPIKey returns a new API key, its lookup prefix, and the sha256 hash to
// store. The full key is shown to the user exactly once. Format: "pfk_<40 hex>".
// The prefix ("pfk_" + first 12 hex) is stored in clear for O(1) lookup; the full
// key is verified against the stored hash.
func GenerateAPIKey() (fullKey, prefix, hash string) {
	b := make([]byte, 20)
	_, _ = rand.Read(b)
	h := hex.EncodeToString(b) // 40 hex chars
	fullKey = "pfk_" + h
	prefix = "pfk_" + h[:12]
	sum := sha256.Sum256([]byte(fullKey))
	hash = hex.EncodeToString(sum[:])
	return
}

// APIKeyPrefix extracts the lookup prefix from a full key.
func APIKeyPrefix(fullKey string) string {
	if len(fullKey) < 16 {
		return ""
	}
	return fullKey[:16] // "pfk_" + 12 hex
}

// VerifyAPIKey checks a full key against a stored sha256 hash (constant time).
func VerifyAPIKey(fullKey, storedHash string) bool {
	sum := sha256.Sum256([]byte(fullKey))
	got := hex.EncodeToString(sum[:])
	return subtle.ConstantTimeCompare([]byte(got), []byte(storedHash)) == 1
}
