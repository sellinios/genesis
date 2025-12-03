// Package engine contains the core Genesis dynamic engine
// This is the heart of Genesis - it reads meta-schema and generates everything dynamically
package engine

import (
	"fmt"
	"strings"

	"github.com/aethra/genesis/internal/models"
	"github.com/aethra/genesis/internal/security"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SchemaEngine handles all schema-related operations
type SchemaEngine struct {
	db *gorm.DB
}

// NewSchemaEngine creates a new schema engine
func NewSchemaEngine(db *gorm.DB) *SchemaEngine {
	return &SchemaEngine{db: db}
}

// =============================================================================
// SCHEMA RETRIEVAL
// =============================================================================

// GetFullSchema returns the complete schema for a tenant
// This is what the frontend uses to render everything dynamically
func (e *SchemaEngine) GetFullSchema(tenantID uuid.UUID) (*TenantSchema, error) {
	schema := &TenantSchema{
		TenantID: tenantID,
	}

	// Get tenant info
	var tenant models.Tenant
	if err := e.db.First(&tenant, "id = ?", tenantID).Error; err != nil {
		return nil, fmt.Errorf("tenant not found: %w", err)
	}
	schema.Tenant = &tenant

	// Get all modules
	var modules []models.Module
	if err := e.db.Where("tenant_id = ? AND is_active = true", tenantID).
		Order("display_order").Find(&modules).Error; err != nil {
		return nil, fmt.Errorf("failed to get modules: %w", err)
	}
	schema.Modules = modules

	// Get all entities with their fields
	var entities []models.Entity
	if err := e.db.Where("tenant_id = ? AND is_active = true", tenantID).
		Preload("Fields", func(db *gorm.DB) *gorm.DB {
			return db.Where("is_active = true").Order("display_order")
		}).
		Preload("Fields.FieldType").
		Order("display_order").Find(&entities).Error; err != nil {
		return nil, fmt.Errorf("failed to get entities: %w", err)
	}
	schema.Entities = entities

	// Get all relations
	var relations []models.Relation
	if err := e.db.Where("tenant_id = ?", tenantID).Find(&relations).Error; err != nil {
		return nil, fmt.Errorf("failed to get relations: %w", err)
	}
	schema.Relations = relations

	// Get all views
	var views []models.View
	if err := e.db.Where("tenant_id = ? AND is_active = true", tenantID).
		Preload("Sections", func(db *gorm.DB) *gorm.DB {
			return db.Order("display_order")
		}).
		Preload("Fields", func(db *gorm.DB) *gorm.DB {
			return db.Order("display_order")
		}).
		Find(&views).Error; err != nil {
		return nil, fmt.Errorf("failed to get views: %w", err)
	}
	schema.Views = views

	// Get all menus
	var menus []models.Menu
	if err := e.db.Where("tenant_id = ? AND is_active = true", tenantID).
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Where("is_active = true").Order("display_order")
		}).
		Find(&menus).Error; err != nil {
		return nil, fmt.Errorf("failed to get menus: %w", err)
	}
	schema.Menus = menus

	// Get field types (global)
	var fieldTypes []models.FieldType
	if err := e.db.Find(&fieldTypes).Error; err != nil {
		return nil, fmt.Errorf("failed to get field types: %w", err)
	}
	schema.FieldTypes = fieldTypes

	return schema, nil
}

