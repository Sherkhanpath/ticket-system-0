package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ---------- small helpers ----------

func nowUTC() time.Time {
	return time.Now().UTC()
}

func idFromCounter(prefix string, n int) string {
	return fmt.Sprintf("%s_%d", prefix, n)
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// writeError writes a JSON error response: {"error": "message"}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// ---------- password hashing ----------
//
// We avoid third-party packages (like golang.org/x/crypto/bcrypt) on
// purpose so the project builds with zero external dependencies.
// Passwords are stored as salt + SHA-256(salt + password), which
// satisfies the "must be hashed, not plain text" requirement.

// hashPassword returns a string of the form "hex(salt):hex(hash)".
func hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	sum := sha256.Sum256(append(salt, []byte(password)...))
	return hex.EncodeToString(salt) + ":" + hex.EncodeToString(sum[:]), nil
}

// verifyPassword checks a plaintext password against a stored hash.
func verifyPassword(password, stored string) bool {
	parts := strings.SplitN(stored, ":", 2)
	if len(parts) != 2 {
		return false
	}
	salt, err := hex.DecodeString(parts[0])
	if err != nil {
		return false
	}
	wantHash, err := hex.DecodeString(parts[1])
	if err != nil {
		return false
	}
	gotSum := sha256.Sum256(append(salt, []byte(password)...))
	return subtle.ConstantTimeCompare(gotSum[:], wantHash) == 1
}

// ---------- minimal JWT (HS256) ----------
//
// A tiny hand-rolled JWT implementation using only the standard
// library, so no external JWT package is required.

type jwtClaims struct {
	Sub string `json:"sub"` // user ID
	Exp int64  `json:"exp"` // unix expiry
}

func base64URLEncode(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func base64URLDecode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

// createToken issues a signed JWT for the given user ID, valid for the
// given duration.
func createToken(userID string, secret []byte, ttl time.Duration) (string, error) {
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}

	claims := jwtClaims{Sub: userID, Exp: nowUTC().Add(ttl).Unix()}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	unsigned := base64URLEncode(headerJSON) + "." + base64URLEncode(claimsJSON)

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(unsigned))
	signature := mac.Sum(nil)

	return unsigned + "." + base64URLEncode(signature), nil
}

// parseAndVerifyToken validates the signature and expiry of a JWT and
// returns the user ID stored in it.
func parseAndVerifyToken(token string, secret []byte) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", errors.New("malformed token")
	}
	unsigned := parts[0] + "." + parts[1]

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(unsigned))
	expectedSig := mac.Sum(nil)

	gotSig, err := base64URLDecode(parts[2])
	if err != nil {
		return "", errors.New("malformed token signature")
	}
	if !hmac.Equal(expectedSig, gotSig) {
		return "", errors.New("invalid token signature")
	}

	claimsJSON, err := base64URLDecode(parts[1])
	if err != nil {
		return "", errors.New("malformed token claims")
	}
	var claims jwtClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return "", errors.New("malformed token claims")
	}

	if nowUTC().Unix() > claims.Exp {
		return "", errors.New("token expired")
	}

	return claims.Sub, nil
}
