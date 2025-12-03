// Package api - Router setup
package api

import (
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// SetupRouter creates and configures the Gin router
func SetupRouter(handler *Handler, adminHandler *AdminHandler) *gin.Engine {
	r := gin.Default()

	// CORS configuration - properly configured for security
	// When credentials are used, specific origins must be provided (not *)
	corsConfig := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Tenant-ID", "X-User-ID", "Accept"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	// Get allowed origins from environment or use defaults for development
	allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if allowedOrigins != "" {
		corsConfig.AllowOrigins = strings.Split(allowedOrigins, ",")
	} else {
		// Development defaults - in production, set CORS_ALLOWED_ORIGINS
		corsConfig.AllowOrigins = []string{
			"http://localhost:3000",
			"http://localhost:8080",
			"http://127.0.0.1:3000",
			"http://127.0.0.1:8080",
		}
	}

	r.Use(cors.New(corsConfig))

	// Health check (no auth required)
	r.GET("/api/health", handler.Health)

	// ==========================================================================
	// ADMIN API - For managing Genesis configuration
	// These endpoints are used to configure tenants, modules, entities, fields
	// No tenant context required - these are super-admin operations
	// ==========================================================================
	admin := r.Group("/admin")
	{
		// Tenant management
		admin.GET("/tenants", adminHandler.ListTenants)
		admin.POST("/tenants", adminHandler.CreateTenant)
		admin.GET("/tenants/:id", adminHandler.GetTenant)
		admin.PUT("/tenants/:id", adminHandler.UpdateTenant)
		admin.DELETE("/tenants/:id", adminHandler.DeleteTenant)

		// User management (under tenant)
		admin.GET("/tenants/:tenant_id/users", adminHandler.ListUsers)
		admin.POST("/tenants/:tenant_id/users", adminHandler.CreateUser)

		// Module management (under tenant)
		admin.GET("/tenants/:tenant_id/modules", adminHandler.ListModules)
		admin.POST("/tenants/:tenant_id/modules", adminHandler.CreateModule)
		admin.GET("/modules/:id", adminHandler.GetModule)
		admin.PUT("/modules/:id", adminHandler.UpdateModule)
		admin.DELETE("/modules/:id", adminHandler.DeleteModule)

		// Entity management (under module)
		admin.GET("/modules/:module_id/entities", adminHandler.ListEntities)
		admin.POST("/modules/:module_id/entities", adminHandler.CreateEntity)
		admin.GET("/entities/:id", adminHandler.GetEntity)
		admin.PUT("/entities/:id", adminHandler.UpdateEntity)
		admin.DELETE("/entities/:id", adminHandler.DeleteEntity)

		// Field management (under entity)
		admin.GET("/entities/:entity_id/fields", adminHandler.ListFields)
		admin.POST("/entities/:entity_id/fields", adminHandler.CreateField)
		admin.GET("/fields/:id", adminHandler.GetField)
		admin.PUT("/fields/:id", adminHandler.UpdateField)
		admin.DELETE("/fields/:id", adminHandler.DeleteField)

		// Field types (global)
		admin.GET("/field-types", adminHandler.ListFieldTypes)
	}

	// ==========================================================================
	// TENANT API - For tenant-specific operations
	// These endpoints require tenant context (X-Tenant-ID header)
	// ==========================================================================
	api := r.Group("/api")
	api.Use(handler.TenantMiddleware())
	api.Use(handler.UserMiddleware())
	{
		// Schema endpoints
		api.GET("/schema", handler.GetSchema)
		api.GET("/schema/:entity", handler.GetEntitySchema)

		// Dynamic data endpoints
		data := api.Group("/data")
		{
			data.GET("/:entity", handler.List)
			data.GET("/:entity/:id", handler.Get)
			data.POST("/:entity", handler.Create)
			data.PUT("/:entity/:id", handler.Update)
			data.DELETE("/:entity/:id", handler.Delete)
			data.POST("/:entity/bulk-delete", handler.BulkDelete)
		}
	}

	return r
}
