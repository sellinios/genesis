-- ============================================================================
-- GENESIS CORE SCHEMA
-- "Everything is Data. Data Defines Everything."
-- ============================================================================

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================================
-- SYSTEM TABLES (Genesis Internal)
-- ============================================================================

-- ----------------------------------------------------------------------------
-- Tenants: Κάθε πελάτης/εταιρεία
-- ----------------------------------------------------------------------------
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code VARCHAR(50) UNIQUE NOT NULL,           -- 'aethra', 'acme', etc.
    name VARCHAR(255) NOT NULL,
    domain VARCHAR(255),                        -- 'aethra.genesis.app'
    settings JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ----------------------------------------------------------------------------
-- Users: System users
-- ----------------------------------------------------------------------------
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255),
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    avatar_url TEXT,
    settings JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT TRUE,
    last_login_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, email)
);

-- ============================================================================
-- META-SCHEMA: Ορισμός Entities (Δυναμικά Tables)
-- ============================================================================

-- ----------------------------------------------------------------------------
-- Modules: Ομαδοποίηση entities (CRM, ERP, HRM, etc.)
-- ----------------------------------------------------------------------------
CREATE TABLE modules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    code VARCHAR(50) NOT NULL,                  -- 'crm', 'erp', 'hrm'
    name VARCHAR(100) NOT NULL,                 -- 'Customer Relations'
    description TEXT,
    icon VARCHAR(50),                           -- 'users', 'briefcase'
    color VARCHAR(20),                          -- '#3B82F6'
    display_order INT DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, code)
);

-- ----------------------------------------------------------------------------
-- Entities: Δυναμικά "tables" - ο πελάτης ορίζει τι θέλει
-- ----------------------------------------------------------------------------
CREATE TABLE entities (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    module_id UUID REFERENCES modules(id) ON DELETE SET NULL,

    code VARCHAR(50) NOT NULL,                  -- 'customer', 'order', 'product'
    name VARCHAR(100) NOT NULL,                 -- 'Customer', 'Order'
    name_plural VARCHAR(100),                   -- 'Customers', 'Orders'
    description TEXT,

    -- Table settings
    table_name VARCHAR(100),                    -- Actual table name: 'data_customer'

    -- Display settings
    icon VARCHAR(50),                           -- 'user', 'shopping-cart'
    color VARCHAR(20),

    -- Behavior
    is_system BOOLEAN DEFAULT FALSE,            -- System entity, can't delete
    is_active BOOLEAN DEFAULT TRUE,
    allow_create BOOLEAN DEFAULT TRUE,
    allow_edit BOOLEAN DEFAULT TRUE,
    allow_delete BOOLEAN DEFAULT TRUE,

    -- Soft delete
    use_soft_delete BOOLEAN DEFAULT TRUE,

    -- Timestamps
    use_timestamps BOOLEAN DEFAULT TRUE,

    -- Audit
    use_audit_log BOOLEAN DEFAULT TRUE,

    -- Settings
    settings JSONB DEFAULT '{}',

    display_order INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(tenant_id, code)
);

