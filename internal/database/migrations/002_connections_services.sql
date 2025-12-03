-- Genesis Migration 002: Database Connections & Web Services
-- External data sources and API integrations - all configurable via database

-- =============================================================================
-- DATABASE CONNECTIONS
-- Connect to external databases (PostgreSQL, MySQL, SQLite, MongoDB, etc.)
-- =============================================================================

CREATE TABLE IF NOT EXISTS data_connections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Basic info
    code VARCHAR(50) NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,

    -- Connection type
    driver VARCHAR(30) NOT NULL, -- postgres, mysql, sqlite, mongodb, mssql, oracle

    -- Connection details (encrypted in production)
    host VARCHAR(255),
    port INTEGER,
    database_name VARCHAR(100),
    username VARCHAR(100),
    password_encrypted TEXT, -- Should be encrypted

    -- Connection string (alternative to individual fields)
    connection_string_encrypted TEXT,

    -- SSL/TLS settings
    ssl_mode VARCHAR(20) DEFAULT 'disable', -- disable, require, verify-ca, verify-full
    ssl_cert TEXT,
    ssl_key TEXT,
    ssl_root_cert TEXT,

    -- Connection pool settings
    max_open_connections INTEGER DEFAULT 10,
    max_idle_connections INTEGER DEFAULT 5,
    connection_max_lifetime INTEGER DEFAULT 3600, -- seconds

    -- Additional options as JSON
    options JSONB DEFAULT '{}',

    -- Status
    is_active BOOLEAN DEFAULT true,
    is_default BOOLEAN DEFAULT false, -- Default connection for this tenant
    last_tested_at TIMESTAMP,
    last_test_result VARCHAR(20), -- success, failed
    last_test_error TEXT,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(tenant_id, code)
);

-- =============================================================================
-- WEB SERVICES (External APIs)
-- Configure REST, SOAP, GraphQL endpoints
-- =============================================================================

CREATE TABLE IF NOT EXISTS web_services (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Basic info
    code VARCHAR(50) NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,

    -- Service type
    service_type VARCHAR(20) NOT NULL, -- rest, soap, graphql, webhook

    -- Base configuration
    base_url VARCHAR(500) NOT NULL,

    -- Authentication
    auth_type VARCHAR(30), -- none, basic, bearer, api_key, oauth2, custom_header
    auth_config JSONB DEFAULT '{}', -- Stores auth details based on type
    -- For basic: {"username": "...", "password_encrypted": "..."}
    -- For bearer: {"token_encrypted": "..."}
    -- For api_key: {"header_name": "X-API-Key", "key_encrypted": "..."}
    -- For oauth2: {"client_id": "...", "client_secret_encrypted": "...", "token_url": "...", "scope": "..."}
    -- For custom_header: {"headers": {"X-Custom": "value"}}

    -- Default headers
    default_headers JSONB DEFAULT '{}',

    -- Timeout settings (milliseconds)
    timeout_connect INTEGER DEFAULT 5000,
    timeout_read INTEGER DEFAULT 30000,
    timeout_write INTEGER DEFAULT 30000,

    -- Retry configuration
    retry_enabled BOOLEAN DEFAULT true,
    retry_max_attempts INTEGER DEFAULT 3,
    retry_delay_ms INTEGER DEFAULT 1000,
    retry_on_status_codes INTEGER[] DEFAULT '{500,502,503,504}',

    -- Rate limiting
    rate_limit_requests INTEGER, -- requests per window
    rate_limit_window_seconds INTEGER, -- window size

    -- SSL/TLS
    ssl_verify BOOLEAN DEFAULT true,
    ssl_cert TEXT,

    -- Status
    is_active BOOLEAN DEFAULT true,
    last_called_at TIMESTAMP,
    last_call_result VARCHAR(20), -- success, failed
    last_call_error TEXT,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(tenant_id, code)
);

-- =============================================================================
-- WEB SERVICE ENDPOINTS
-- Individual endpoints within a web service
-- =============================================================================

CREATE TABLE IF NOT EXISTS web_service_endpoints (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    web_service_id UUID NOT NULL REFERENCES web_services(id) ON DELETE CASCADE,

    -- Basic info
    code VARCHAR(50) NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,

    -- HTTP details
    method VARCHAR(10) NOT NULL DEFAULT 'GET', -- GET, POST, PUT, PATCH, DELETE
    path VARCHAR(500) NOT NULL, -- Relative path, can include params like /users/{id}

    -- Request configuration
    query_params JSONB DEFAULT '{}', -- Default query parameters
    headers JSONB DEFAULT '{}', -- Endpoint-specific headers (merged with service headers)

    -- Request body template (for POST/PUT/PATCH)
    request_body_type VARCHAR(20), -- json, form, xml, raw
    request_body_template TEXT, -- Template with placeholders {{field_name}}

    -- Response configuration
    response_type VARCHAR(20) DEFAULT 'json', -- json, xml, text, binary
    response_mapping JSONB DEFAULT '{}', -- Map response fields to internal fields
    -- Example: {"customer_name": "data.customer.full_name", "total": "data.order.total_amount"}

    -- Success criteria
    success_status_codes INTEGER[] DEFAULT '{200,201,204}',
    success_condition TEXT, -- JSONPath or expression to validate success

    -- Caching
    cache_enabled BOOLEAN DEFAULT false,
    cache_ttl_seconds INTEGER DEFAULT 300,
    cache_key_template VARCHAR(500), -- Template for cache key

    -- Status
    is_active BOOLEAN DEFAULT true,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(web_service_id, code)
);

-- =============================================================================
-- DATA MAPPINGS
-- Map external data sources to Genesis entities
-- =============================================================================

