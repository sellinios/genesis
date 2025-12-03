// Package config provides configuration management for Genesis
package config

import (
	"crypto/rand"
	"encoding/base64"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SystemConfig represents a configuration entry stored in database
type SystemConfig struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Key       string    `gorm:"uniqueIndex;not null;size:100"`
	Value     string    `gorm:"type:text"`
	ValueType string    `gorm:"size:20"` // string, int, bool, json
	Category  string    `gorm:"size:50;index"`
	IsSecret  bool      `gorm:"default:false"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// TableName returns the table name for SystemConfig
func (SystemConfig) TableName() string {
	return "system_config"
}

// SystemSetup tracks installation status
type SystemSetup struct {
	ID             uuid.UUID  `gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	SetupCompleted bool       `gorm:"default:false"`
	SetupStep      int        `gorm:"default:0"`
	SuperAdminID   *uuid.UUID `gorm:"type:uuid"`
	InstalledAt    *time.Time
	GenesisVersion string `gorm:"size:20"`
	CreatedAt      time.Time
}

// TableName returns the table name for SystemSetup
func (SystemSetup) TableName() string {
	return "system_setup"
}

// ConfigService manages configuration
type ConfigService struct {
	db    *gorm.DB
	cache map[string]string
	mu    sync.RWMutex
}

// NewConfigService creates a new config service
func NewConfigService(db *gorm.DB) *ConfigService {
	svc := &ConfigService{
		db:    db,
		cache: make(map[string]string),
	}
	svc.loadCache()
	return svc
}

// loadCache loads all config values into memory
func (s *ConfigService) loadCache() {
	var configs []SystemConfig
	if err := s.db.Find(&configs).Error; err != nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, cfg := range configs {
		s.cache[cfg.Key] = cfg.Value
	}
}

// Get returns a config value by key
func (s *ConfigService) Get(key string) string {
	// Check environment variable override first
	if envVal := os.Getenv("GENESIS_" + key); envVal != "" {
		return envVal
	}

	s.mu.RLock()
	if val, ok := s.cache[key]; ok {
		s.mu.RUnlock()
		return val
	}
	s.mu.RUnlock()

	// Try database
	var cfg SystemConfig
	if err := s.db.Where("key = ?", key).First(&cfg).Error; err == nil {
		s.mu.Lock()
		s.cache[key] = cfg.Value
		s.mu.Unlock()
		return cfg.Value
	}

	return ""
}

// GetWithDefault returns a config value or default if not found
func (s *ConfigService) GetWithDefault(key, defaultValue string) string {
	if val := s.Get(key); val != "" {
		return val
	}
	return defaultValue
}

// GetInt returns a config value as int
func (s *ConfigService) GetInt(key string, defaultValue int) int {
	val := s.Get(key)
	if val == "" {
		return defaultValue
	}
	if i, err := strconv.Atoi(val); err == nil {
		return i
	}
	return defaultValue
}

// GetBool returns a config value as bool
func (s *ConfigService) GetBool(key string, defaultValue bool) bool {
	val := s.Get(key)
	if val == "" {
		return defaultValue
	}
	return val == "true" || val == "1" || val == "yes"
}

// Set sets a config value
func (s *ConfigService) Set(key, value, category string, isSecret bool) error {
	cfg := SystemConfig{
		Key:       key,
		Value:     value,
		ValueType: "string",
		Category:  category,
		IsSecret:  isSecret,
		UpdatedAt: time.Now(),
	}

	// Upsert
	err := s.db.Where("key = ?", key).Assign(cfg).FirstOrCreate(&cfg).Error
	if err != nil {
		return err
	}

	// Update cache
	s.mu.Lock()
	s.cache[key] = value
	s.mu.Unlock()

	return nil
}

// Delete removes a config value
func (s *ConfigService) Delete(key string) error {
	err := s.db.Where("key = ?", key).Delete(&SystemConfig{}).Error
	if err != nil {
		return err
	}

	s.mu.Lock()
	delete(s.cache, key)
	s.mu.Unlock()

	return nil
}

// GetCategory returns all config values for a category
func (s *ConfigService) GetCategory(category string) map[string]string {
	var configs []SystemConfig
	result := make(map[string]string)

	if err := s.db.Where("category = ?", category).Find(&configs).Error; err != nil {
		return result
	}

	for _, cfg := range configs {
		result[cfg.Key] = cfg.Value
	}

	return result
}

// GetAllConfig returns all non-secret configuration
func (s *ConfigService) GetAllConfig() map[string]string {
	var configs []SystemConfig
	result := make(map[string]string)

	if err := s.db.Where("is_secret = false").Find(&configs).Error; err != nil {
		return result
	}

	for _, cfg := range configs {
		result[cfg.Key] = cfg.Value
	}

	return result
}