-- ----------------------------------------------------------------------------
-- Field Types: Τύποι πεδίων που υποστηρίζει το Genesis
-- ----------------------------------------------------------------------------
CREATE TABLE field_types (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code VARCHAR(50) UNIQUE NOT NULL,           -- 'string', 'integer', 'email'
    name VARCHAR(100) NOT NULL,
    category VARCHAR(50),                       -- 'basic', 'advanced', 'relation'

    -- Database mapping
    db_type VARCHAR(50) NOT NULL,               -- 'VARCHAR', 'INTEGER', 'JSONB'
    db_default_length INT,                      -- 255 for VARCHAR

    -- Validation
    validation_rules JSONB DEFAULT '{}',        -- Default validation

    -- UI Component
    default_component VARCHAR(50),              -- 'TextInput', 'Select'
    component_options JSONB DEFAULT '{}',

    -- Settings
    settings JSONB DEFAULT '{}',

    is_system BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ----------------------------------------------------------------------------
-- Fields: Πεδία κάθε entity
-- ----------------------------------------------------------------------------
CREATE TABLE fields (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    entity_id UUID REFERENCES entities(id) ON DELETE CASCADE,
    field_type_id UUID REFERENCES field_types(id),

    code VARCHAR(50) NOT NULL,                  -- 'first_name', 'email', 'status'
    name VARCHAR(100) NOT NULL,                 -- 'First Name', 'Email'
    description TEXT,

    -- Database column settings
    column_name VARCHAR(100),                   -- Actual column: 'first_name'
    column_type VARCHAR(50),                    -- Override: 'VARCHAR(500)'

    -- Constraints
    is_required BOOLEAN DEFAULT FALSE,
    is_unique BOOLEAN DEFAULT FALSE,
    is_primary BOOLEAN DEFAULT FALSE,
    is_auto BOOLEAN DEFAULT FALSE,              -- Auto-increment, UUID, etc.
    default_value TEXT,

    -- Validation
    min_length INT,
    max_length INT,
    min_value NUMERIC,
    max_value NUMERIC,
    regex_pattern VARCHAR(255),
    validation_rules JSONB DEFAULT '{}',

    -- Display
    display_order INT DEFAULT 0,
    in_list BOOLEAN DEFAULT FALSE,              -- Show in list view
    in_detail BOOLEAN DEFAULT TRUE,             -- Show in detail view
    in_form BOOLEAN DEFAULT TRUE,               -- Show in create/edit form
    in_search BOOLEAN DEFAULT FALSE,            -- Include in search
    in_filter BOOLEAN DEFAULT FALSE,            -- Show as filter option
    in_sort BOOLEAN DEFAULT FALSE,              -- Allow sorting

    -- UI Settings
    placeholder TEXT,
    help_text TEXT,
    component VARCHAR(50),                      -- Override component
    component_options JSONB DEFAULT '{}',

    -- System
    is_system BOOLEAN DEFAULT FALSE,            -- System field, can't delete
    is_active BOOLEAN DEFAULT TRUE,
    settings JSONB DEFAULT '{}',

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(entity_id, code)
);

-- ----------------------------------------------------------------------------
-- Relations: Σχέσεις μεταξύ entities
-- ----------------------------------------------------------------------------
CREATE TABLE relations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,

    -- Source entity (the one with the foreign key)
    source_entity_id UUID REFERENCES entities(id) ON DELETE CASCADE,
    source_field_code VARCHAR(50) NOT NULL,     -- 'customer_id'

    -- Target entity (the referenced entity)
    target_entity_id UUID REFERENCES entities(id) ON DELETE CASCADE,
    target_field_code VARCHAR(50) DEFAULT 'id', -- Usually 'id'

    -- Relation type
    relation_type VARCHAR(20) NOT NULL,         -- 'belongs_to', 'has_many', 'has_one', 'many_to_many'

    -- Display
    name VARCHAR(100),                          -- 'Customer', 'Orders'

    -- Behavior
    on_delete VARCHAR(20) DEFAULT 'SET NULL',   -- 'CASCADE', 'SET NULL', 'RESTRICT'
    is_required BOOLEAN DEFAULT FALSE,

    -- For many_to_many
    junction_table VARCHAR(100),                -- 'order_products'

    -- Settings
    settings JSONB DEFAULT '{}',

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================================
-- VIEWS: Ορισμός UI Views
-- ============================================================================

-- ----------------------------------------------------------------------------
-- Views: List, Detail, Form, etc.
-- ----------------------------------------------------------------------------
CREATE TABLE views (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    entity_id UUID REFERENCES entities(id) ON DELETE CASCADE,

    code VARCHAR(50) NOT NULL,                  -- 'list', 'detail', 'form', 'kanban'
    name VARCHAR(100) NOT NULL,                 -- 'Customer List'
    view_type VARCHAR(30) NOT NULL,             -- 'list', 'detail', 'form', 'kanban', 'calendar', 'custom'

    -- Settings
    is_default BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    settings JSONB DEFAULT '{}',                -- View-specific settings

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(entity_id, code)
);

-- ----------------------------------------------------------------------------
-- View Sections: Sections in detail/form views
-- ----------------------------------------------------------------------------
CREATE TABLE view_sections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    view_id UUID REFERENCES views(id) ON DELETE CASCADE,

    code VARCHAR(50) NOT NULL,
    name VARCHAR(100) NOT NULL,                 -- 'Basic Info', 'Contact Details'
    description TEXT,

    -- Layout
    display_order INT DEFAULT 0,
    columns INT DEFAULT 2,                      -- 1, 2, 3, 4
    collapsible BOOLEAN DEFAULT FALSE,
    collapsed_default BOOLEAN DEFAULT FALSE,

    -- Settings
    settings JSONB DEFAULT '{}',

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ----------------------------------------------------------------------------
-- View Fields: Which fields appear in each view
-- ----------------------------------------------------------------------------
CREATE TABLE view_fields (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    view_id UUID REFERENCES views(id) ON DELETE CASCADE,
    section_id UUID REFERENCES view_sections(id) ON DELETE SET NULL,
    field_id UUID REFERENCES fields(id) ON DELETE CASCADE,

    display_order INT DEFAULT 0,

    -- Overrides
    label VARCHAR(100),                         -- Override field name
    width VARCHAR(20),                          -- '100px', '25%', 'auto'
    component VARCHAR(50),                      -- Override component
    component_options JSONB DEFAULT '{}',

    -- Visibility
    is_visible BOOLEAN DEFAULT TRUE,
    is_readonly BOOLEAN DEFAULT FALSE,

    -- Settings
    settings JSONB DEFAULT '{}',

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================================
-- PERMISSIONS
-- ============================================================================

-- ----------------------------------------------------------------------------
-- Roles
-- ----------------------------------------------------------------------------
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,

    code VARCHAR(50) NOT NULL,                  -- 'admin', 'manager', 'user'
    name VARCHAR(100) NOT NULL,
    description TEXT,

    is_system BOOLEAN DEFAULT FALSE,            -- Can't delete system roles
    is_active BOOLEAN DEFAULT TRUE,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(tenant_id, code)
);

-- ----------------------------------------------------------------------------
-- Permissions: Per entity, per action
-- ----------------------------------------------------------------------------
CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
    entity_id UUID REFERENCES entities(id) ON DELETE CASCADE,

    -- Actions
    can_view BOOLEAN DEFAULT FALSE,
    can_create BOOLEAN DEFAULT FALSE,
    can_edit BOOLEAN DEFAULT FALSE,
    can_delete BOOLEAN DEFAULT FALSE,
    can_export BOOLEAN DEFAULT FALSE,
    can_import BOOLEAN DEFAULT FALSE,

    -- Field-level permissions (optional)
    field_permissions JSONB DEFAULT '{}',       -- {"salary": {"view": false}}

    -- Row-level permissions (optional)
    row_filter JSONB DEFAULT NULL,              -- {"assigned_to": "$current_user"}

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(role_id, entity_id)
);

