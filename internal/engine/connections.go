// Package engine - Connection Manager for External Databases
package engine

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/aethra/genesis/internal/models"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"gorm.io/gorm"
)

// ConnectionManager manages external database connections
type ConnectionManager struct {
	db            *gorm.DB
	connections   map[uuid.UUID]*sql.DB
	mu            sync.RWMutex
	encryptionKey []byte
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(db *gorm.DB, encryptionKey string) *ConnectionManager {
	key := make([]byte, 32)
	copy(key, []byte(encryptionKey))

	return &ConnectionManager{
		db:            db,
		connections:   make(map[uuid.UUID]*sql.DB),
		encryptionKey: key,
	}
}

// GetConnection retrieves or creates a connection to an external database
func (cm *ConnectionManager) GetConnection(ctx context.Context, connectionID uuid.UUID) (*sql.DB, error) {
	// Check cache first
	cm.mu.RLock()
	if conn, exists := cm.connections[connectionID]; exists {
		cm.mu.RUnlock()
		// Verify connection is still alive
		if err := conn.PingContext(ctx); err == nil {
			return conn, nil
		}
		// Connection dead, remove from cache
		cm.mu.Lock()
		delete(cm.connections, connectionID)
		cm.mu.Unlock()
	} else {
		cm.mu.RUnlock()
	}

	// Load connection config from database
	var config models.DataConnection
	if err := cm.db.First(&config, "id = ?", connectionID).Error; err != nil {
		return nil, fmt.Errorf("connection not found: %w", err)
	}

	if !config.IsActive {
		return nil, fmt.Errorf("connection is not active")
	}

	// Create new connection
	conn, err := cm.createConnection(ctx, &config)
	if err != nil {
		return nil, err
	}

	// Cache connection
	cm.mu.Lock()
	cm.connections[connectionID] = conn
	cm.mu.Unlock()

	return conn, nil
}

// GetConnectionByCode retrieves a connection by tenant and code
func (cm *ConnectionManager) GetConnectionByCode(ctx context.Context, tenantID uuid.UUID, code string) (*sql.DB, error) {
	var config models.DataConnection
	if err := cm.db.First(&config, "tenant_id = ? AND code = ?", tenantID, code).Error; err != nil {
		return nil, fmt.Errorf("connection not found: %w", err)
	}

	return cm.GetConnection(ctx, config.ID)
}

// createConnection establishes a new database connection
func (cm *ConnectionManager) createConnection(ctx context.Context, config *models.DataConnection) (*sql.DB, error) {
	dsn, err := cm.buildDSN(config)
	if err != nil {
		return nil, err
	}

	driverName := cm.mapDriver(config.Driver)
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(config.MaxOpenConnections)
	db.SetMaxIdleConns(config.MaxIdleConnections)
	db.SetConnMaxLifetime(time.Duration(config.ConnectionMaxLifetime) * time.Second)

	// Verify connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// buildDSN constructs the connection string based on driver type
func (cm *ConnectionManager) buildDSN(config *models.DataConnection) (string, error) {
	// If connection string is provided, use it directly
	if config.ConnectionStringEncrypted != "" {
		decrypted, err := cm.decrypt(config.ConnectionStringEncrypted)
		if err != nil {
			return "", fmt.Errorf("failed to decrypt connection string: %w", err)
		}
		return decrypted, nil
	}

	// Decrypt password
	password := ""
	if config.PasswordEncrypted != "" {
		var err error
		password, err = cm.decrypt(config.PasswordEncrypted)
		if err != nil {
			return "", fmt.Errorf("failed to decrypt password: %w", err)
		}
	}

	switch config.Driver {
	case "postgres":
		return cm.buildPostgresDSN(config, password), nil
	case "mysql":
		return cm.buildMySQLDSN(config, password), nil
	case "mssql":
		return cm.buildMSSQLDSN(config, password), nil
	default:
		return "", fmt.Errorf("unsupported driver: %s", config.Driver)
	}
}

// buildPostgresDSN constructs PostgreSQL connection string
func (cm *ConnectionManager) buildPostgresDSN(config *models.DataConnection, password string) string {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host,
		config.Port,
		config.Username,
		password,
		config.DatabaseName,
		config.SSLMode,
	)
	return dsn
}

// buildMySQLDSN constructs MySQL connection string
func (cm *ConnectionManager) buildMySQLDSN(config *models.DataConnection, password string) string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?parseTime=true",
		config.Username,
		password,
		config.Host,
		config.Port,
		config.DatabaseName,
	)
}