// GetEntitySchema returns the schema for a specific entity
func (e *SchemaEngine) GetEntitySchema(tenantID uuid.UUID, entityCode string) (*EntitySchema, error) {
	var entity models.Entity
	if err := e.db.Where("tenant_id = ? AND code = ? AND is_active = true", tenantID, entityCode).
		Preload("Fields", func(db *gorm.DB) *gorm.DB {
			return db.Where("is_active = true").Order("display_order")
		}).
		Preload("Fields.FieldType").
		First(&entity).Error; err != nil {
		return nil, fmt.Errorf("entity not found: %w", err)
	}

	// Get relations for this entity
	var relations []models.Relation
	if err := e.db.Where("tenant_id = ? AND (source_entity_id = ? OR target_entity_id = ?)",
		tenantID, entity.ID, entity.ID).
		Preload("SourceEntity").
		Preload("TargetEntity").
		Find(&relations).Error; err != nil {
		return nil, fmt.Errorf("failed to get relations: %w", err)
	}

	// Get views for this entity
	var views []models.View
	if err := e.db.Where("tenant_id = ? AND entity_id = ? AND is_active = true", tenantID, entity.ID).
		Preload("Sections", func(db *gorm.DB) *gorm.DB {
			return db.Order("display_order")
		}).
		Preload("Fields", func(db *gorm.DB) *gorm.DB {
			return db.Order("display_order")
		}).
		Find(&views).Error; err != nil {
		return nil, fmt.Errorf("failed to get views: %w", err)
	}

	// Get actions for this entity
	var actions []models.Action
	if err := e.db.Where("tenant_id = ? AND entity_id = ? AND is_active = true", tenantID, entity.ID).
		Order("display_order").Find(&actions).Error; err != nil {
		return nil, fmt.Errorf("failed to get actions: %w", err)
	}

	return &EntitySchema{
		Entity:    &entity,
		Relations: relations,
		Views:     views,
		Actions:   actions,
	}, nil
}

// =============================================================================
// DYNAMIC TABLE MANAGEMENT
// =============================================================================

// CreateEntityTable creates the actual database table for an entity
func (e *SchemaEngine) CreateEntityTable(entity *models.Entity, fields []models.Field) error {
	tableName, err := e.safeTableName(entity)
	if err != nil {
		return err
	}

	// Build CREATE TABLE statement
	var columns []string

	for _, field := range fields {
		colDef, err := e.buildColumnDefinition(&field)
		if err != nil {
			return fmt.Errorf("failed to build column definition for %s: %w", field.Code, err)
		}
		if colDef != "" {
			columns = append(columns, colDef)
		}
	}

	// Add system columns if enabled
	if entity.UseTimestamps {
		columns = append(columns, "created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP")
		columns = append(columns, "updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP")
	}

	if entity.UseSoftDelete {
		columns = append(columns, "deleted_at TIMESTAMP")
	}

	// Always add tenant_id for multi-tenancy
	columns = append(columns, "tenant_id UUID NOT NULL REFERENCES tenants(id)")

	sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n  %s\n)", tableName, strings.Join(columns, ",\n  "))

	// Use transaction for atomicity
	return e.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(sql).Error; err != nil {
			return fmt.Errorf("failed to create table %s: %w", tableName, err)
		}

		// Create indexes
		if err := e.createTableIndexes(tx, tableName, fields); err != nil {
			return fmt.Errorf("failed to create indexes: %w", err)
		}

		// Create updated_at trigger if timestamps enabled
		if entity.UseTimestamps {
			if err := e.createUpdatedAtTrigger(tx, tableName); err != nil {
				return fmt.Errorf("failed to create trigger: %w", err)
			}
		}

		return nil
	})
}

// AddField adds a new column to an existing entity table
func (e *SchemaEngine) AddField(entity *models.Entity, field *models.Field) error {
	tableName, err := e.safeTableName(entity)
	if err != nil {
		return err
	}

	colDef, err := e.buildColumnDefinition(field)
	if err != nil {
		return err
	}

	if colDef == "" {
		return nil // Computed field, no column needed
	}

	sql := fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS %s", tableName, colDef)

	if err := e.db.Exec(sql).Error; err != nil {
		return fmt.Errorf("failed to add column: %w", err)
	}

	return nil
}

// RemoveField removes a column from an entity table
func (e *SchemaEngine) RemoveField(entity *models.Entity, field *models.Field) error {
	tableName, err := e.safeTableName(entity)
	if err != nil {
		return err
	}

	columnName, err := e.safeColumnName(field)
	if err != nil {
		return err
	}

	sql := fmt.Sprintf("ALTER TABLE %s DROP COLUMN IF EXISTS %s", tableName, columnName)

	if err := e.db.Exec(sql).Error; err != nil {
		return fmt.Errorf("failed to remove column: %w", err)
	}

	return nil
}

// =============================================================================
// HELPER METHODS
// =============================================================================