-- ----------------------------------------------------------------------------
-- User Roles
-- ----------------------------------------------------------------------------
CREATE TABLE user_roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID REFERENCES roles(id) ON DELETE CASCADE,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(user_id, role_id)
);

-- ============================================================================
-- ACTIONS & WORKFLOWS
-- ============================================================================

-- ----------------------------------------------------------------------------
-- Actions: Custom actions on entities
-- ----------------------------------------------------------------------------
CREATE TABLE actions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    entity_id UUID REFERENCES entities(id) ON DELETE CASCADE,

    code VARCHAR(50) NOT NULL,                  -- 'approve', 'send_email', 'archive'
    name VARCHAR(100) NOT NULL,
    description TEXT,

    -- Type
    action_type VARCHAR(30) NOT NULL,           -- 'button', 'bulk', 'scheduled', 'trigger'

    -- Display
    icon VARCHAR(50),
    color VARCHAR(20),

    -- Conditions
    show_conditions JSONB DEFAULT '{}',         -- When to show the action

    -- Handler
    handler_type VARCHAR(30) NOT NULL,          -- 'workflow', 'webhook', 'code'
    handler_config JSONB DEFAULT '{}',

    -- Settings
    requires_confirmation BOOLEAN DEFAULT FALSE,
    confirmation_message TEXT,

    is_active BOOLEAN DEFAULT TRUE,
    display_order INT DEFAULT 0,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(entity_id, code)
);

-- ----------------------------------------------------------------------------
-- Workflows: Automated processes
-- ----------------------------------------------------------------------------
CREATE TABLE workflows (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    entity_id UUID REFERENCES entities(id) ON DELETE SET NULL,

    code VARCHAR(50) NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,

    -- Trigger
    trigger_type VARCHAR(30) NOT NULL,          -- 'on_create', 'on_update', 'on_delete', 'manual', 'scheduled'
    trigger_conditions JSONB DEFAULT '{}',

    -- Schedule (for scheduled triggers)
    schedule_cron VARCHAR(100),

    is_active BOOLEAN DEFAULT TRUE,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(tenant_id, code)
);

