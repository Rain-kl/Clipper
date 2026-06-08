package cap

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"
)

const jwtHeaderB64 = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"

// ChallengeConfig holds parameters for the PoW challenge
type ChallengeConfig struct {
	Count      int           // Number of puzzles (c)
	Size       int           // Salt length (s)
	Difficulty int           // Difficulty prefix length (d)
	ExpiresMs  time.Duration // Challenge TTL
}

// ChallengeResponse is returned to the client
type ChallengeResponse struct {
	Challenge struct {
		C int `json:"c"`
		S int `json:"s"`
		D int `json:"d"`
	} `json:"challenge"`
	Token   string `json:"token"`
	Expires int64  `json:"expires"` // ms timestamp
}

// ChallengePayload represents the signed JWT payload
type ChallengePayload struct {
	Nonce      string `json:"n"`
	Count      int    `json:"c"`
	Size       int    `json:"s"`
	Difficulty int    `json:"d"`
	Expires    int64  `json:"exp"` // ms timestamp
	IssuedAt   int64  `json:"iat"` // ms timestamp
	Scope      string `json:"sk,omitempty"`
}

// RedeemRequest payload sent by client
type RedeemRequest struct {
	Token     string `json:"token"`
	Solutions []int  `json:"solutions"`
}

// RedeemResponse returned to client after verification
type RedeemResponse struct {
	Success bool   `json:"success"`
	Token   string `json:"token,omitempty"`
	Expires int64  `json:"expires,omitempty"`
	Error   string `json:"error,omitempty"`
}

func b64urlEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func b64urlDecode(str string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(str)
}

func randomHex(byteLen int) string {
	bytes := make([]byte, byteLen)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}

func jwtSign(payload []byte, secret []byte) string {
	body := b64urlEncode(payload)
	sigInput := jwtHeaderB64 + "." + body

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(sigInput))
	sig := mac.Sum(nil)

	return sigInput + "." + b64urlEncode(sig)
}

func jwtVerify(token string, secret []byte) ([]byte, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token format")
	}
	if parts[0] != jwtHeaderB64 {
		return nil, errors.New("invalid header")
	}

	sigInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(sigInput))
	expectedSig := mac.Sum(nil)

	actualSig, err := b64urlDecode(parts[2])
	if err != nil {
		return nil, err
	}

	if !hmac.Equal(expectedSig, actualSig) {
		return nil, errors.New("signature mismatch")
	}

	payload, err := b64urlDecode(parts[1])
	if err != nil {
		return nil, err
	}

	return payload, nil
}

func jwtSigHex(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ""
	}
	sigBytes, err := b64urlDecode(parts[2])
	if err != nil {
		return ""
	}
	return hex.EncodeToString(sigBytes)
}

// GenerateChallenge produces a new challenge and signed token
func GenerateChallenge(secret []byte, conf ChallengeConfig, scope string) (*ChallengeResponse, error) {
	if conf.Count <= 0 {
		conf.Count = 50
	}
	if conf.Size <= 0 {
		conf.Size = 32
	}
	if conf.Difficulty <= 0 {
		conf.Difficulty = 4
	}
	if conf.ExpiresMs <= 0 {
		conf.ExpiresMs = 10 * time.Minute
	}

	now := time.Now().UnixNano() / int64(time.Millisecond)
	expires := now + int64(conf.ExpiresMs/time.Millisecond)

	payload := ChallengePayload{
		Nonce:      randomHex(25),
		Count:      conf.Count,
		Size:       conf.Size,
		Difficulty: conf.Difficulty,
		Expires:    expires,
		IssuedAt:   now,
		Scope:      scope,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	token := jwtSign(payloadBytes, secret)

	resp := &ChallengeResponse{
		Token:   token,
		Expires: expires,
	}
	resp.Challenge.C = conf.Count
	resp.Challenge.S = conf.Size
	resp.Challenge.D = conf.Difficulty

	return resp, nil
}

// VerifyChallengeSolutions verifies client submitted solutions
func VerifyChallengeSolutions(token string, solutions []int, secret []byte, expectedScope string) (*ChallengePayload, error) {
	payloadBytes, err := jwtVerify(token, secret)
	if err != nil {
		return nil, errors.New("invalid_token")
	}

	var payload ChallengePayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, errors.New("invalid_token")
	}

	if expectedScope != "" && payload.Scope != expectedScope {
		return nil, errors.New("scope_mismatch")
	}

	now := time.Now().UnixNano() / int64(time.Millisecond)
	if payload.Expires < now {
		return nil, errors.New("expired")
	}

	if len(solutions) != payload.Count {
		return nil, errors.New("invalid_solutions")
	}

	tokenFnv := fnv1a(token)
	for i := 0; i < payload.Count; i++ {
		idxStr := strconv.Itoa(i + 1)
		saltSeed := fnv1aResume(tokenFnv, idxStr)
		targetSeed := fnv1aResume(saltSeed, "d")
		salt := prngFromHash(saltSeed, payload.Size)
		target := prngFromHash(targetSeed, payload.Difficulty)

		hashInput := salt + strconv.Itoa(solutions[i])
		hashBytes := sha256.Sum256([]byte(hashInput))
		hashHex := hex.EncodeToString(hashBytes[:])

		if !strings.HasPrefix(hashHex, target) {
			return nil, errors.New("invalid_solution")
		}
	}

	return &payload, nil
}

// Solve is a utility function to solve a challenge (mainly used for tests and reference implementation)
func Solve(token string, count, size, difficulty int) []int {
	solutions := make([]int, count)
	tokenFnv := fnv1a(token)
	for i := 0; i < count; i++ {
		idxStr := strconv.Itoa(i + 1)
		saltSeed := fnv1aResume(tokenFnv, idxStr)
		targetSeed := fnv1aResume(saltSeed, "d")
		salt := prngFromHash(saltSeed, size)
		target := prngFromHash(targetSeed, difficulty)

		for nonce := 0; nonce < 1000000; nonce++ {
			hashInput := salt + strconv.Itoa(nonce)
			hashBytes := sha256.Sum256([]byte(hashInput))
			hashHex := hex.EncodeToString(hashBytes[:])
			if strings.HasPrefix(hashHex, target) {
				solutions[i] = nonce
				break
			}
		}
	}
	return solutions
}
