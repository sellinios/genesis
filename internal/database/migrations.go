// Package database provides database utilities including migrations
package database

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// MigrationRecord tracks which migrations have been applied
type MigrationRecord struct {
	ID        uint      `gorm:"primaryKey"`
	Name      string    `gorm:"uniqueIndex;size:255"`
	AppliedAt time.Time `gorm:"autoCreateTime"`
}

// TableName returns the table name for migrations
func (MigrationRecord) TableName() string {
	return "_genesis_migrations"
}

// RunMigrations executes all pending SQL migrations
func RunMigrations(db *gorm.DB) error {
	// Ensure migrations table exists
	if err := db.AutoMigrate(&MigrationRecord{}); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get list of migration files
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Sort files by name (001_, 002_, etc.)
	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)

	// Run each migration
	for _, file := range files {
		// Check if already applied
		var count int64
		db.Model(&MigrationRecord{}).Where("name = ?", file).Count(&count)
		if count > 0 {
			log.Printf("  ✓ Migration %s already applied", file)
			continue
		}

		// Read migration file
		content, err := fs.ReadFile(migrationsFS, "migrations/"+file)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", file, err)
		}

		// Execute migration
		log.Printf("  → Applying migration %s...", file)
		if err := db.Exec(string(content)).Error; err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", file, err)
		}

		// Record migration
		if err := db.Create(&MigrationRecord{Name: file}).Error; err != nil {
			return fmt.Errorf("failed to record migration %s: %w", file, err)
		}

		log.Printf("  ✓ Migration %s applied successfully", file)
	}

	return nil
}
