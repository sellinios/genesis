// Package api contains the HTTP API handlers for Genesis
package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/aethra/genesis/internal/auth"
	"github.com/aethra/genesis/internal/engine"
	"github.com/aethra/genesis/internal/errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler contains all API handlers
type Handler struct {
	schemaEngine      *engine.SchemaEngine
	dataEngine        *engine.DataEngine
	jwtService        *auth.JWTService
	permissionService *auth.PermissionService
}

// NewHandler creates a new API handler
func NewHandler(schemaEngine *engine.SchemaEngine, dataEngine *engine.DataEngine) *Handler {
	return &Handler{
		schemaEngine: schemaEngine,
		dataEngine:   dataEngine,
		jwtService:   auth.NewJWTService(),
	}
}

// NewHandlerWithPermissions creates a new API handler with permission checking
func NewHandlerWithPermissions(schemaEngine *engine.SchemaEngine, dataEngine *engine.DataEngine, permService *auth.PermissionService) *Handler {
	return &Handler{
		schemaEngine:      schemaEngine,
		dataEngine:        dataEngine,
		jwtService:        auth.NewJWTService(),
		permissionService: permService,
	}
}

// =============================================================================
// MIDDLEWARE
// =============================================================================

// TenantMiddleware extracts tenant from header or subdomain
func (h *Handler) TenantMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try header first
		tenantIDStr := c.GetHeader("X-Tenant-ID")
		if tenantIDStr == "" {
			// Try query param (for testing)
			tenantIDStr = c.Query("tenant_id")
		}

		if tenantIDStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
			c.Abort()
			return
		}

		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
			c.Abort()
			return
		}

		c.Set("tenant_id", tenantID)
		c.Next()
	}
}

// UserMiddleware extracts user from JWT token
func (h *Handler) UserMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// No auth header - continue without user context (for optional auth)
			c.Next()
			return
		}

		// Check Bearer prefix
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Validate token
		claims, err := h.jwtService.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			c.Abort()
			return
		}

		// Set user context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_roles", claims.Roles)

		// Validate tenant matches (if tenant_id is set)
		if tenantID, exists := c.Get("tenant_id"); exists {
			if tid, ok := tenantID.(uuid.UUID); ok && tid != claims.TenantID {
				c.JSON(http.StatusForbidden, gin.H{"error": "user does not belong to this tenant"})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// RequireAuthMiddleware requires authentication (must be used after UserMiddleware)
func (h *Handler) RequireAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, exists := c.Get("user_id"); !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// PermissionMiddleware checks if user has permission for the requested action
func (h *Handler) PermissionMiddleware(action auth.Action) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip if no permission service configured
		if h.permissionService == nil {
			c.Next()
			return
		}

		// Get required context
		userID, userExists := c.Get("user_id")
		tenantID, tenantExists := c.Get("tenant_id")
		entityCode := c.Param("entity")

		// If no user or tenant, skip permission check (other middleware will handle)
		if !userExists || !tenantExists || entityCode == "" {
			c.Next()
			return
		}

		// Check permission
		hasPermission, err := h.permissionService.CheckPermission(
			tenantID.(uuid.UUID),
			userID.(uuid.UUID),
			entityCode,
			action,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check permissions"})
			c.Abort()
			return
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "permission denied",
				"action":  string(action),
				"entity":  entityCode,
				"message": "You do not have permission to perform this action",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// =============================================================================
// SCHEMA ENDPOINTS
// =============================================================================

// GetSchema returns the full schema for a tenant
// GET /api/schema
func (h *Handler) GetSchema(c *gin.Context) {
	tenantID := c.MustGet("tenant_id").(uuid.UUID)

	schema, err := h.schemaEngine.GetFullSchema(tenantID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, schema)
}

// GetEntitySchema returns the schema for a specific entity
// GET /api/schema/:entity
func (h *Handler) GetEntitySchema(c *gin.Context) {
	tenantID := c.MustGet("tenant_id").(uuid.UUID)
	entityCode := c.Param("entity")

	schema, err := h.schemaEngine.GetEntitySchema(tenantID, entityCode)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.handleError(c, errors.NewNotFoundError("entity"))
		} else {
			h.handleError(c, err)
		}
		return
	}

	c.JSON(http.StatusOK, schema)
}

// =============================================================================
// DYNAMIC DATA ENDPOINTS
// =============================================================================

// List returns a paginated list of records
// GET /api/data/:entity
func (h *Handler) List(c *gin.Context) {
	tenantID := c.MustGet("tenant_id").(uuid.UUID)
	entityCode := c.Param("entity")

	// Parse query params
	params := engine.QueryParams{
		Page:     parseIntParam(c.Query("page"), 1),
		PageSize: parseIntParam(c.Query("page_size"), 25),
		Sort:     c.Query("sort"),
		SortDir:  c.Query("sort_dir"),
		Search:   c.Query("search"),
		Filters:  make(map[string]interface{}),
	}

	// Parse filters from query params
	// Format: filter[field]=value
	for key, values := range c.Request.URL.Query() {
		if len(key) > 7 && key[:7] == "filter[" && key[len(key)-1] == ']' {
			fieldName := key[7 : len(key)-1]
			if len(values) > 0 {
				params.Filters[fieldName] = values[0]
			}
		}
	}

	result, err := h.dataEngine.List(tenantID, entityCode, params)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// Get returns a single record
// GET /api/data/:entity/:id
func (h *Handler) Get(c *gin.Context) {
	tenantID := c.MustGet("tenant_id").(uuid.UUID)
	entityCode := c.Param("entity")
	recordID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.handleError(c, errors.NewBadRequestError("invalid id"))
		return
	}

	record, err := h.dataEngine.Get(tenantID, entityCode, recordID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.handleError(c, errors.NewNotFoundError("record"))
		} else {
			h.handleError(c, err)
		}
		return
	}

	c.JSON(http.StatusOK, record)
}

