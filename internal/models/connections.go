// Package models - Database Connections & Web Services
package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// =============================================================================
// DATABASE CONNECTIONS
// =============================================================================

// DataConnection represents an external database connection configuration
type DataConnection struct {
	ID       uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TenantID uuid.UUID `json:"tenant_id" gorm:"type:uuid;index"`

	// Basic info
	Code        string `json:"code" gorm:"not null;size:50"`
	Name        string `json:"name" gorm:"not null;size:100"`
	Description string `json:"description"`

	// Connection type
	Driver string `json:"driver" gorm:"not null;size:30"` // postgres, mysql, sqlite, mongodb, mssql, oracle

	// Connection details
	Host              string `json:"host" gorm:"size:255"`
	Port              int    `json:"port"`
	DatabaseName      string `json:"database_name" gorm:"size:100"`
	Username          string `json:"username" gorm:"size:100"`
	PasswordEncrypted string `json:"-" gorm:"column:password_encrypted"` // Hidden from JSON

	// Connection string (alternative)
	ConnectionStringEncrypted string `json:"-" gorm:"column:connection_string_encrypted"`

	// SSL/TLS settings
	SSLMode     string `json:"ssl_mode" gorm:"size:20;default:'disable'"`
	SSLCert     string `json:"-" gorm:"column:ssl_cert"`
	SSLKey      string `json:"-" gorm:"column:ssl_key"`
	SSLRootCert string `json:"-" gorm:"column:ssl_root_cert"`

	// Connection pool
	MaxOpenConnections    int `json:"max_open_connections" gorm:"default:10"`
	MaxIdleConnections    int `json:"max_idle_connections" gorm:"default:5"`
	ConnectionMaxLifetime int `json:"connection_max_lifetime" gorm:"default:3600"` // seconds

	// Additional options
	Options JSONB `json:"options" gorm:"type:jsonb;default:'{}'"`

	// Status
	IsActive       bool       `json:"is_active" gorm:"default:true"`
	IsDefault      bool       `json:"is_default" gorm:"default:false"`
	LastTestedAt   *time.Time `json:"last_tested_at"`
	LastTestResult string     `json:"last_test_result" gorm:"size:20"`
	LastTestError  string     `json:"last_test_error"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relations
	Tenant *Tenant `json:"tenant,omitempty" gorm:"foreignKey:TenantID"`
}

// TableName specifies the table name for GORM
func (DataConnection) TableName() string {
	return "data_connections"
}

// =============================================================================
// WEB SERVICES
// =============================================================================

// WebService represents an external API/web service configuration
type WebService struct {
	ID       uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TenantID uuid.UUID `json:"tenant_id" gorm:"type:uuid;index"`

	// Basic info
	Code        string `json:"code" gorm:"not null;size:50"`
	Name        string `json:"name" gorm:"not null;size:100"`
	Description string `json:"description"`

	// Service type
	ServiceType string `json:"service_type" gorm:"not null;size:20"` // rest, soap, graphql, webhook

	// Base configuration
	BaseURL string `json:"base_url" gorm:"not null;size:500"`

	// Authentication
	AuthType   string `json:"auth_type" gorm:"size:30"` // none, basic, bearer, api_key, oauth2, custom_header
	AuthConfig JSONB  `json:"auth_config" gorm:"type:jsonb;default:'{}'"`

	// Default headers
	DefaultHeaders JSONB `json:"default_headers" gorm:"type:jsonb;default:'{}'"`

	// Timeout settings (milliseconds)
	TimeoutConnect int `json:"timeout_connect" gorm:"default:5000"`
	TimeoutRead    int `json:"timeout_read" gorm:"default:30000"`
	TimeoutWrite   int `json:"timeout_write" gorm:"default:30000"`

	// Retry configuration
	RetryEnabled       bool          `json:"retry_enabled" gorm:"default:true"`
	RetryMaxAttempts   int           `json:"retry_max_attempts" gorm:"default:3"`
	RetryDelayMs       int           `json:"retry_delay_ms" gorm:"default:1000"`
	RetryOnStatusCodes pq.Int32Array `json:"retry_on_status_codes" gorm:"type:integer[]"`

	// Rate limiting
	RateLimitRequests      *int `json:"rate_limit_requests"`
	RateLimitWindowSeconds *int `json:"rate_limit_window_seconds"`

	// SSL/TLS
	SSLVerify bool   `json:"ssl_verify" gorm:"default:true"`
	SSLCert   string `json:"-" gorm:"column:ssl_cert"`

	// Status
	IsActive       bool       `json:"is_active" gorm:"default:true"`
	LastCalledAt   *time.Time `json:"last_called_at"`
	LastCallResult string     `json:"last_call_result" gorm:"size:20"`
	LastCallError  string     `json:"last_call_error"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relations
	Tenant    *Tenant              `json:"tenant,omitempty" gorm:"foreignKey:TenantID"`
	Endpoints []WebServiceEndpoint `json:"endpoints,omitempty" gorm:"foreignKey:WebServiceID"`
}