func (e *SchemaEngine) safeTableName(entity *models.Entity) (string, error) {
	tableName := entity.TableName
	if tableName == "" {
		tableName = fmt.Sprintf("data_%s", entity.Code)
	}
	if err := security.ValidateIdentifier(tableName); err != nil {
		return "", fmt.Errorf("invalid table name '%s': %w", tableName, err)
	}
	return security.QuoteIdentifier(tableName), nil
}

func (e *SchemaEngine) safeColumnName(field *models.Field) (string, error) {
	columnName := field.ColumnName
	if columnName == "" {
		columnName = field.Code
	}
	if err := security.ValidateIdentifier(columnName); err != nil {
		return "", fmt.Errorf("invalid column name '%s': %w", columnName, err)
	}
	return security.QuoteIdentifier(columnName), nil
}

func (e *SchemaEngine) getTableName(entity *models.Entity) string {
	if entity.TableName != "" {
		return entity.TableName
	}
	return fmt.Sprintf("data_%s", entity.Code)
}

func (e *SchemaEngine) getColumnName(field *models.Field) string {
	if field.ColumnName != "" {
		return field.ColumnName
	}
	return field.Code
}

func (e *SchemaEngine) buildColumnDefinition(field *models.Field) (string, error) {
	columnName := e.getColumnName(field)

	// Validate column name
	if err := security.ValidateIdentifier(columnName); err != nil {
		return "", fmt.Errorf("invalid column name: %w", err)
	}

	quotedColumnName := security.QuoteIdentifier(columnName)

	// Get column type
	var columnType string
	if field.ColumnType != "" {
		columnType = field.ColumnType
	} else if field.FieldType != nil {
		columnType = e.mapFieldTypeToSQL(field.FieldType, field)
	} else {
		columnType = "VARCHAR(255)"
	}

	// Skip computed fields (no DB column)
	if columnType == "" {
		return "", nil
	}

	def := fmt.Sprintf("%s %s", quotedColumnName, columnType)

	// Add constraints
	if field.IsPrimary {
		def += " PRIMARY KEY"
	}

	if field.IsAuto && field.FieldType != nil && field.FieldType.Code == "uuid" {
		def += " DEFAULT uuid_generate_v4()"
	}

	if field.IsRequired && !field.IsPrimary {
		def += " NOT NULL"
	}

	if field.IsUnique {
		def += " UNIQUE"
	}

	if field.DefaultValue != nil && *field.DefaultValue != "" {
		// Validate default value to prevent SQL injection
		defaultVal := *field.DefaultValue
		if isSimpleLiteral(defaultVal) {
			def += fmt.Sprintf(" DEFAULT %s", defaultVal)
		}
	}

	return def, nil
}

// isSimpleLiteral checks if a value is a safe SQL literal
func isSimpleLiteral(val string) bool {
	// Allow: NULL, TRUE, FALSE, CURRENT_TIMESTAMP, numbers, single-quoted strings
	switch strings.ToUpper(val) {
	case "NULL", "TRUE", "FALSE", "CURRENT_TIMESTAMP", "CURRENT_DATE", "CURRENT_TIME":
		return true
	}
	// Check if it's a number
	if _, err := fmt.Sscanf(val, "%f", new(float64)); err == nil {
		return true
	}
	// Check if it's a single-quoted string (very basic check)
	if len(val) >= 2 && val[0] == '\'' && val[len(val)-1] == '\'' {
		// Ensure no unescaped quotes inside
		inner := val[1 : len(val)-1]
		if !strings.Contains(inner, "'") || strings.Contains(inner, "''") {
			return true
		}
	}
	return false
}

func (e *SchemaEngine) mapFieldTypeToSQL(ft *models.FieldType, field *models.Field) string {
	switch ft.Code {
	case "uuid":
		return "UUID"
	case "string":
		length := 255
		if field.MaxLength != nil {
			length = *field.MaxLength
		} else if ft.DBDefaultLength != nil {
			length = *ft.DBDefaultLength
		}
		return fmt.Sprintf("VARCHAR(%d)", length)
	case "text", "richtext":
		return "TEXT"
	case "integer":
		return "INTEGER"
	case "decimal":
		return "NUMERIC(15,2)"
	case "boolean":
		return "BOOLEAN"
	case "date":
		return "DATE"
	case "datetime":
		return "TIMESTAMP"
	case "time":
		return "TIME"
	case "email", "phone", "url", "slug":
		return "VARCHAR(255)"
	case "enum":
		return "VARCHAR(50)"
	case "multi_enum", "tags":
		return "TEXT[]"
	case "json":
		return "JSONB"
	case "file", "image":
		return "VARCHAR(500)"
	case "color":
		return "VARCHAR(20)"
	case "icon":
		return "VARCHAR(50)"
	case "belongs_to":
		return "UUID"
	case "has_many", "many_to_many", "formula", "aggregation":
		return "" // No column needed
	default:
		return "VARCHAR(255)"
	}
}