-- ----------------------------------------------------------------------------
-- Workflow Steps
-- ----------------------------------------------------------------------------
CREATE TABLE workflow_steps (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workflow_id UUID REFERENCES workflows(id) ON DELETE CASCADE,

    step_order INT NOT NULL,
    name VARCHAR(100) NOT NULL,

    -- Step type
    step_type VARCHAR(30) NOT NULL,             -- 'condition', 'action', 'delay', 'loop'

    -- Configuration
    config JSONB NOT NULL,

    -- Branching
    on_success_step_id UUID REFERENCES workflow_steps(id),
    on_failure_step_id UUID REFERENCES workflow_steps(id),

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================================
-- UI: Menus & Navigation
-- ============================================================================

-- ----------------------------------------------------------------------------
-- Menus
-- ----------------------------------------------------------------------------
CREATE TABLE menus (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,

    code VARCHAR(50) NOT NULL,                  -- 'main', 'sidebar', 'user'
    name VARCHAR(100) NOT NULL,

    is_active BOOLEAN DEFAULT TRUE,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(tenant_id, code)
);

-- ----------------------------------------------------------------------------
-- Menu Items
-- ----------------------------------------------------------------------------
CREATE TABLE menu_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    menu_id UUID REFERENCES menus(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES menu_items(id) ON DELETE CASCADE,

    -- Link target
    entity_id UUID REFERENCES entities(id) ON DELETE SET NULL,  -- Link to entity list
    module_id UUID REFERENCES modules(id) ON DELETE SET NULL,   -- Link to module
    custom_url VARCHAR(255),                                    -- Or custom URL

    -- Display
    label VARCHAR(100) NOT NULL,
    icon VARCHAR(50),

    display_order INT DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,

    -- Permissions
    required_permission VARCHAR(50),            -- 'view', 'admin', etc.

    settings JSONB DEFAULT '{}',

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================================
-- AUDIT & HISTORY
-- ============================================================================

-- ----------------------------------------------------------------------------
-- Audit Log: Ποιος έκανε τι, πότε
-- ----------------------------------------------------------------------------
CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,

    -- What
    entity_id UUID REFERENCES entities(id) ON DELETE SET NULL,
    entity_code VARCHAR(50),
    record_id UUID,

    -- Action
    action VARCHAR(30) NOT NULL,                -- 'create', 'update', 'delete', 'view'

    -- Data
    old_values JSONB,
    new_values JSONB,
    changed_fields TEXT[],

    -- Context
    ip_address INET,
    user_agent TEXT,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create index for audit queries
CREATE INDEX idx_audit_log_tenant_entity ON audit_log(tenant_id, entity_code);
CREATE INDEX idx_audit_log_record ON audit_log(entity_code, record_id);
CREATE INDEX idx_audit_log_user ON audit_log(user_id);
CREATE INDEX idx_audit_log_created ON audit_log(created_at);

-- ============================================================================
-- INSERT DEFAULT FIELD TYPES
-- ============================================================================

INSERT INTO field_types (code, name, category, db_type, db_default_length, default_component, validation_rules) VALUES
-- Basic Types
('uuid', 'UUID', 'basic', 'UUID', NULL, 'Hidden', '{}'),
('string', 'Text', 'basic', 'VARCHAR', 255, 'TextInput', '{}'),
('text', 'Long Text', 'basic', 'TEXT', NULL, 'TextArea', '{}'),
('richtext', 'Rich Text', 'basic', 'TEXT', NULL, 'RichTextEditor', '{}'),
('integer', 'Integer', 'basic', 'INTEGER', NULL, 'NumberInput', '{}'),
('decimal', 'Decimal', 'basic', 'NUMERIC', NULL, 'NumberInput', '{"precision": 2}'),
('boolean', 'Yes/No', 'basic', 'BOOLEAN', NULL, 'Checkbox', '{}'),
('date', 'Date', 'basic', 'DATE', NULL, 'DatePicker', '{}'),
('datetime', 'Date & Time', 'basic', 'TIMESTAMP', NULL, 'DateTimePicker', '{}'),
('time', 'Time', 'basic', 'TIME', NULL, 'TimePicker', '{}'),

-- Formatted Types
('email', 'Email', 'formatted', 'VARCHAR', 255, 'EmailInput', '{"format": "email"}'),
('phone', 'Phone', 'formatted', 'VARCHAR', 50, 'PhoneInput', '{}'),
('url', 'URL', 'formatted', 'VARCHAR', 500, 'UrlInput', '{"format": "url"}'),
('slug', 'Slug', 'formatted', 'VARCHAR', 255, 'SlugInput', '{"format": "slug"}'),

-- Choice Types
('enum', 'Single Select', 'choice', 'VARCHAR', 50, 'Select', '{}'),
('multi_enum', 'Multi Select', 'choice', 'TEXT[]', NULL, 'MultiSelect', '{}'),
('tags', 'Tags', 'choice', 'TEXT[]', NULL, 'TagInput', '{}'),

-- Special Types
('json', 'JSON', 'special', 'JSONB', NULL, 'JsonEditor', '{}'),
('file', 'File', 'special', 'VARCHAR', 500, 'FileUpload', '{}'),
('image', 'Image', 'special', 'VARCHAR', 500, 'ImageUpload', '{}'),
('color', 'Color', 'special', 'VARCHAR', 20, 'ColorPicker', '{}'),
('icon', 'Icon', 'special', 'VARCHAR', 50, 'IconPicker', '{}'),

-- Relation Types
('belongs_to', 'Belongs To', 'relation', 'UUID', NULL, 'RelationSelect', '{}'),
('has_many', 'Has Many', 'relation', 'VIRTUAL', NULL, 'RelationList', '{}'),
('many_to_many', 'Many to Many', 'relation', 'VIRTUAL', NULL, 'RelationMultiSelect', '{}'),

-- Computed Types
('formula', 'Formula', 'computed', 'VIRTUAL', NULL, 'FormulaDisplay', '{}'),
('aggregation', 'Aggregation', 'computed', 'VIRTUAL', NULL, 'AggregationDisplay', '{}');

-- ============================================================================
-- FUNCTIONS & TRIGGERS
-- ============================================================================

-- Auto-update updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply to all tables with updated_at
CREATE TRIGGER update_tenants_updated_at BEFORE UPDATE ON tenants FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_modules_updated_at BEFORE UPDATE ON modules FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_entities_updated_at BEFORE UPDATE ON entities FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_fields_updated_at BEFORE UPDATE ON fields FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_views_updated_at BEFORE UPDATE ON views FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_roles_updated_at BEFORE UPDATE ON roles FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_permissions_updated_at BEFORE UPDATE ON permissions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_workflows_updated_at BEFORE UPDATE ON workflows FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_menus_updated_at BEFORE UPDATE ON menus FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- INDEXES
-- ============================================================================

CREATE INDEX idx_users_tenant ON users(tenant_id);
CREATE INDEX idx_users_email ON users(tenant_id, email);
CREATE INDEX idx_modules_tenant ON modules(tenant_id);
CREATE INDEX idx_entities_tenant ON entities(tenant_id);
CREATE INDEX idx_entities_module ON entities(module_id);
CREATE INDEX idx_fields_entity ON fields(entity_id);
CREATE INDEX idx_relations_source ON relations(source_entity_id);
CREATE INDEX idx_relations_target ON relations(target_entity_id);
CREATE INDEX idx_views_entity ON views(entity_id);
CREATE INDEX idx_permissions_role ON permissions(role_id);
CREATE INDEX idx_permissions_entity ON permissions(entity_id);
CREATE INDEX idx_menu_items_menu ON menu_items(menu_id);

-- ============================================================================
-- UI COMPONENTS: React components stored in database
-- ============================================================================

-- ----------------------------------------------------------------------------
-- UI Components: Reusable React components
-- ----------------------------------------------------------------------------
CREATE TABLE ui_components (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,  -- NULL = system component

    code VARCHAR(100) UNIQUE NOT NULL,          -- 'DataGrid', 'Form', 'Button'
    name VARCHAR(100) NOT NULL,
    description TEXT,
    category VARCHAR(50),                       -- 'layout', 'input', 'display', 'data'

    -- Component code (JSX/React)
    code_jsx TEXT NOT NULL,                     -- The actual React component code

    -- Props definition
    props_schema JSONB DEFAULT '{}',            -- JSON Schema for props

    -- Dependencies
    dependencies TEXT[] DEFAULT '{}',           -- ['react', 'lodash']

    -- Settings
    is_system BOOLEAN DEFAULT FALSE,            -- System component, can't edit
    is_active BOOLEAN DEFAULT TRUE,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ----------------------------------------------------------------------------
-- UI Layouts: Page layouts (header, sidebar, footer, content areas)
-- ----------------------------------------------------------------------------
CREATE TABLE ui_layouts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,

    code VARCHAR(100) NOT NULL,                 -- 'default', 'admin', 'minimal'
    name VARCHAR(100) NOT NULL,
    description TEXT,

    -- Layout structure (JSX with slots)
    code_jsx TEXT NOT NULL,                     -- Layout with {children}, {sidebar}, etc.

    -- Slots definition
    slots JSONB DEFAULT '["content"]',          -- Available slots

    -- Settings
    is_default BOOLEAN DEFAULT FALSE,
    is_system BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(tenant_id, code)
);

-- ----------------------------------------------------------------------------
-- UI Pages: Dynamic pages composed of components
-- ----------------------------------------------------------------------------
CREATE TABLE ui_pages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,

    code VARCHAR(100) NOT NULL,                 -- 'dashboard', 'customers', 'settings'
    name VARCHAR(100) NOT NULL,
    description TEXT,

    -- Route
    route VARCHAR(255) NOT NULL,                -- '/dashboard', '/m/:module/:entity'

    -- Layout
    layout_id UUID REFERENCES ui_layouts(id) ON DELETE SET NULL,

    -- Page type
    page_type VARCHAR(50) NOT NULL,             -- 'static', 'entity_list', 'entity_detail', 'entity_form', 'custom'

    -- For entity pages
    entity_id UUID REFERENCES entities(id) ON DELETE CASCADE,

    -- Page content (JSX or component composition)
    code_jsx TEXT,                              -- Custom page code
    components JSONB DEFAULT '[]',              -- Component instances with props

    -- Settings
    title VARCHAR(255),                         -- Page title
    icon VARCHAR(50),
    requires_auth BOOLEAN DEFAULT TRUE,
    required_permission VARCHAR(50),

    is_system BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,

    display_order INT DEFAULT 0,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(tenant_id, route)
);

-- ----------------------------------------------------------------------------
-- UI Themes: Visual themes (colors, fonts, etc.)
-- ----------------------------------------------------------------------------
CREATE TABLE ui_themes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,

    code VARCHAR(50) NOT NULL,                  -- 'dark', 'light', 'custom'
    name VARCHAR(100) NOT NULL,

    -- Theme variables
    variables JSONB NOT NULL DEFAULT '{
        "colors": {
            "primary": "#00d4ff",
            "secondary": "#7b2cbf",
            "background": "#0f0f1a",
            "surface": "#1a1a2e",
            "text": "#ffffff",
            "textMuted": "rgba(255,255,255,0.5)",
            "border": "rgba(255,255,255,0.1)",
            "success": "#10b981",
            "warning": "#f59e0b",
            "error": "#ef4444"
        },
        "fonts": {
            "body": "-apple-system, BlinkMacSystemFont, Segoe UI, Roboto, sans-serif",
            "mono": "Consolas, Monaco, monospace"
        },
        "spacing": {
            "xs": "4px",
            "sm": "8px",
            "md": "16px",
            "lg": "24px",
            "xl": "32px"
        },
        "borderRadius": {
            "sm": "4px",
            "md": "8px",
            "lg": "16px"
        }
    }',

    -- Custom CSS overrides
    custom_css TEXT,

    is_default BOOLEAN DEFAULT FALSE,
    is_system BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(tenant_id, code)
);

