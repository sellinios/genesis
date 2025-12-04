// Package api - Generator API Handler
package api

import (
	"net/http"

	"github.com/aethra/genesis/internal/generator"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GeneratorHandler handles code generation requests
type GeneratorHandler struct {
	db       *gorm.DB
	compiler *generator.Compiler
}

// NewGeneratorHandler creates a new generator handler
func NewGeneratorHandler(db *gorm.DB) *GeneratorHandler {
	return &GeneratorHandler{
		db:       db,
		compiler: generator.NewCompiler(db),
	}
}

// GetCompiler returns the compiler instance
func (h *GeneratorHandler) GetCompiler() *generator.Compiler {
	return h.compiler
}

// GenerateAll generates components for all entities
func (h *GeneratorHandler) GenerateAll(c *gin.Context) {
	tenantIDStr := c.GetString("tenant_id")
	if tenantIDStr == "" {
		// Get first tenant
		var tenantID uuid.UUID
		h.db.Raw("SELECT id FROM tenants LIMIT 1").Scan(&tenantID)
		tenantIDStr = tenantID.String()
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	// Generate entity components
	if err := h.compiler.GenerateEntityComponents(tenantID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Compile all components
	if err := h.compiler.CompileAll(tenantID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	components := h.compiler.GetAllComponents()

	c.JSON(http.StatusOK, gin.H{
		"message":    "Generation complete",
		"components": len(components),
	})
}

// GetBundle returns the compiled component bundle
func (h *GeneratorHandler) GetBundle(c *gin.Context) {
	bundle := h.compiler.GetComponentBundle()

	c.Header("Content-Type", "application/javascript")
	c.Header("Cache-Control", "public, max-age=3600")
	c.String(http.StatusOK, bundle)
}

// GetComponent returns a specific compiled component
func (h *GeneratorHandler) GetComponent(c *gin.Context) {
	code := c.Param("code")

	comp := h.compiler.GetComponent(code)
	if comp == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Component not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":       comp.Code,
		"source":     comp.SourceJSX,
		"compiled":   comp.CompiledJS,
		"hash":       comp.Hash,
		"compiledAt": comp.CompiledAt,
	})
}

// InvalidateCache clears the component cache
func (h *GeneratorHandler) InvalidateCache(c *gin.Context) {
	h.compiler.InvalidateCache()
	c.JSON(http.StatusOK, gin.H{"message": "Cache invalidated"})
}

// Warmup compiles all components on startup
func (h *GeneratorHandler) Warmup() error {
	var tenantID uuid.UUID
	if err := h.db.Raw("SELECT id FROM tenants LIMIT 1").Scan(&tenantID).Error; err != nil {
		return nil // No tenants yet, skip warmup
	}

	if tenantID == uuid.Nil {
		return nil
	}

	// Generate components for entities
	if err := h.compiler.GenerateEntityComponents(tenantID); err != nil {
		return err
	}

	// Compile all
	return h.compiler.CompileAll(tenantID)
}
