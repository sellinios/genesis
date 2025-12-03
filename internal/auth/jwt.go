// Package auth provides authentication utilities for Genesis
package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Claims represents JWT claims for Genesis
type Claims struct {
	UserID   uuid.UUID `json:"user_id"`
	TenantID uuid.UUID `json:"tenant_id"`
	Email    string    `json:"email"`
	Roles    []string  `json:"roles,omitempty"`
	jwt.RegisteredClaims
}

// TokenPair represents access and refresh tokens
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// JWTService handles JWT operations
type JWTService struct {
	secretKey          []byte
	accessTokenExpiry  time.Duration
	refreshTokenExpiry time.Duration
	issuer             string
}

// NewJWTService creates a new JWT service
func NewJWTService() *JWTService {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// Generate a random secret for development
		secret = generateRandomSecret()
		fmt.Println("⚠️  Warning: JWT_SECRET not set, using random secret (not suitable for production)")
	}

	accessExpiry := 24 * time.Hour      // 24 hours default
	refreshExpiry := 7 * 24 * time.Hour // 7 days default

	if exp := os.Getenv("JWT_ACCESS_EXPIRY_HOURS"); exp != "" {
		if hours, err := time.ParseDuration(exp + "h"); err == nil {
			accessExpiry = hours
		}
	}

	if exp := os.Getenv("JWT_REFRESH_EXPIRY_DAYS"); exp != "" {
		if days, err := time.ParseDuration(exp + "h"); err == nil {
			refreshExpiry = days * 24
		}
	}

	return &JWTService{
		secretKey:          []byte(secret),
		accessTokenExpiry:  accessExpiry,
		refreshTokenExpiry: refreshExpiry,
		issuer:             "genesis",
	}
}

// GenerateTokenPair generates access and refresh tokens
func (s *JWTService) GenerateTokenPair(userID, tenantID uuid.UUID, email string, roles []string) (*TokenPair, error) {
	now := time.Now()
	accessExpiresAt := now.Add(s.accessTokenExpiry)
	refreshExpiresAt := now.Add(s.refreshTokenExpiry)

	// Create access token
	accessClaims := &Claims{
		UserID:   userID,
		TenantID: tenantID,
		Email:    email,
		Roles:    roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExpiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    s.issuer,
			Subject:   userID.String(),
			ID:        uuid.New().String(),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(s.secretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Create refresh token (minimal claims)
	refreshClaims := &Claims{
		UserID:   userID,
		TenantID: tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExpiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    s.issuer,
			Subject:   userID.String(),
			ID:        uuid.New().String(),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(s.secretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresAt:    accessExpiresAt,
		TokenType:    "Bearer",
	}, nil
}

// ValidateToken validates a JWT token and returns the claims
func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// RefreshAccessToken generates a new access token using a refresh token
func (s *JWTService) RefreshAccessToken(refreshTokenString string, email string, roles []string) (*TokenPair, error) {
	claims, err := s.ValidateToken(refreshTokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Generate new token pair
	return s.GenerateTokenPair(claims.UserID, claims.TenantID, email, roles)
}

// generateRandomSecret generates a random 32-byte secret
func generateRandomSecret() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "genesis-default-secret-change-me"
	}
	return base64.StdEncoding.EncodeToString(bytes)
}

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// CheckPassword verifies a password against a bcrypt hash
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
