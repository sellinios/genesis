// Package api - Router setup
package api

import (
	"os"
	"strings"
	"time"

	"github.com/aethra/genesis/internal/auth"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// SetupRouter creates and configures the Gin router
func SetupRouter(handler *Handler, adminHandler *AdminHandler, authHandler *AuthHandler, setupHandler *SetupHandler, adminPanelHandler *AdminPanelHandler, uiHandler *UIHandler) *gin.Engine {
	r := gin.Default()

	// Setup wizard (only shows if no tenants exist)
	// After setup, redirects to admin panel
	r.GET("/", setupHandler.SetupPage)
	r.GET("/setup", setupHandler.SetupPage)
	r.POST("/setup", setupHandler.DoSetup)

	// Admin Panel UI (for config)
	r.GET("/panel", adminPanelHandler.AdminPanel)

	// Dynamic App UI (React from database)
	r.GET("/login", uiHandler.LoginPage)
	r.GET("/app", uiHandler.AppPage)
	r.GET("/app/*path", uiHandler.AppPage)

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
	// AUTH API - Authentication endpoints (no auth required)
	// ==========================================================================
	authRoutes := r.Group("/auth")
	{
		authRoutes.POST("/login", authHandler.Login)
		authRoutes.POST("/register", authHandler.Register)
		authRoutes.POST("/refresh", authHandler.RefreshToken)
	}

	// Authenticated auth endpoints
	authProtected := r.Group("/auth")
	authProtected.Use(handler.UserMiddleware())
	authProtected.Use(handler.RequireAuthMiddleware())
	{
		authProtected.GET("/me", authHandler.GetMe)
		authProtected.POST("/change-password", authHandler.ChangePassword)
		authProtected.POST("/logout", authHandler.Logout)
	}

	// ==========================================================================
	// ADMIN API - For managing Genesis configuration
	// These endpoints are used to configure tenants, modules, entities, fields
	// Requires authentication with admin or super_admin role
	// ==========================================================================
	admin := r.Group("/admin")
	admin.Use(handler.UserMiddleware())
	admin.Use(handler.RequireAuthMiddleware())
	admin.Use(handler.RequireAdminMiddleware())
	{
		// Tenant management
		admin.GET("/tenants", adminHandler.ListTenants)
		admin.POST("/tenants", adminHandler.CreateTenant)
		admin.GET("/tenants/:id", adminHandler.GetTenant)
		admin.PUT("/tenants/:id", adminHandler.UpdateTenant)
		admin.DELETE("/tenants/:id", adminHandler.DeleteTenant)

		// User management
		admin.GET("/users", adminHandler.ListUsers)
		admin.POST("/users", adminHandler.CreateUser)

		// Module management
		admin.GET("/modules", adminHandler.ListModules)
		admin.POST("/modules", adminHandler.CreateModule)
		admin.GET("/modules/:id", adminHandler.GetModule)
		admin.PUT("/modules/:id", adminHandler.UpdateModule)
		admin.DELETE("/modules/:id", adminHandler.DeleteModule)

		// Entity management
		admin.GET("/entities", adminHandler.ListEntities)
		admin.POST("/entities", adminHandler.CreateEntity)
		admin.GET("/entities/:id", adminHandler.GetEntity)
		admin.PUT("/entities/:id", adminHandler.UpdateEntity)
		admin.DELETE("/entities/:id", adminHandler.DeleteEntity)

		// Field management
		admin.GET("/fields", adminHandler.ListFields)
		admin.POST("/fields", adminHandler.CreateField)
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

		// Dynamic data endpoints with permission checking
		// Read operations
		data := api.Group("/data")
		{
			// View permission required for listing and getting
			data.GET("/:entity", handler.PermissionMiddleware(auth.ActionView), handler.List)
			data.GET("/:entity/:id", handler.PermissionMiddleware(auth.ActionView), handler.Get)

			// Create permission required
			data.POST("/:entity", handler.PermissionMiddleware(auth.ActionCreate), handler.Create)

			// Edit permission required
			data.PUT("/:entity/:id", handler.PermissionMiddleware(auth.ActionEdit), handler.Update)

			// Delete permission required
			data.DELETE("/:entity/:id", handler.PermissionMiddleware(auth.ActionDelete), handler.Delete)
			data.POST("/:entity/bulk-delete", handler.PermissionMiddleware(auth.ActionDelete), handler.BulkDelete)
		}
	}

	return r
}