CREATE TABLE IF NOT EXISTS data_mappings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Basic info
    code VARCHAR(50) NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,

    -- Source type
    source_type VARCHAR(20) NOT NULL, -- database, web_service

    -- Source reference (one of these will be set)
    data_connection_id UUID REFERENCES data_connections(id) ON DELETE CASCADE,
    web_service_endpoint_id UUID REFERENCES web_service_endpoints(id) ON DELETE CASCADE,

    -- For database sources
    source_query TEXT, -- SQL query or table name

    -- Target Genesis entity
    target_entity_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,

    -- Field mappings
    field_mappings JSONB NOT NULL DEFAULT '[]',
    -- Example: [
    --   {"source": "customer_name", "target": "name", "transform": null},
    --   {"source": "created_date", "target": "created_at", "transform": "date"},
    --   {"source": "status_code", "target": "status", "transform": "lookup:status_codes"}
    -- ]

    -- Sync configuration
    sync_mode VARCHAR(20) DEFAULT 'manual', -- manual, scheduled, realtime, on_demand
    sync_direction VARCHAR(20) DEFAULT 'import', -- import, export, bidirectional
    sync_schedule VARCHAR(100), -- Cron expression for scheduled sync

    -- Conflict resolution
    conflict_strategy VARCHAR(20) DEFAULT 'skip', -- skip, overwrite, merge, error
    unique_key_fields TEXT[], -- Fields that identify unique records

    -- Filters
    source_filter TEXT, -- WHERE clause or filter expression

    -- Status
    is_active BOOLEAN DEFAULT true,
    last_sync_at TIMESTAMP,
    last_sync_result VARCHAR(20), -- success, partial, failed
    last_sync_records_processed INTEGER,
    last_sync_error TEXT,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(tenant_id, code)
);

-- =============================================================================
-- SYNC LOGS
-- Track data synchronization history
-- =============================================================================

CREATE TABLE IF NOT EXISTS sync_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    data_mapping_id UUID NOT NULL REFERENCES data_mappings(id) ON DELETE CASCADE,

    -- Sync details
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,

    -- Results
    status VARCHAR(20) NOT NULL DEFAULT 'running', -- running, success, partial, failed
    records_fetched INTEGER DEFAULT 0,
    records_created INTEGER DEFAULT 0,
    records_updated INTEGER DEFAULT 0,
    records_skipped INTEGER DEFAULT 0,
    records_failed INTEGER DEFAULT 0,

    -- Error details
    errors JSONB DEFAULT '[]', -- Array of error objects

    -- Triggered by
    triggered_by VARCHAR(50), -- user:<id>, schedule, api, webhook

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- =============================================================================
-- SCHEDULED JOBS
-- Generic job scheduler for syncs, cleanups, reports, etc.
-- =============================================================================

CREATE TABLE IF NOT EXISTS scheduled_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE, -- NULL for system jobs

    -- Job info
    code VARCHAR(50) NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,

    -- Job type and target
    job_type VARCHAR(30) NOT NULL, -- data_sync, cleanup, report, webhook, custom
    target_id UUID, -- Reference to data_mapping_id, workflow_id, etc.
    target_config JSONB DEFAULT '{}', -- Additional configuration

    -- Schedule (cron expression)
    schedule VARCHAR(100) NOT NULL, -- e.g., "0 0 * * *" for daily at midnight
    timezone VARCHAR(50) DEFAULT 'UTC',

    -- Execution settings
    timeout_seconds INTEGER DEFAULT 3600,
    retry_on_failure BOOLEAN DEFAULT true,
    max_retries INTEGER DEFAULT 3,

    -- Status
    is_active BOOLEAN DEFAULT true,
    last_run_at TIMESTAMP,
    last_run_status VARCHAR(20), -- success, failed
    last_run_duration_ms INTEGER,
    next_run_at TIMESTAMP,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(COALESCE(tenant_id, '00000000-0000-0000-0000-000000000000'::UUID), code)
);

-- =============================================================================
-- INDEXES
-- =============================================================================

CREATE INDEX IF NOT EXISTS idx_data_connections_tenant ON data_connections(tenant_id);
CREATE INDEX IF NOT EXISTS idx_data_connections_driver ON data_connections(driver);

CREATE INDEX IF NOT EXISTS idx_web_services_tenant ON web_services(tenant_id);
CREATE INDEX IF NOT EXISTS idx_web_services_type ON web_services(service_type);

CREATE INDEX IF NOT EXISTS idx_web_service_endpoints_service ON web_service_endpoints(web_service_id);

CREATE INDEX IF NOT EXISTS idx_data_mappings_tenant ON data_mappings(tenant_id);
CREATE INDEX IF NOT EXISTS idx_data_mappings_source_type ON data_mappings(source_type);
CREATE INDEX IF NOT EXISTS idx_data_mappings_target ON data_mappings(target_entity_id);

CREATE INDEX IF NOT EXISTS idx_sync_logs_mapping ON sync_logs(data_mapping_id);
CREATE INDEX IF NOT EXISTS idx_sync_logs_started ON sync_logs(started_at);

CREATE INDEX IF NOT EXISTS idx_scheduled_jobs_tenant ON scheduled_jobs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_scheduled_jobs_next_run ON scheduled_jobs(next_run_at) WHERE is_active = true;

-- =============================================================================
-- TRIGGERS
-- =============================================================================

CREATE TRIGGER update_data_connections_updated_at
    BEFORE UPDATE ON data_connections
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_web_services_updated_at
    BEFORE UPDATE ON web_services
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_web_service_endpoints_updated_at
    BEFORE UPDATE ON web_service_endpoints
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_data_mappings_updated_at
    BEFORE UPDATE ON data_mappings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_scheduled_jobs_updated_at
    BEFORE UPDATE ON scheduled_jobs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