// TableName specifies the table name for GORM
func (WebService) TableName() string {
	return "web_services"
}

// =============================================================================
// WEB SERVICE ENDPOINTS
// =============================================================================

// WebServiceEndpoint represents an individual endpoint within a web service
type WebServiceEndpoint struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	WebServiceID uuid.UUID `json:"web_service_id" gorm:"type:uuid;index"`

	// Basic info
	Code        string `json:"code" gorm:"not null;size:50"`
	Name        string `json:"name" gorm:"not null;size:100"`
	Description string `json:"description"`

	// HTTP details
	Method string `json:"method" gorm:"not null;size:10;default:'GET'"` // GET, POST, PUT, PATCH, DELETE
	Path   string `json:"path" gorm:"not null;size:500"`                // Relative path

	// Request configuration
	QueryParams JSONB `json:"query_params" gorm:"type:jsonb;default:'{}'"`
	Headers     JSONB `json:"headers" gorm:"type:jsonb;default:'{}'"`

	// Request body
	RequestBodyType     string `json:"request_body_type" gorm:"size:20"` // json, form, xml, raw
	RequestBodyTemplate string `json:"request_body_template"`

	// Response configuration
	ResponseType    string `json:"response_type" gorm:"size:20;default:'json'"` // json, xml, text, binary
	ResponseMapping JSONB  `json:"response_mapping" gorm:"type:jsonb;default:'{}'"`

	// Success criteria
	SuccessStatusCodes pq.Int32Array `json:"success_status_codes" gorm:"type:integer[]"`
	SuccessCondition   string        `json:"success_condition"`

	// Caching
	CacheEnabled     bool   `json:"cache_enabled" gorm:"default:false"`
	CacheTTLSeconds  int    `json:"cache_ttl_seconds" gorm:"default:300"`
	CacheKeyTemplate string `json:"cache_key_template" gorm:"size:500"`

	// Status
	IsActive bool `json:"is_active" gorm:"default:true"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relations
	WebService *WebService `json:"web_service,omitempty" gorm:"foreignKey:WebServiceID"`
}

// TableName specifies the table name for GORM
func (WebServiceEndpoint) TableName() string {
	return "web_service_endpoints"
}

// =============================================================================
// DATA MAPPINGS
// =============================================================================

// DataMapping represents a mapping between external data and Genesis entities
type DataMapping struct {
	ID       uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TenantID uuid.UUID `json:"tenant_id" gorm:"type:uuid;index"`

	// Basic info
	Code        string `json:"code" gorm:"not null;size:50"`
	Name        string `json:"name" gorm:"not null;size:100"`
	Description string `json:"description"`

	// Source type
	SourceType string `json:"source_type" gorm:"not null;size:20"` // database, web_service

	// Source references
	DataConnectionID     *uuid.UUID `json:"data_connection_id" gorm:"type:uuid"`
	WebServiceEndpointID *uuid.UUID `json:"web_service_endpoint_id" gorm:"type:uuid"`

	// For database sources
	SourceQuery string `json:"source_query"`

	// Target Genesis entity
	TargetEntityID uuid.UUID `json:"target_entity_id" gorm:"type:uuid;index"`

	// Field mappings
	FieldMappings JSONB `json:"field_mappings" gorm:"type:jsonb;not null;default:'[]'"`

	// Sync configuration
	SyncMode      string `json:"sync_mode" gorm:"size:20;default:'manual'"`      // manual, scheduled, realtime, on_demand
	SyncDirection string `json:"sync_direction" gorm:"size:20;default:'import'"` // import, export, bidirectional
	SyncSchedule  string `json:"sync_schedule" gorm:"size:100"`                  // Cron expression

	// Conflict resolution
	ConflictStrategy string         `json:"conflict_strategy" gorm:"size:20;default:'skip'"` // skip, overwrite, merge, error
	UniqueKeyFields  pq.StringArray `json:"unique_key_fields" gorm:"type:text[]"`

	// Filters
	SourceFilter string `json:"source_filter"`

	// Status
	IsActive                 bool       `json:"is_active" gorm:"default:true"`
	LastSyncAt               *time.Time `json:"last_sync_at"`
	LastSyncResult           string     `json:"last_sync_result" gorm:"size:20"`
	LastSyncRecordsProcessed int        `json:"last_sync_records_processed"`
	LastSyncError            string     `json:"last_sync_error"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relations
	Tenant             *Tenant             `json:"tenant,omitempty" gorm:"foreignKey:TenantID"`
	DataConnection     *DataConnection     `json:"data_connection,omitempty" gorm:"foreignKey:DataConnectionID"`
	WebServiceEndpoint *WebServiceEndpoint `json:"web_service_endpoint,omitempty" gorm:"foreignKey:WebServiceEndpointID"`
	TargetEntity       *Entity             `json:"target_entity,omitempty" gorm:"foreignKey:TargetEntityID"`
}

