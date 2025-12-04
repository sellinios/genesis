// Genesis - The Dynamic Business Platform
package main

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aethra/genesis/internal/api"
	"github.com/aethra/genesis/internal/auth"
	"github.com/aethra/genesis/internal/database"
	"github.com/aethra/genesis/internal/engine"
	"github.com/aethra/genesis/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var Version = "1.0.0"

func main() {
	if len(os.Args) > 1 {
		runCLI()
		return
	}
	startServer()
}

func startServer() {
	fmt.Printf("Genesis %s - Starting...\n", Version)

	db := connectDB()
	log.Println("Database connected")

	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
	log.Println("Migrations complete")

	schemaEngine := engine.NewSchemaEngine(db)
	dataEngine := engine.NewDataEngine(db, schemaEngine)
	permissionService := auth.NewPermissionService(db)

	handler := api.NewHandlerWithPermissions(schemaEngine, dataEngine, permissionService)
	adminHandler := api.NewAdminHandler(db)
	authHandler := api.NewAuthHandler(db)
	setupHandler := api.NewSetupHandler(db)
	adminPanelHandler := api.NewAdminPanelHandler(db)
	uiHandler := api.NewUIHandler(db)
	router := api.SetupRouter(handler, adminHandler, authHandler, setupHandler, adminPanelHandler, uiHandler)

	port := getEnv("PORT", "8090")
	log.Printf("Server starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func connectDB() *gorm.DB {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		requireEnv("DB_HOST"),
		requireEnv("DB_PORT"),
		requireEnv("DB_USER"),
		requireEnv("DB_PASSWORD"),
		requireEnv("DB_NAME"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	return db
}

func requireEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Missing required env: %s", key)
	}
	return value
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// CLI
func runCLI() {
	cmd := os.Args[1]
	switch cmd {
	case "serve":
		startServer()
	case "setup":
		runSetup()
	case "migrate":
		db := connectDB()
		if err := database.RunMigrations(db); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		fmt.Println("Migrations complete")
	case "tenant":
		runTenantCmd()
	case "user":
		runUserCmd()
	default:
		printUsage()
	}
}

func printUsage() {
	fmt.Println(`Usage: genesis <command>
Commands:
  setup                         Interactive setup wizard
  serve                         Start server
  migrate                       Run migrations
  tenant list                   List tenants
  tenant create --code= --name= Create tenant
  user list --tenant=           List users
  user create --tenant= --email= --password= Create user`)
}

func runTenantCmd() {
	if len(os.Args) < 3 {
		printUsage()
		return
	}
	db := connectDB()
	switch os.Args[2] {
	case "list":
		var tenants []models.Tenant
		db.Find(&tenants)
		for _, t := range tenants {
			fmt.Printf("%s - %s\n", t.Code, t.Name)
		}
	case "create":
		code, name := getFlag("--code"), getFlag("--name")
		if code == "" || name == "" {
			printUsage()
			return
		}
		if err := db.Create(&models.Tenant{Code: code, Name: name, IsActive: true}).Error; err != nil {
			log.Fatalf("Failed: %v", err)
		}
		fmt.Printf("Tenant created: %s\n", code)
	case "delete":
		code := getFlag("--code")
		if code == "" {
			printUsage()
			return
		}
		db.Where("code = ?", code).Delete(&models.Tenant{})
		fmt.Printf("Tenant deleted: %s\n", code)
	}
}

func runUserCmd() {
	if len(os.Args) < 3 {
		printUsage()
		return
	}
	db := connectDB()
	switch os.Args[2] {
	case "list":
		tenantCode := getFlag("--tenant")
		if tenantCode == "" {
			printUsage()
			return
		}
		var tenant models.Tenant
		if db.Where("code = ?", tenantCode).First(&tenant).Error != nil {
			log.Fatal("Tenant not found")
		}
		var users []models.User
		db.Where("tenant_id = ?", tenant.ID).Find(&users)
		for _, u := range users {
			fmt.Printf("%s <%s>\n", u.FirstName+" "+u.LastName, u.Email)
		}
	case "create":
		tenantCode := getFlag("--tenant")
		email := getFlag("--email")
		password := getFlag("--password")
		if tenantCode == "" || email == "" || password == "" {
			printUsage()
			return
		}
		var tenant models.Tenant
		if db.Where("code = ?", tenantCode).First(&tenant).Error != nil {
			log.Fatal("Tenant not found")
		}
		hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err := db.Create(&models.User{
			TenantID:     tenant.ID,
			Email:        email,
			PasswordHash: string(hash),
			FirstName:    getFlag("--first"),
			LastName:     getFlag("--last"),
			IsActive:     true,
		}).Error; err != nil {
			log.Fatalf("Failed: %v", err)
		}
		fmt.Printf("User created: %s\n", email)
	}
}

func getFlag(name string) string {
	prefix := name + "="
	for _, arg := range os.Args {
		if len(arg) > len(prefix) && arg[:len(prefix)] == prefix {
			return arg[len(prefix):]
		}
	}
	return ""
}

// Interactive Setup
func runSetup() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("\n=== Genesis Setup Wizard ===\n")

	// Database configuration
	fmt.Println("Database Configuration:")
	dbHost := prompt(reader, "  DB Host", "localhost")
	dbPort := prompt(reader, "  DB Port", "5432")
	dbUser := prompt(reader, "  DB User", "genesis")
	dbPassword := promptPassword(reader, "  DB Password")
	dbName := prompt(reader, "  DB Name", "genesis")

	// Set environment and test connection
	os.Setenv("DB_HOST", dbHost)
	os.Setenv("DB_PORT", dbPort)
	os.Setenv("DB_USER", dbUser)
	os.Setenv("DB_PASSWORD", dbPassword)
	os.Setenv("DB_NAME", dbName)

	fmt.Println("\nConnecting to database...")
	db := connectDB()
	fmt.Println("Connected!")

	// Run migrations
	fmt.Println("Running migrations...")
	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
	fmt.Println("Migrations complete!")

	// Create tenant
	fmt.Println("\nTenant Configuration:")
	tenantCode := prompt(reader, "  Tenant Code", "")
	tenantName := prompt(reader, "  Tenant Name", "")

	tenant := models.Tenant{Code: tenantCode, Name: tenantName, IsActive: true}
	if err := db.Create(&tenant).Error; err != nil {
		log.Fatalf("Failed to create tenant: %v", err)
	}
	fmt.Printf("Tenant '%s' created!\n", tenantCode)

	// Create admin user
	fmt.Println("\nAdmin User:")
	adminEmail := prompt(reader, "  Email", "admin@"+tenantCode+".com")
	adminPassword := promptPassword(reader, "  Password")
	adminFirst := prompt(reader, "  First Name", "Admin")
	adminLast := prompt(reader, "  Last Name", "User")

	hash, _ := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	user := models.User{
		TenantID:     tenant.ID,
		Email:        adminEmail,
		PasswordHash: string(hash),
		FirstName:    adminFirst,
		LastName:     adminLast,
		IsActive:     true,
	}
	if err := db.Create(&user).Error; err != nil {
		log.Fatalf("Failed to create user: %v", err)
	}
	fmt.Printf("Admin user '%s' created!\n", adminEmail)

	// Generate JWT secret
	jwtSecret := generateSecret(32)
	encryptionKey := generateSecret(32)

	// Server config
	fmt.Println("\nServer Configuration:")
	port := prompt(reader, "  Port", "8090")

	// Print systemd environment
	fmt.Println("\n=== Setup Complete ===")
	fmt.Println("\nAdd these to your systemd service or docker-compose:")
	fmt.Println("----------------------------------------")
	fmt.Printf("DB_HOST=%s\n", dbHost)
	fmt.Printf("DB_PORT=%s\n", dbPort)
	fmt.Printf("DB_USER=%s\n", dbUser)
	fmt.Printf("DB_PASSWORD=%s\n", dbPassword)
	fmt.Printf("DB_NAME=%s\n", dbName)
	fmt.Printf("JWT_SECRET=%s\n", jwtSecret)
	fmt.Printf("ENCRYPTION_KEY=%s\n", encryptionKey)
	fmt.Printf("PORT=%s\n", port)
	fmt.Println("----------------------------------------")
	fmt.Printf("\nStart server: genesis serve\n")
	fmt.Printf("Login: %s / [your password]\n", adminEmail)
}

func prompt(reader *bufio.Reader, label, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", label, defaultVal)
	} else {
		fmt.Printf("%s: ", label)
	}
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}

func promptPassword(reader *bufio.Reader, label string) string {
	fmt.Printf("%s: ", label)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func generateSecret(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)[:length]
}