-- Indexes for UI tables
CREATE INDEX idx_ui_components_tenant ON ui_components(tenant_id);
CREATE INDEX idx_ui_components_category ON ui_components(category);
CREATE INDEX idx_ui_layouts_tenant ON ui_layouts(tenant_id);
CREATE INDEX idx_ui_pages_tenant ON ui_pages(tenant_id);
CREATE INDEX idx_ui_pages_route ON ui_pages(tenant_id, route);
CREATE INDEX idx_ui_pages_entity ON ui_pages(entity_id);
CREATE INDEX idx_ui_themes_tenant ON ui_themes(tenant_id);

-- Triggers for UI tables
CREATE TRIGGER update_ui_components_updated_at BEFORE UPDATE ON ui_components FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_ui_layouts_updated_at BEFORE UPDATE ON ui_layouts FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_ui_pages_updated_at BEFORE UPDATE ON ui_pages FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_ui_themes_updated_at BEFORE UPDATE ON ui_themes FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON TABLE ui_components IS 'Reusable React components stored in database';
COMMENT ON TABLE ui_layouts IS 'Page layouts with slots for content';
COMMENT ON TABLE ui_pages IS 'Dynamic pages composed of components';
COMMENT ON TABLE ui_themes IS 'Visual themes (colors, fonts, spacing)';
COMMENT ON TABLE tenants IS 'Κάθε πελάτης/εταιρεία - multi-tenant isolation';

