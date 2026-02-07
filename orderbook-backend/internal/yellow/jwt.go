package yellow

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// JWTClaims represents the Yellow Network JWT token claims
type JWTClaims struct {
	Address    string `json:"address"`
	SessionKey string `json:"session_key"`
	ExpiresAt  int64  `json:"expires_at"`
	Scope      string `json:"scope"`
}

// UserSession represents an authenticated user session
type UserSession struct {
	Address    string
	SessionKey string
	JWTToken   string
	ExpiresAt  time.Time
}

// ParseJWT parses a Yellow Network JWT token (simplified version)
// Note: In production, you should verify the signature against Yellow's public key
func ParseJWT(tokenString string) (*JWTClaims, error) {
	// JWT format: header.payload.signature
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}

	// Decode payload (base64url)
	payload := parts[1]

	// Add padding if needed
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}

	// For now, we'll just return a basic claims structure
	// In production, decode the base64 and verify signature
	claims := &JWTClaims{
		// These would be extracted from the actual JWT
		// For now, we accept the token as-is if it exists
	}

	return claims, nil
}

// ValidateToken validates a Yellow JWT token
func ValidateToken(tokenString string) (*UserSession, error) {
	if tokenString == "" {
		return nil, fmt.Errorf("empty token")
	}

	// Parse the token
	claims, err := ParseJWT(tokenString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// Check expiration (if available)
	if claims.ExpiresAt > 0 {
		expiresAt := time.Unix(claims.ExpiresAt, 0)
		if time.Now().After(expiresAt) {
			return nil, fmt.Errorf("token expired")
		}

		return &UserSession{
			Address:    claims.Address,
			SessionKey: claims.SessionKey,
			JWTToken:   tokenString,
			ExpiresAt:  expiresAt,
		}, nil
	}

	// If no expiration info, create session without expiry check
	return &UserSession{
		Address:    claims.Address,
		SessionKey: claims.SessionKey,
		JWTToken:   tokenString,
		ExpiresAt:  time.Now().Add(1 * time.Hour),
	}, nil
}

// YellowAuthMessage represents the WebSocket auth message from frontend
type YellowAuthMessage struct {
	Type       string `json:"type"`
	JWTToken   string `json:"jwt_token"`
	SessionKey string `json:"session_key"`
}

// ParseYellowAuth parses a Yellow auth message from WebSocket
func ParseYellowAuth(data []byte) (*YellowAuthMessage, error) {
	var msg YellowAuthMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}

	if msg.Type != "yellow_auth" {
		return nil, fmt.Errorf("invalid message type: %s", msg.Type)
	}

	return &msg, nil
}
