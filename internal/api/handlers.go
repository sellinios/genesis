// Package api contains the HTTP API handlers for Genesis
package api

import (
	"net/http"
	"strconv"

	"github.com/aethra/genesis/internal/engine"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler contains all API handlers
type Handler struct {
	schemaEngine *engine.SchemaEngine
	dataEngine   *engine.DataEngine
}

// NewHandler creates a new API handler
func NewHandler(schemaEngine *engine.SchemaEngine, dataEngine *engine.DataEngine) *Handler {
	return &Handler{
		schemaEngine: schemaEngine,
		dataEngine:   dataEngine,
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

// UserMiddleware extracts user from JWT token (simplified for now)
func (h *Handler) UserMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Implement JWT validation
		// For now, just get user_id from header
		userIDStr := c.GetHeader("X-User-ID")
		if userIDStr != "" {
			if userID, err := uuid.Parse(userIDStr); err == nil {
				c.Set("user_id", userID)
			}
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	record, err := h.dataEngine.Get(tenantID, entityCode, recordID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	// Get user ID if available
	var userID *uuid.UUID
	if uid, exists := c.Get("user_id"); exists {
		id := uid.(uuid.UUID)
		userID = &id
	}

	if err := h.dataEngine.Delete(tenantID, entityCode, recordID, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	var recordIDs []uuid.UUID
	for _, idStr := range request.IDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id: " + idStr})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