// IsSetupComplete checks if initial setup is complete
func (s *ConfigService) IsSetupComplete() bool {
	var setup SystemSetup
	if err := s.db.First(&setup).Error; err != nil {
		return false
	}
	return setup.SetupCompleted
}

// GetSetupStatus returns the current setup status
func (s *ConfigService) GetSetupStatus() (*SystemSetup, error) {
	var setup SystemSetup
	if err := s.db.First(&setup).Error; err != nil {
		// Create initial setup record if not exists
		if err == gorm.ErrRecordNotFound {
			setup = SystemSetup{
				ID:             uuid.New(),
				SetupCompleted: false,
				SetupStep:      0,
				CreatedAt:      time.Now(),
			}
			if err := s.db.Create(&setup).Error; err != nil {
				return nil, err
			}
			return &setup, nil
		}
		return nil, err
	}
	return &setup, nil
}

// CompleteSetup marks setup as complete
func (s *ConfigService) CompleteSetup(superAdminID uuid.UUID, version string) error {
	now := time.Now()
	return s.db.Model(&SystemSetup{}).
		Where("1=1").
		Updates(map[string]interface{}{
			"setup_completed": true,
			"super_admin_id":  superAdminID,
			"installed_at":    now,
			"genesis_version": version,
		}).Error
}

// GenerateJWTSecret generates a secure random JWT secret
func GenerateJWTSecret() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "genesis-fallback-secret-" + uuid.New().String()
	}
	return base64.URLEncoding.EncodeToString(bytes)
}

// SetupDefaultConfig sets up default configuration values
func (s *ConfigService) SetupDefaultConfig() error {
	defaults := map[string]struct {
		value    string
		category string
		secret   bool
	}{
		// Server
		"SERVER_PORT":          {"8090", "server", false},
		"SERVER_MODE":          {"debug", "server", false},
		"SERVER_READ_TIMEOUT":  {"30", "server", false},
		"SERVER_WRITE_TIMEOUT": {"30", "server", false},

		// Auth
		"JWT_SECRET":         {GenerateJWTSecret(), "auth", true},
		"JWT_ACCESS_EXPIRY":  {"24", "auth", false},
		"JWT_REFRESH_EXPIRY": {"168", "auth", false}, // 7 days in hours

		// CORS
		"CORS_ALLOWED_ORIGINS":   {"http://localhost:3000,http://localhost:8080", "cors", false},
		"CORS_ALLOW_CREDENTIALS": {"true", "cors", false},
	}

	for key, cfg := range defaults {
		// Only set if not already set
		if s.Get(key) == "" {
			if err := s.Set(key, cfg.value, cfg.category, cfg.secret); err != nil {
				return err
			}
		}
	}

	return nil
}

// Config holds the runtime configuration
type Config struct {
	Server   ServerConfig
	Auth     AuthConfig
	CORS     CORSConfig
	Database DatabaseConfig
}

// ServerConfig holds server settings
type ServerConfig struct {
	Port         string
	Mode         string
	ReadTimeout  int
	WriteTimeout int
}

// AuthConfig holds authentication settings
type AuthConfig struct {
	JWTSecret     string
	AccessExpiry  int
	RefreshExpiry int
}

// CORSConfig holds CORS settings
type CORSConfig struct {
	AllowedOrigins   []string
	AllowCredentials bool
}

// DatabaseConfig holds database settings
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

// LoadConfig loads configuration from database into a Config struct
func (s *ConfigService) LoadConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         s.GetWithDefault("SERVER_PORT", "8090"),
			Mode:         s.GetWithDefault("SERVER_MODE", "debug"),
			ReadTimeout:  s.GetInt("SERVER_READ_TIMEOUT", 30),
			WriteTimeout: s.GetInt("SERVER_WRITE_TIMEOUT", 30),
		},
		Auth: AuthConfig{
			JWTSecret:     s.GetWithDefault("JWT_SECRET", ""),
			AccessExpiry:  s.GetInt("JWT_ACCESS_EXPIRY", 24),
			RefreshExpiry: s.GetInt("JWT_REFRESH_EXPIRY", 168),
		},
		CORS: CORSConfig{
			AllowedOrigins:   splitString(s.GetWithDefault("CORS_ALLOWED_ORIGINS", "http://localhost:3000")),
			AllowCredentials: s.GetBool("CORS_ALLOW_CREDENTIALS", true),
		},
	}
}

// splitString splits a comma-separated string into a slice
func splitString(s string) []string {
	if s == "" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