-- ============================================================================
-- DEFAULT UI COMPONENTS
-- ============================================================================

-- Default Layout
INSERT INTO ui_layouts (code, name, description, code_jsx, slots, is_system, is_default) VALUES
('admin', 'Admin Layout', 'Main admin layout with sidebar', '
<div class="app">
  <aside class="sidebar">
    <div class="sidebar-header">
      <div class="logo">Genesis</div>
      <div class="tenant-name">{tenant.name}</div>
    </div>
    <nav class="nav">
      {navigation}
    </nav>
    <div class="sidebar-footer">
      {user_menu}
    </div>
  </aside>
  <main class="main">
    <header class="header">
      <h1 class="page-title">{page.title}</h1>
      <div class="header-actions">{actions}</div>
    </header>
    <div class="content">
      {content}
    </div>
  </main>
</div>
', '["navigation", "user_menu", "actions", "content"]', true, true);

-- Default Theme
INSERT INTO ui_themes (code, name, variables, is_system, is_default) VALUES
('genesis-dark', 'Genesis Dark', '{
  "colors": {
    "primary": "#00d4ff",
    "secondary": "#7b2cbf",
    "background": "#0f0f1a",
    "surface": "#1a1a2e",
    "surfaceHover": "rgba(255,255,255,0.05)",
    "text": "#ffffff",
    "textMuted": "rgba(255,255,255,0.5)",
    "border": "rgba(255,255,255,0.1)",
    "success": "#10b981",
    "warning": "#f59e0b",
    "error": "#ef4444",
    "info": "#3b82f6"
  },
  "fonts": {
    "body": "-apple-system, BlinkMacSystemFont, Segoe UI, Roboto, sans-serif",
    "heading": "-apple-system, BlinkMacSystemFont, Segoe UI, Roboto, sans-serif",
    "mono": "Consolas, Monaco, monospace"
  },
  "fontSizes": {
    "xs": "11px",
    "sm": "12px",
    "md": "14px",
    "lg": "16px",
    "xl": "20px",
    "xxl": "24px"
  },
  "spacing": {
    "xs": "4px",
    "sm": "8px",
    "md": "16px",
    "lg": "24px",
    "xl": "32px"
  },
  "borderRadius": {
    "sm": "4px",
    "md": "8px",
    "lg": "12px",
    "xl": "16px",
    "full": "9999px"
  },
  "shadows": {
    "sm": "0 1px 2px rgba(0,0,0,0.3)",
    "md": "0 4px 12px rgba(0,0,0,0.4)",
    "lg": "0 8px 24px rgba(0,0,0,0.5)"
  }
}', true, true);

-- Core UI Components
INSERT INTO ui_components (code, name, category, code_jsx, props_schema, is_system) VALUES

-- Button Component
('Button', 'Button', 'input', '
function Button({ children, variant = "primary", size = "md", icon, loading, disabled, onClick }) {
  const baseClass = "btn";
  const variantClass = `btn-${variant}`;
  const sizeClass = `btn-${size}`;

  return (
    <button
      className={`${baseClass} ${variantClass} ${sizeClass}`}
      onClick={onClick}
      disabled={disabled || loading}
    >
      {loading && <span className="spinner" />}
      {icon && <span className="btn-icon">{icon}</span>}
      {children}
    </button>
  );
}
', '{
  "type": "object",
  "properties": {
    "variant": {"type": "string", "enum": ["primary", "secondary", "danger", "ghost"]},
    "size": {"type": "string", "enum": ["sm", "md", "lg"]},
    "icon": {"type": "string"},
    "loading": {"type": "boolean"},
    "disabled": {"type": "boolean"}
  }
}', true),

-- TextInput Component
('TextInput', 'Text Input', 'input', '
function TextInput({ label, value, onChange, placeholder, error, required, disabled, type = "text" }) {
  return (
    <div className="form-group">
      {label && <label className="form-label">{label} {required && <span className="required">*</span>}</label>}
      <input
        type={type}
        className={`form-input ${error ? "error" : ""}`}
        value={value || ""}
        onChange={e => onChange(e.target.value)}
        placeholder={placeholder}
        disabled={disabled}
      />
      {error && <span className="form-error">{error}</span>}
    </div>
  );
}
', '{
  "type": "object",
  "properties": {
    "label": {"type": "string"},
    "placeholder": {"type": "string"},
    "required": {"type": "boolean"},
    "disabled": {"type": "boolean"},
    "type": {"type": "string", "enum": ["text", "email", "password", "number", "tel", "url"]}
  }
}', true),

-- Select Component
('Select', 'Select', 'input', '
function Select({ label, value, onChange, options = [], placeholder, error, required, disabled }) {
  return (
    <div className="form-group">
      {label && <label className="form-label">{label} {required && <span className="required">*</span>}</label>}
      <select
        className={`form-input ${error ? "error" : ""}`}
        value={value || ""}
        onChange={e => onChange(e.target.value)}
        disabled={disabled}
      >
        {placeholder && <option value="">{placeholder}</option>}
        {options.map(opt => (
          <option key={opt.value} value={opt.value}>{opt.label}</option>
        ))}
      </select>
      {error && <span className="form-error">{error}</span>}
    </div>
  );
}
', '{
  "type": "object",
  "properties": {
    "label": {"type": "string"},
    "options": {"type": "array", "items": {"type": "object", "properties": {"value": {"type": "string"}, "label": {"type": "string"}}}},
    "placeholder": {"type": "string"},
    "required": {"type": "boolean"}
  }
}', true),

-- DataGrid Component
('DataGrid', 'Data Grid', 'data', '
function DataGrid({ entity, columns, data = [], loading, onRowClick, onEdit, onDelete, onCreate }) {
  if (loading) {
    return <div className="loading"><div className="spinner" /></div>;
  }

  if (data.length === 0) {
    return (
      <div className="empty-state">
        <div className="empty-icon">📋</div>
        <div className="empty-title">No {entity.name_plural || entity.name + "s"} yet</div>
        <div className="empty-text">Create your first record to get started</div>
        {onCreate && <button className="btn btn-primary" onClick={onCreate}>+ Create {entity.name}</button>}
      </div>
    );
  }

  return (
    <div className="table-container">
      <table>
        <thead>
          <tr>
            {columns.map(col => (
              <th key={col.code}>{col.name}</th>
            ))}
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {data.map(row => (
            <tr key={row.id} onClick={() => onRowClick && onRowClick(row)}>
              {columns.map(col => (
                <td key={col.code}>{formatValue(row[col.code], col)}</td>
              ))}
              <td className="actions">
                {onEdit && <button className="btn btn-sm btn-secondary" onClick={e => { e.stopPropagation(); onEdit(row); }}>Edit</button>}
                {onDelete && <button className="btn btn-sm btn-danger" onClick={e => { e.stopPropagation(); onDelete(row); }}>Delete</button>}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function formatValue(value, field) {
  if (value === null || value === undefined) return "-";
  if (field.field_type === "boolean") return value ? "✓" : "✗";
  if (field.field_type === "date") return new Date(value).toLocaleDateString();
  if (field.field_type === "datetime") return new Date(value).toLocaleString();
  return String(value);
}
', '{
  "type": "object",
  "properties": {
    "entity": {"type": "object"},
    "columns": {"type": "array"},
    "data": {"type": "array"}
  }
}', true),

-- DynamicForm Component
('DynamicForm', 'Dynamic Form', 'data', '
function DynamicForm({ entity, fields, values = {}, onChange, onSubmit, onCancel, loading, mode = "create" }) {
  const handleFieldChange = (code, value) => {
    onChange({ ...values, [code]: value });
  };

  const renderField = (field) => {
    const value = values[field.code];
    const commonProps = {
      key: field.id,
      label: field.name,
      value: value,
      onChange: (val) => handleFieldChange(field.code, val),
      required: field.is_required,
      placeholder: field.placeholder,
      disabled: loading
    };

    switch (field.field_type_code) {
      case "string":
      case "email":
      case "url":
      case "phone":
        return <TextInput {...commonProps} type={field.field_type_code === "email" ? "email" : "text"} />;
      case "text":
      case "richtext":
        return <TextArea {...commonProps} />;
      case "integer":
      case "decimal":
        return <TextInput {...commonProps} type="number" />;
      case "boolean":
        return <Checkbox {...commonProps} />;
      case "date":
        return <TextInput {...commonProps} type="date" />;
      case "datetime":
        return <TextInput {...commonProps} type="datetime-local" />;
      case "enum":
        return <Select {...commonProps} options={field.options || []} />;
      default:
        return <TextInput {...commonProps} />;
    }
  };

  return (
    <form onSubmit={(e) => { e.preventDefault(); onSubmit(values); }} className="dynamic-form">
      <div className="form-fields">
        {fields.filter(f => f.in_form !== false).map(renderField)}
      </div>
      <div className="form-actions">
        {onCancel && <button type="button" className="btn btn-secondary" onClick={onCancel} disabled={loading}>Cancel</button>}
        <button type="submit" className="btn btn-primary" disabled={loading}>
          {loading ? "Saving..." : (mode === "create" ? "Create" : "Save")}
        </button>
      </div>
    </form>
  );
}

function TextArea({ label, value, onChange, placeholder, required, disabled }) {
  return (
    <div className="form-group">
      {label && <label className="form-label">{label} {required && <span className="required">*</span>}</label>}
      <textarea
        className="form-input"
        value={value || ""}
        onChange={e => onChange(e.target.value)}
        placeholder={placeholder}
        disabled={disabled}
        rows={4}
      />
    </div>
  );
}

function Checkbox({ label, value, onChange, disabled }) {
  return (
    <div className="form-group form-checkbox">
      <label className="checkbox-label">
        <input
          type="checkbox"
          checked={!!value}
          onChange={e => onChange(e.target.checked)}
          disabled={disabled}
        />
        {label}
      </label>
    </div>
  );
}
', '{
  "type": "object",
  "properties": {
    "entity": {"type": "object"},
    "fields": {"type": "array"},
    "values": {"type": "object"},
    "mode": {"type": "string", "enum": ["create", "edit"]}
  }
}', true),

-- Card Component
('Card', 'Card', 'layout', '
function Card({ title, children, actions, collapsible, defaultCollapsed }) {
  const [collapsed, setCollapsed] = React.useState(defaultCollapsed);

  return (
    <div className={`card ${collapsed ? "collapsed" : ""}`}>
      {title && (
        <div className="card-header">
          <h3 className="card-title">
            {collapsible && (
              <button className="collapse-btn" onClick={() => setCollapsed(!collapsed)}>
                {collapsed ? "▶" : "▼"}
              </button>
            )}
            {title}
          </h3>
          {actions && <div className="card-actions">{actions}</div>}
        </div>
      )}
      {!collapsed && <div className="card-body">{children}</div>}
    </div>
  );
}
', '{
  "type": "object",
  "properties": {
    "title": {"type": "string"},
    "collapsible": {"type": "boolean"},
    "defaultCollapsed": {"type": "boolean"}
  }
}', true),

-- StatCard Component
('StatCard', 'Stat Card', 'display', '
function StatCard({ label, value, icon, trend, trendDirection }) {
  return (
    <div className="stat-card">
      {icon && <div className="stat-icon">{icon}</div>}
      <div className="stat-content">
        <div className="stat-value">{value}</div>
        <div className="stat-label">{label}</div>
        {trend && (
          <div className={`stat-trend ${trendDirection}`}>
            {trendDirection === "up" ? "↑" : "↓"} {trend}
          </div>
        )}
      </div>
    </div>
  );
}
', '{
  "type": "object",
  "properties": {
    "label": {"type": "string"},
    "value": {"type": ["string", "number"]},
    "icon": {"type": "string"},
    "trend": {"type": "string"},
    "trendDirection": {"type": "string", "enum": ["up", "down"]}
  }
}', true),

-- Modal Component
('Modal', 'Modal', 'layout', '
function Modal({ isOpen, onClose, title, children, footer, size = "md" }) {
  if (!isOpen) return null;

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className={`modal modal-${size}`} onClick={e => e.stopPropagation()}>
        <div className="modal-header">
          <h2 className="modal-title">{title}</h2>
          <button className="modal-close" onClick={onClose}>&times;</button>
        </div>
        <div className="modal-body">{children}</div>
        {footer && <div className="modal-footer">{footer}</div>}
      </div>
    </div>
  );
}
', '{
  "type": "object",
  "properties": {
    "title": {"type": "string"},
    "size": {"type": "string", "enum": ["sm", "md", "lg", "xl"]}
  }
}', true),

-- Navigation Component
('Navigation', 'Navigation', 'layout', '
function Navigation({ modules, entities, currentPath }) {
  const moduleMap = {};
  modules.forEach(m => moduleMap[m.id] = { ...m, entities: [] });
  entities.forEach(e => {
    if (e.module_id && moduleMap[e.module_id]) {
      moduleMap[e.module_id].entities.push(e);
    }
  });

  return (
    <nav className="nav">
      <div className="nav-section">
        <div className="nav-section-title">Overview</div>
        <a href="/app" className={`nav-item ${currentPath === "/app" ? "active" : ""}`}>
          <span className="nav-icon">📊</span>
          Dashboard
        </a>
      </div>
      {Object.values(moduleMap).map(mod => (
        <div key={mod.id} className="nav-section">
          <div className="nav-section-title">{mod.icon} {mod.name}</div>
          {mod.entities.map(ent => (
            <a
              key={ent.id}
              href={`/app/${mod.code}/${ent.code}`}
              className={`nav-item ${currentPath === `/app/${mod.code}/${ent.code}` ? "active" : ""}`}
            >
              <span className="nav-icon">{ent.icon || "📄"}</span>
              {ent.name_plural || ent.name}
            </a>
          ))}
        </div>
      ))}
    </nav>
  );
}
', '{
  "type": "object",
  "properties": {
    "modules": {"type": "array"},
    "entities": {"type": "array"},
    "currentPath": {"type": "string"}
  }
}', true);
COMMENT ON TABLE entities IS 'Meta-table: Ορίζει δυναμικά entities (tables)';
COMMENT ON TABLE fields IS 'Meta-table: Ορίζει τα πεδία κάθε entity';
COMMENT ON TABLE field_types IS 'Διαθέσιμοι τύποι πεδίων';
COMMENT ON TABLE relations IS 'Σχέσεις μεταξύ entities';
COMMENT ON TABLE views IS 'Ορισμός UI views (list, detail, form)';
COMMENT ON TABLE permissions IS 'RBAC permissions per entity';
COMMENT ON TABLE workflows IS 'Automated workflows/processes';
COMMENT ON TABLE audit_log IS 'Full audit trail of all actions';
