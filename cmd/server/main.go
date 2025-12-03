// Genesis Server - The Dynamic Business Platform
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/aethra/genesis/internal/api"
	"github.com/aethra/genesis/internal/auth"
	"github.com/aethra/genesis/internal/database"
	"github.com/aethra/genesis/internal/engine"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Version is set at build time
var Version = "dev"

func main() {
	fmt.Printf(`
   ██████╗ ███████╗███╗   ██╗███████╗███████╗██╗███████╗
  ██╔════╝ ██╔════╝████╗  ██║██╔════╝██╔════╝██║██╔════╝
  ██║  ███╗█████╗  ██╔██╗ ██║█████╗  ███████╗██║███████╗
  ██║   ██║██╔══╝  ██║╚██╗██║██╔══╝  ╚════██║██║╚════██║
  ╚██████╔╝███████╗██║ ╚████║███████╗███████║██║███████║
   ╚═════╝ ╚══════╝╚═╝  ╚═══╝╚══════╝╚══════╝╚═╝╚══════╝

  Everything is Data. Data Defines Everything.
  Version: %s
`, Version)

	// Get configuration from environment
	// Defaults match docker-compose.yml for consistency
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "genesis")
	dbPassword := getEnv("DB_PASSWORD", "genesis_secret_change_me")
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

	// Run embedded migrations
	log.Println("→ Running database migrations...")
	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("✓ Database migrations complete")

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
