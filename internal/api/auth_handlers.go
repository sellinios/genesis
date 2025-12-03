// Package api - Authentication handlers
package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/aethra/genesis/internal/auth"
	"github.com/aethra/genesis/internal/errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LoginRateLimiter implements rate limiting for login attempts
type LoginRateLimiter struct {
	attempts map[string]*loginAttempt
	mu       sync.RWMutex
}

type loginAttempt struct {
	count     int
	firstTry  time.Time
	blockedAt *time.Time
}

// NewLoginRateLimiter creates a new rate limiter
func NewLoginRateLimiter() *LoginRateLimiter {
	rl := &LoginRateLimiter{
		attempts: make(map[string]*loginAttempt),
	}
	// Start cleanup goroutine
	go rl.cleanup()
	return rl
}

// Allow checks if a login attempt is allowed
func (rl *LoginRateLimiter) Allow(key string) (bool, int, time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	attempt, exists := rl.attempts[key]

	if !exists {
		rl.attempts[key] = &loginAttempt{count: 1, firstTry: now}
		return true, 4, 0 // 5 attempts allowed, 4 remaining
	}

	// If blocked, check if block has expired (15 minutes)
	if attempt.blockedAt != nil {
		blockDuration := 15 * time.Minute
		if now.Sub(*attempt.blockedAt) < blockDuration {
			remaining := blockDuration - now.Sub(*attempt.blockedAt)
			return false, 0, remaining
		}
		// Block expired, reset
		attempt.count = 1
		attempt.firstTry = now
		attempt.blockedAt = nil
		return true, 4, 0
	}

	// Reset if window (5 minutes) has passed
	if now.Sub(attempt.firstTry) > 5*time.Minute {
		attempt.count = 1
		attempt.firstTry = now
		return true, 4, 0
	}

	// Increment and check
	attempt.count++
	if attempt.count > 5 {
		attempt.blockedAt = &now
		return false, 0, 15 * time.Minute
	}

	return true, 5 - attempt.count, 0
}

// RecordFailure records a failed login attempt
func (rl *LoginRateLimiter) RecordFailure(key string) {
	// Already counted in Allow(), this is for tracking
}

// Reset resets the attempts for a key (on successful login)
func (rl *LoginRateLimiter) Reset(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.attempts, key)
}

// cleanup removes old entries periodically
func (rl *LoginRateLimiter) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, attempt := range rl.attempts {
			// Remove entries older than 30 minutes
			if now.Sub(attempt.firstTry) > 30*time.Minute {
				delete(rl.attempts, key)
			}
		}
		rl.mu.Unlock()
	}
}

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	db          *gorm.DB
	jwtService  *auth.JWTService
	rateLimiter *LoginRateLimiter
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(db *gorm.DB) *AuthHandler {
	return &AuthHandler{
		db:          db,
		jwtService:  auth.NewJWTService(),
		rateLimiter: NewLoginRateLimiter(),
	}
}

// LoginRequest represents login credentials
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	TenantID string `json:"tenant_id" binding:"required"`
}

// RegisterRequest represents registration data
type RegisterRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=8"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	TenantID  string `json:"tenant_id" binding:"required"`
}

// RefreshRequest represents a token refresh request
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// UserResponse represents user data in responses (without password)
type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	AvatarURL string    `json:"avatar_url,omitempty"`
	IsActive  bool      `json:"is_active"`
}

// Login authenticates a user and returns tokens
// POST /auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	// Rate limiting key: IP + email combination
	clientIP := c.ClientIP()
	rateLimitKey := clientIP + ":" + req.Email

	// Check rate limit
	allowed, remaining, retryAfter := h.rateLimiter.Allow(rateLimitKey)
	if !allowed {
		c.Header("Retry-After", retryAfter.String())
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":       "too many login attempts",
			"retry_after": retryAfter.Seconds(),
			"message":     "Please wait before trying again",
		})
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}

	// Find user
	var user struct {
		ID           uuid.UUID
		TenantID     uuid.UUID
		Email        string
		PasswordHash string
		FirstName    string
		LastName     string
		AvatarURL    *string
		IsActive     bool
	}

	err = h.db.Table("users").
		Where("email = ? AND tenant_id = ?", req.Email, tenantID).
		First(&user).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.Header("X-RateLimit-Remaining", string(rune(remaining)))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		} else {
			status, response := errors.ToHTTPError(errors.NewInternalError(err))
			c.JSON(status, response)
		}
		return
	}

	if !user.IsActive {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "account is disabled"})
		return
	}

	// Verify password
	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":               "invalid credentials",
			"attempts_remaining":  remaining,
		})
		return
	}

	// Successful login - reset rate limiter
	h.rateLimiter.Reset(rateLimitKey)

	// Get user roles
	var roles []string
	h.db.Table("user_roles").
		Joins("JOIN roles ON roles.id = user_roles.role_id").
		Where("user_roles.user_id = ?", user.ID).
		Pluck("roles.code", &roles)

	// Generate tokens
	tokens, err := h.jwtService.GenerateTokenPair(user.ID, user.TenantID, user.Email, roles)
	if err != nil {
		status, response := errors.ToHTTPError(errors.NewInternalError(err))
		c.JSON(status, response)
		return
	}

	// Update last login
	h.db.Table("users").Where("id = ?", user.ID).Update("last_login_at", gorm.Expr("CURRENT_TIMESTAMP"))

	// Build response
	avatarURL := ""
	if user.AvatarURL != nil {
		avatarURL = *user.AvatarURL
	}

	c.JSON(http.StatusOK, gin.H{
		"user": UserResponse{
			ID:        user.ID,
			TenantID:  user.TenantID,
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			AvatarURL: avatarURL,
			IsActive:  user.IsActive,
		},
		"tokens": tokens,
		"roles":  roles,
	})
}