// Create creates a new record
// POST /api/data/:entity
func (h *Handler) Create(c *gin.Context) {
	tenantID := c.MustGet("tenant_id").(uuid.UUID)
	entityCode := c.Param("entity")

	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		h.handleError(c, errors.NewBadRequestError("invalid request body"))
		return
	}

	// Get user ID if available
	var userID *uuid.UUID
	if uid, exists := c.Get("user_id"); exists {
		id := uid.(uuid.UUID)
		userID = &id
	}

	record, err := h.dataEngine.Create(tenantID, entityCode, data, userID)
	if err != nil {
		if strings.Contains(err.Error(), "validation") || strings.Contains(err.Error(), "required") {
			h.handleError(c, errors.NewValidationError("", err.Error()))
		} else {
			h.handleError(c, err)
		}
		return
	}

	c.JSON(http.StatusCreated, record)
}

// Update updates an existing record
// PUT /api/data/:entity/:id
func (h *Handler) Update(c *gin.Context) {
	tenantID := c.MustGet("tenant_id").(uuid.UUID)
	entityCode := c.Param("entity")
	recordID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.handleError(c, errors.NewBadRequestError("invalid id"))
		return
	}

	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		h.handleError(c, errors.NewBadRequestError("invalid request body"))
		return
	}

	// Get user ID if available
	var userID *uuid.UUID
	if uid, exists := c.Get("user_id"); exists {
		id := uid.(uuid.UUID)
		userID = &id
	}

	record, err := h.dataEngine.Update(tenantID, entityCode, recordID, data, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.handleError(c, errors.NewNotFoundError("record"))
		} else if strings.Contains(err.Error(), "validation") {
			h.handleError(c, errors.NewValidationError("", err.Error()))
		} else {
			h.handleError(c, err)
		}
		return
	}

	c.JSON(http.StatusOK, record)
}

// Delete deletes a record
// DELETE /api/data/:entity/:id
func (h *Handler) Delete(c *gin.Context) {
	tenantID := c.MustGet("tenant_id").(uuid.UUID)
	entityCode := c.Param("entity")
	recordID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.handleError(c, errors.NewBadRequestError("invalid id"))
		return
	}

	// Get user ID if available
	var userID *uuid.UUID
	if uid, exists := c.Get("user_id"); exists {
		id := uid.(uuid.UUID)
		userID = &id
	}

	if err := h.dataEngine.Delete(tenantID, entityCode, recordID, userID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.handleError(c, errors.NewNotFoundError("record"))
		} else {
			h.handleError(c, err)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted successfully"})
}

// BulkDelete deletes multiple records
// POST /api/data/:entity/bulk-delete
func (h *Handler) BulkDelete(c *gin.Context) {
	tenantID := c.MustGet("tenant_id").(uuid.UUID)
	entityCode := c.Param("entity")

	var request struct {
		IDs []string `json:"ids"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		h.handleError(c, errors.NewBadRequestError("invalid request body"))
		return
	}

	var recordIDs []uuid.UUID
	for _, idStr := range request.IDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			h.handleError(c, errors.NewBadRequestError("invalid id: "+idStr))
			return
		}
		recordIDs = append(recordIDs, id)
	}

	// Get user ID if available
	var userID *uuid.UUID
	if uid, exists := c.Get("user_id"); exists {
		id := uid.(uuid.UUID)
		userID = &id
	}

	if err := h.dataEngine.BulkDelete(tenantID, entityCode, recordIDs, userID); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted successfully", "count": len(recordIDs)})
}

// =============================================================================
// HEALTH CHECK
// =============================================================================

// Health returns the health status
// GET /api/health
func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "genesis",
		"version": "1.0.0",
	})
}

// =============================================================================
// HELPERS
// =============================================================================

func parseIntParam(value string, defaultValue int) int {
	if value == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return i
}

// handleError handles errors and sends appropriate HTTP responses
func (h *Handler) handleError(c *gin.Context, err error) {
	status, response := errors.ToHTTPError(err)
	c.JSON(status, response)
}