// buildMSSQLDSN constructs Microsoft SQL Server connection string
func (cm *ConnectionManager) buildMSSQLDSN(config *models.DataConnection, password string) string {
	return fmt.Sprintf(
		"server=%s;port=%d;user id=%s;password=%s;database=%s",
		config.Host,
		config.Port,
		config.Username,
		password,
		config.DatabaseName,
	)
}

// mapDriver maps Genesis driver names to Go sql driver names
func (cm *ConnectionManager) mapDriver(driver string) string {
	switch driver {
	case "postgres":
		return "postgres"
	case "mysql":
		return "mysql"
	case "mssql":
		return "sqlserver"
	default:
		return driver
	}
}

// TestConnection tests a database connection configuration
func (cm *ConnectionManager) TestConnection(ctx context.Context, config *models.DataConnection) error {
	conn, err := cm.createConnection(ctx, config)
	if err != nil {
		return err
	}
	defer conn.Close()

	return conn.PingContext(ctx)
}

// UpdateTestResult updates the test result for a connection
func (cm *ConnectionManager) UpdateTestResult(connectionID uuid.UUID, success bool, errorMsg string) error {
	now := time.Now()
	result := "success"
	if !success {
		result = "failed"
	}

	return cm.db.Model(&models.DataConnection{}).
		Where("id = ?", connectionID).
		Updates(map[string]interface{}{
			"last_tested_at":   &now,
			"last_test_result": result,
			"last_test_error":  errorMsg,
		}).Error
}

// ExecuteQuery executes a query on an external database
func (cm *ConnectionManager) ExecuteQuery(ctx context.Context, connectionID uuid.UUID, query string, args ...interface{}) (*sql.Rows, error) {
	conn, err := cm.GetConnection(ctx, connectionID)
	if err != nil {
		return nil, err
	}

	return conn.QueryContext(ctx, query, args...)
}

// ExecuteQueryRow executes a query that returns a single row
func (cm *ConnectionManager) ExecuteQueryRow(ctx context.Context, connectionID uuid.UUID, query string, args ...interface{}) (*sql.Row, error) {
	conn, err := cm.GetConnection(ctx, connectionID)
	if err != nil {
		return nil, err
	}

	return conn.QueryRowContext(ctx, query, args...), nil
}

// CloseConnection closes a specific connection
func (cm *ConnectionManager) CloseConnection(connectionID uuid.UUID) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if conn, exists := cm.connections[connectionID]; exists {
		delete(cm.connections, connectionID)
		return conn.Close()
	}
	return nil
}

// CloseAll closes all managed connections
func (cm *ConnectionManager) CloseAll() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for id, conn := range cm.connections {
		conn.Close()
		delete(cm.connections, id)
	}
}

// Encrypt encrypts a string using AES-GCM
func (cm *ConnectionManager) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(cm.encryptionKey)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts a string using AES-GCM
func (cm *ConnectionManager) decrypt(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(cm.encryptionKey)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// SaveConnection saves a connection configuration with encrypted credentials
func (cm *ConnectionManager) SaveConnection(config *models.DataConnection, password string, connectionString string) error {
	if password != "" {
		encrypted, err := cm.Encrypt(password)
		if err != nil {
			return fmt.Errorf("failed to encrypt password: %w", err)
		}
		config.PasswordEncrypted = encrypted
	}

	if connectionString != "" {
		encrypted, err := cm.Encrypt(connectionString)
		if err != nil {
			return fmt.Errorf("failed to encrypt connection string: %w", err)
		}
		config.ConnectionStringEncrypted = encrypted
	}

	return cm.db.Save(config).Error
}

// ListConnections lists all connections for a tenant
func (cm *ConnectionManager) ListConnections(tenantID uuid.UUID) ([]models.DataConnection, error) {
	var connections []models.DataConnection
	if err := cm.db.Where("tenant_id = ?", tenantID).Find(&connections).Error; err != nil {
		return nil, err
	}
	return connections, nil
}

// GetDefaultConnection returns the default connection for a tenant
func (cm *ConnectionManager) GetDefaultConnection(ctx context.Context, tenantID uuid.UUID) (*sql.DB, error) {
	var config models.DataConnection
	if err := cm.db.First(&config, "tenant_id = ? AND is_default = true", tenantID).Error; err != nil {
		return nil, fmt.Errorf("no default connection found: %w", err)
	}

	return cm.GetConnection(ctx, config.ID)
}