// Register creates a new user account
// POST /auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}

	// Check if tenant exists
	var tenantExists bool
	h.db.Table("tenants").
		Select("1").
		Where("id = ? AND is_active = true", tenantID).
		Find(&tenantExists)

	if !tenantExists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
		return
	}

	// Check if email already exists
	var existingCount int64
	h.db.Table("users").
		Where("email = ? AND tenant_id = ?", req.Email, tenantID).
		Count(&existingCount)

	if existingCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}

	// Hash password
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		status, response := errors.ToHTTPError(errors.NewInternalError(err))
		c.JSON(status, response)
		return
	}

	// Create user
	userID := uuid.New()
	err = h.db.Exec(`
		INSERT INTO users (id, tenant_id, email, password_hash, first_name, last_name, is_active)
		VALUES (?, ?, ?, ?, ?, ?, true)
	`, userID, tenantID, req.Email, passwordHash, req.FirstName, req.LastName).Error

	if err != nil {
		status, response := errors.ToHTTPError(errors.NewInternalError(err))
		c.JSON(status, response)
		return
	}

	// Assign default role if exists
	var defaultRoleID uuid.UUID
	err = h.db.Table("roles").
		Select("id").
		Where("tenant_id = ? AND code = ?", tenantID, "user").
		First(&defaultRoleID).Error

	if err == nil {
		h.db.Exec("INSERT INTO user_roles (id, user_id, role_id) VALUES (?, ?, ?)",
			uuid.New(), userID, defaultRoleID)
	}

	// Generate tokens
	tokens, err := h.jwtService.GenerateTokenPair(userID, tenantID, req.Email, []string{"user"})
	if err != nil {
		status, response := errors.ToHTTPError(errors.NewInternalError(err))
		c.JSON(status, response)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user": UserResponse{
			ID:        userID,
			TenantID:  tenantID,
			Email:     req.Email,
			FirstName: req.FirstName,
			LastName:  req.LastName,
			IsActive:  true,
		},
		"tokens": tokens,
	})
}

// RefreshToken generates new tokens using a refresh token
// POST /auth/refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// Validate refresh token and get claims
	claims, err := h.jwtService.ValidateToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}

	// Get user info (to get latest email and roles)
	var user struct {
		Email    string
		IsActive bool
	}
	err = h.db.Table("users").
		Select("email, is_active").
		Where("id = ?", claims.UserID).
		First(&user).Error

	if err != nil || !user.IsActive {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found or disabled"})
		return
	}

	// Get current roles
	var roles []string
	h.db.Table("user_roles").
		Joins("JOIN roles ON roles.id = user_roles.role_id").
		Where("user_roles.user_id = ?", claims.UserID).
		Pluck("roles.code", &roles)

	// Generate new token pair
	tokens, err := h.jwtService.GenerateTokenPair(claims.UserID, claims.TenantID, user.Email, roles)
	if err != nil {
		status, response := errors.ToHTTPError(errors.NewInternalError(err))
		c.JSON(status, response)
		return
	}

	c.JSON(http.StatusOK, gin.H{"tokens": tokens})
}

// GetMe returns the current authenticated user
// GET /auth/me
func (h *AuthHandler) GetMe(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	var user struct {
		ID        uuid.UUID
		TenantID  uuid.UUID
		Email     string
		FirstName string
		LastName  string
		AvatarURL *string
		IsActive  bool
	}

	err := h.db.Table("users").
		Where("id = ?", userID).
		First(&user).Error

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Get roles
	var roles []string
	h.db.Table("user_roles").
		Joins("JOIN roles ON roles.id = user_roles.role_id").
		Where("user_roles.user_id = ?", userID).
		Pluck("roles.code", &roles)

	avatarURL := ""
	if user.AvatarURL != nil {
		avatarURL = *user.AvatarURL
	}

	c.JSON(http.StatusOK, gin.H{
		"user": UserResponse{
			ID:        user.ID,
			TenantID:  user.TenantID,
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			AvatarURL: avatarURL,
			IsActive:  user.IsActive,
		},
		"roles": roles,
	})
}

// ChangePassword changes the user's password
// POST /auth/change-password
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	// Get current password hash
	var passwordHash string
	err := h.db.Table("users").
		Select("password_hash").
		Where("id = ?", userID).
		First(&passwordHash).Error

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Verify current password
	if !auth.CheckPassword(req.CurrentPassword, passwordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "current password is incorrect"})
		return
	}

	// Hash new password
	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		status, response := errors.ToHTTPError(errors.NewInternalError(err))
		c.JSON(status, response)
		return
	}

	// Update password
	err = h.db.Table("users").
		Where("id = ?", userID).
		Update("password_hash", newHash).Error

	if err != nil {
		status, response := errors.ToHTTPError(errors.NewInternalError(err))
		c.JSON(status, response)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password changed successfully"})
}

// Logout invalidates the current session
// POST /auth/logout
// Note: For stateless JWT, we return success and let the client discard the token
// For production, consider implementing a token blacklist in Redis
func (h *AuthHandler) Logout(c *gin.Context) {
	// Get user info for audit logging
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	// Update last activity (optional audit)
	h.db.Table("users").Where("id = ?", userID).Update("updated_at", gorm.Expr("CURRENT_TIMESTAMP"))

	// In a production system, you would:
	// 1. Add the token to a blacklist (Redis with TTL matching token expiry)
	// 2. Or use short-lived access tokens with refresh token rotation

	c.JSON(http.StatusOK, gin.H{
		"message": "logged out successfully",
		"note":    "please discard your tokens client-side",
	})
}
