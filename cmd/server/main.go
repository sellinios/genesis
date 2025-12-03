// Genesis Server - The Dynamic Business Platform
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/aethra/genesis/internal/api"
	"github.com/aethra/genesis/internal/auth"
	"github.com/aethra/genesis/internal/engine"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	fmt.Println(`
   ██████╗ ███████╗███╗   ██╗███████╗███████╗██╗███████╗
  ██╔════╝ ██╔════╝████╗  ██║██╔════╝██╔════╝██║██╔════╝
  ██║  ███╗█████╗  ██╔██╗ ██║█████╗  ███████╗██║███████╗
  ██║   ██║██╔══╝  ██║╚██╗██║██╔══╝  ╚════██║██║╚════██║
  ╚██████╔╝███████╗██║ ╚████║███████╗███████║██║███████║
   ╚═════╝ ╚══════╝╚═╝  ╚═══╝╚══════╝╚══════╝╚═╝╚══════╝

  Everything is Data. Data Defines Everything.
  `)

	// Get configuration from environment
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5433")
	dbUser := getEnv("DB_USER", "genesis_user")
	dbPassword := getEnv("DB_PASSWORD", "genesis_pass")
	dbName := getEnv("DB_NAME", "genesis")
	serverPort := getEnv("PORT", "8090")

	// Connect to database
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("✓ Connected to database")

	// Note: Tables are created via SQL migrations in /migrations folder
	// Run: psql -d genesis -f migrations/001_genesis_core.sql
	log.Println("✓ Database ready (using SQL migrations)")

	// Initialize engines
	schemaEngine := engine.NewSchemaEngine(db)
	dataEngine := engine.NewDataEngine(db, schemaEngine)

	log.Println("✓ Engines initialized")

	// Setup permission service
	permissionService := auth.NewPermissionService(db)
	log.Println("✓ Permission service initialized")

	// Setup API handlers
	handler := api.NewHandlerWithPermissions(schemaEngine, dataEngine, permissionService)
	adminHandler := api.NewAdminHandler(db)
	authHandler := api.NewAuthHandler(db)
	router := api.SetupRouter(handler, adminHandler, authHandler)

	log.Println("✓ API routes configured")
	log.Println("  - Auth API: /auth/*")
	log.Println("  - Admin API: /admin/*")
	log.Println("  - Tenant API: /api/*")

	// Start server
	log.Printf("🚀 Genesis server starting on port %s", serverPort)
	if err := router.Run(":" + serverPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