// TableName specifies the table name for GORM
func (DataMapping) TableName() string {
	return "data_mappings"
}

// =============================================================================
// SYNC LOGS
// =============================================================================

// SyncLog represents a synchronization log entry
type SyncLog struct {
	ID            uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TenantID      uuid.UUID `json:"tenant_id" gorm:"type:uuid;index"`
	DataMappingID uuid.UUID `json:"data_mapping_id" gorm:"type:uuid;index"`

	// Sync details
	StartedAt   time.Time  `json:"started_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	CompletedAt *time.Time `json:"completed_at"`

	// Results
	Status         string `json:"status" gorm:"not null;size:20;default:'running'"` // running, success, partial, failed
	RecordsFetched int    `json:"records_fetched" gorm:"default:0"`
	RecordsCreated int    `json:"records_created" gorm:"default:0"`
	RecordsUpdated int    `json:"records_updated" gorm:"default:0"`
	RecordsSkipped int    `json:"records_skipped" gorm:"default:0"`
	RecordsFailed  int    `json:"records_failed" gorm:"default:0"`

	// Errors
	Errors JSONB `json:"errors" gorm:"type:jsonb;default:'[]'"`

	// Triggered by
	TriggeredBy string `json:"triggered_by" gorm:"size:50"`

	CreatedAt time.Time `json:"created_at"`

	// Relations
	DataMapping *DataMapping `json:"data_mapping,omitempty" gorm:"foreignKey:DataMappingID"`
}

// TableName specifies the table name for GORM
func (SyncLog) TableName() string {
	return "sync_logs"
}

// =============================================================================
// SCHEDULED JOBS
// =============================================================================

// ScheduledJob represents a scheduled job configuration
type ScheduledJob struct {
	ID       uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TenantID *uuid.UUID `json:"tenant_id" gorm:"type:uuid;index"` // NULL for system jobs

	// Job info
	Code        string `json:"code" gorm:"not null;size:50"`
	Name        string `json:"name" gorm:"not null;size:100"`
	Description string `json:"description"`

	// Job type and target
	JobType      string     `json:"job_type" gorm:"not null;size:30"` // data_sync, cleanup, report, webhook, custom
	TargetID     *uuid.UUID `json:"target_id" gorm:"type:uuid"`
	TargetConfig JSONB      `json:"target_config" gorm:"type:jsonb;default:'{}'"`

	// Schedule
	Schedule string `json:"schedule" gorm:"not null;size:100"` // Cron expression
	Timezone string `json:"timezone" gorm:"size:50;default:'UTC'"`

	// Execution settings
	TimeoutSeconds int  `json:"timeout_seconds" gorm:"default:3600"`
	RetryOnFailure bool `json:"retry_on_failure" gorm:"default:true"`
	MaxRetries     int  `json:"max_retries" gorm:"default:3"`

	// Status
	IsActive          bool       `json:"is_active" gorm:"default:true"`
	LastRunAt         *time.Time `json:"last_run_at"`
	LastRunStatus     string     `json:"last_run_status" gorm:"size:20"`
	LastRunDurationMs int        `json:"last_run_duration_ms"`
	NextRunAt         *time.Time `json:"next_run_at"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relations
	Tenant *Tenant `json:"tenant,omitempty" gorm:"foreignKey:TenantID"`
}

// TableName specifies the table name for GORM
func (ScheduledJob) TableName() string {
	return "scheduled_jobs"
}