func (e *SchemaEngine) createTableIndexes(tx *gorm.DB, tableName string, fields []models.Field) error {
	for _, field := range fields {
		columnName := e.getColumnName(&field)

		// Validate column name
		if err := security.ValidateIdentifier(columnName); err != nil {
			continue // Skip invalid column names
		}

		quotedColumnName := security.QuoteIdentifier(columnName)

		// Extract unquoted table name for index naming
		unquotedTableName := strings.Trim(tableName, `"`)

		// Create index for searchable fields
		if field.InSearch || field.InFilter || field.InSort {
			indexName := fmt.Sprintf("idx_%s_%s", unquotedTableName, columnName)
			if err := security.ValidateIdentifier(indexName); err != nil {
				continue
			}
			quotedIndexName := security.QuoteIdentifier(indexName)
			sql := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s(%s)", quotedIndexName, tableName, quotedColumnName)
			if err := tx.Exec(sql).Error; err != nil {
				return err
			}
		}

		// Create unique index if needed
		if field.IsUnique && !field.IsPrimary {
			indexName := fmt.Sprintf("idx_%s_%s_unique", unquotedTableName, columnName)
			if err := security.ValidateIdentifier(indexName); err != nil {
				continue
			}
			quotedIndexName := security.QuoteIdentifier(indexName)
			sql := fmt.Sprintf("CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s(%s)", quotedIndexName, tableName, quotedColumnName)
			if err := tx.Exec(sql).Error; err != nil {
				return err
			}
		}
	}

	// Always create tenant_id index
	unquotedTableName := strings.Trim(tableName, `"`)
	tenantIndexName := fmt.Sprintf("idx_%s_tenant_id", unquotedTableName)
	if err := security.ValidateIdentifier(tenantIndexName); err == nil {
		quotedTenantIndexName := security.QuoteIdentifier(tenantIndexName)
		sql := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s(tenant_id)", quotedTenantIndexName, tableName)
		return tx.Exec(sql).Error
	}
	return nil
}

func (e *SchemaEngine) createUpdatedAtTrigger(tx *gorm.DB, tableName string) error {
	unquotedTableName := strings.Trim(tableName, `"`)
	triggerName := fmt.Sprintf("update_%s_updated_at", unquotedTableName)

	// Validate trigger name
	if err := security.ValidateIdentifier(triggerName); err != nil {
		return fmt.Errorf("invalid trigger name: %w", err)
	}

	quotedTriggerName := security.QuoteIdentifier(triggerName)
	sql := fmt.Sprintf(`
		CREATE OR REPLACE TRIGGER %s
		BEFORE UPDATE ON %s
		FOR EACH ROW
		EXECUTE FUNCTION update_updated_at_column()
	`, quotedTriggerName, tableName)
	return tx.Exec(sql).Error
}

// =============================================================================
// SCHEMA TYPES
// =============================================================================

// TenantSchema represents the complete schema for a tenant
type TenantSchema struct {
	TenantID   uuid.UUID          `json:"tenant_id"`
	Tenant     *models.Tenant     `json:"tenant"`
	Modules    []models.Module    `json:"modules"`
	Entities   []models.Entity    `json:"entities"`
	Relations  []models.Relation  `json:"relations"`
	Views      []models.View      `json:"views"`
	Menus      []models.Menu      `json:"menus"`
	FieldTypes []models.FieldType `json:"field_types"`
}

// EntitySchema represents the schema for a single entity
type EntitySchema struct {
	Entity    *models.Entity    `json:"entity"`
	Relations []models.Relation `json:"relations"`
	Views     []models.View     `json:"views"`
	Actions   []models.Action   `json:"actions"`
}
