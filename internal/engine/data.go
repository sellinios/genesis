// Package engine - Data Engine
// Handles all dynamic CRUD operations based on entity schema
package engine

import (
	"fmt"
	"strings"
	"time"

	"github.com/aethra/genesis/internal/models"
	"github.com/aethra/genesis/internal/security"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DataEngine handles all dynamic data operations
type DataEngine struct {
	db           *gorm.DB
	schemaEngine *SchemaEngine
}

// NewDataEngine creates a new data engine
func NewDataEngine(db *gorm.DB, schemaEngine *SchemaEngine) *DataEngine {
	return &DataEngine{
		db:           db,
		schemaEngine: schemaEngine,
	}
}

// =============================================================================
// QUERY TYPES
// =============================================================================

// QueryParams represents parameters for listing/filtering data
type QueryParams struct {
	Page     int                    `json:"page"`
	PageSize int                    `json:"page_size"`
	Sort     string                 `json:"sort"`
	SortDir  string                 `json:"sort_dir"`
	Search   string                 `json:"search"`
	Filters  map[string]interface{} `json:"filters"`
	Include  []string               `json:"include"` // Relations to include
}

// QueryResult represents the result of a list query
type QueryResult struct {
	Data       []map[string]interface{} `json:"data"`
	Total      int64                    `json:"total"`
	Page       int                      `json:"page"`
	PageSize   int                      `json:"page_size"`
	TotalPages int                      `json:"total_pages"`
}

// =============================================================================
// CRUD OPERATIONS
// =============================================================================

// List returns a paginated list of records for an entity
func (e *DataEngine) List(tenantID uuid.UUID, entityCode string, params QueryParams) (*QueryResult, error) {
	// Get entity schema
	schema, err := e.schemaEngine.GetEntitySchema(tenantID, entityCode)
	if err != nil {
		return nil, err
	}

	tableName, err := e.safeTableName(schema.Entity)
	if err != nil {
		return nil, err
	}

	// Default pagination
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 25
	}
	if params.PageSize > 100 {
		params.PageSize = 100
	}

	// Build base query with parameterized tenant_id
	query := e.db.Table(tableName).Where("tenant_id = ?", tenantID)

	// Apply soft delete filter
	if schema.Entity.UseSoftDelete {
		query = query.Where("deleted_at IS NULL")
	}

	// Apply search with parameterized query
	if params.Search != "" {
		searchCols := e.getSearchableColumns(schema.Entity.Fields)
		if len(searchCols) > 0 {
			escaped := security.EscapeLikePattern(params.Search)
			searchParam := "%" + escaped + "%"
			var conditions []string
			for _, col := range searchCols {
				conditions = append(conditions, fmt.Sprintf(`%s ILIKE ? ESCAPE '\'`, security.QuoteIdentifier(col)))
			}
			// Build OR conditions with same parameter for all
			conditionStr := "(" + strings.Join(conditions, " OR ") + ")"
			// Create parameter slice with same value for each condition
			searchParams := make([]interface{}, len(searchCols))
			for i := range searchParams {
				searchParams[i] = searchParam
			}
			query = query.Where(conditionStr, searchParams...)
		}
	}

	// Apply filters with validation
	for field, value := range params.Filters {
		if e.isValidField(schema.Entity.Fields, field) {
			// Validate and quote the field name
			if err := security.ValidateIdentifier(field); err != nil {
				continue // Skip invalid field names
			}
			quotedField := security.QuoteIdentifier(field)
			query = query.Where(fmt.Sprintf("%s = ?", quotedField), value)
		}
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count records: %w", err)
	}

	// Apply sorting with validation
	if params.Sort != "" && e.isValidField(schema.Entity.Fields, params.Sort) {
		if err := security.ValidateIdentifier(params.Sort); err == nil {
			sortDir := "ASC"
			if strings.ToUpper(params.SortDir) == "DESC" {
				sortDir = "DESC"
			}
			quotedSort := security.QuoteIdentifier(params.Sort)
			query = query.Order(fmt.Sprintf("%s %s", quotedSort, sortDir))
		}
	} else {
		query = query.Order("created_at DESC")
	}

	// Apply pagination
	offset := (params.Page - 1) * params.PageSize
	query = query.Offset(offset).Limit(params.PageSize)

	// Execute query
	var results []map[string]interface{}
	rows, err := query.Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to query records: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	for rows.Next() {
		record := make(map[string]interface{})
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))

		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		for i, col := range columns {
			record[col] = values[i]
		}

		results = append(results, record)
	}

	totalPages := int(total) / params.PageSize
	if int(total)%params.PageSize > 0 {
		totalPages++
	}

	return &QueryResult{
		Data:       results,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}, nil
}

// Get returns a single record by ID
func (e *DataEngine) Get(tenantID uuid.UUID, entityCode string, recordID uuid.UUID) (map[string]interface{}, error) {
	schema, err := e.schemaEngine.GetEntitySchema(tenantID, entityCode)
	if err != nil {
		return nil, err
	}

	tableName, err := e.safeTableName(schema.Entity)
	if err != nil {
		return nil, err
	}

	query := e.db.Table(tableName).Where("tenant_id = ? AND id = ?", tenantID, recordID)

	if schema.Entity.UseSoftDelete {
		query = query.Where("deleted_at IS NULL")
	}

	var result map[string]interface{}
	rows, err := query.Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to query record: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	if rows.Next() {
		result = make(map[string]interface{})
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))

		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		for i, col := range columns {
			result[col] = values[i]
		}
	} else {
		return nil, fmt.Errorf("record not found")
	}

	return result, nil
}

// Create creates a new record
func (e *DataEngine) Create(tenantID uuid.UUID, entityCode string, data map[string]interface{}, userID *uuid.UUID) (map[string]interface{}, error) {
	schema, err := e.schemaEngine.GetEntitySchema(tenantID, entityCode)
	if err != nil {
		return nil, err
	}

	tableName, err := e.safeTableName(schema.Entity)
	if err != nil {
		return nil, err
	}

	// Validate and filter data
	filteredData, err := e.validateAndFilterData(schema.Entity.Fields, data, true)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Add system fields
	newID := uuid.New()
	filteredData["id"] = newID
	filteredData["tenant_id"] = tenantID

	if schema.Entity.UseTimestamps {
		now := time.Now()
		filteredData["created_at"] = now
		filteredData["updated_at"] = now
	}

	// Build INSERT statement with validated column names
	columns := make([]string, 0, len(filteredData))
	placeholders := make([]string, 0, len(filteredData))
	values := make([]interface{}, 0, len(filteredData))

	i := 1
	for col, val := range filteredData {
		// Validate column name (system fields are always valid)
		if !isSystemField(col) {
			if err := security.ValidateIdentifier(col); err != nil {
				continue
			}
		}
		columns = append(columns, security.QuoteIdentifier(col))
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		values = append(values, val)
		i++
	}

	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING *",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))

	// Execute and get the created record
	rows, err := e.db.Raw(sql, values...).Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to create record: %w", err)
	}
	defer rows.Close()

	resultColumns, _ := rows.Columns()
	var result map[string]interface{}

	if rows.Next() {
		result = make(map[string]interface{})
		scanValues := make([]interface{}, len(resultColumns))
		valuePtrs := make([]interface{}, len(resultColumns))

		for i := range scanValues {
			valuePtrs[i] = &scanValues[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}

		for i, col := range resultColumns {
			result[col] = scanValues[i]
		}
	}

	// Create audit log
	if schema.Entity.UseAuditLog && userID != nil {
		e.createAuditLog(tenantID, userID, schema.Entity, newID, "create", nil, result)
	}

	return result, nil
}

// Update updates an existing record
func (e *DataEngine) Update(tenantID uuid.UUID, entityCode string, recordID uuid.UUID, data map[string]interface{}, userID *uuid.UUID) (map[string]interface{}, error) {
	schema, err := e.schemaEngine.GetEntitySchema(tenantID, entityCode)
	if err != nil {
		return nil, err
	}

	tableName, err := e.safeTableName(schema.Entity)
	if err != nil {
		return nil, err
	}

	// Get old values for audit
	var oldValues map[string]interface{}
	if schema.Entity.UseAuditLog && userID != nil {
		oldValues, _ = e.Get(tenantID, entityCode, recordID)
	}

	// Validate and filter data
	filteredData, err := e.validateAndFilterData(schema.Entity.Fields, data, false)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	if len(filteredData) == 0 {
		return nil, fmt.Errorf("no valid fields to update")
	}

	// Add updated_at
	if schema.Entity.UseTimestamps {
		filteredData["updated_at"] = time.Now()
	}

	// Build UPDATE statement with validated column names
	setClauses := make([]string, 0, len(filteredData))
	values := make([]interface{}, 0, len(filteredData)+2)

	i := 1
	for col, val := range filteredData {
		// Validate column name (system fields are always valid)
		if !isSystemField(col) {
			if err := security.ValidateIdentifier(col); err != nil {
				continue
			}
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", security.QuoteIdentifier(col), i))
		values = append(values, val)
		i++
	}

	values = append(values, tenantID, recordID)

	sql := fmt.Sprintf("UPDATE %s SET %s WHERE tenant_id = $%d AND id = $%d RETURNING *",
		tableName,
		strings.Join(setClauses, ", "),
		i,
		i+1)

	// Execute and get the updated record
	rows, err := e.db.Raw(sql, values...).Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to update record: %w", err)
	}
	defer rows.Close()

	resultColumns, _ := rows.Columns()
	var result map[string]interface{}

	if rows.Next() {
		result = make(map[string]interface{})
		scanValues := make([]interface{}, len(resultColumns))
		valuePtrs := make([]interface{}, len(resultColumns))

		for i := range scanValues {
			valuePtrs[i] = &scanValues[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}

		for i, col := range resultColumns {
			result[col] = scanValues[i]
		}
	} else {
		return nil, fmt.Errorf("record not found")
	}

	// Create audit log
	if schema.Entity.UseAuditLog && userID != nil {
		e.createAuditLog(tenantID, userID, schema.Entity, recordID, "update", oldValues, result)
	}

	return result, nil
}

// Delete deletes a record (soft or hard delete based on entity config)
func (e *DataEngine) Delete(tenantID uuid.UUID, entityCode string, recordID uuid.UUID, userID *uuid.UUID) error {
	schema, err := e.schemaEngine.GetEntitySchema(tenantID, entityCode)
	if err != nil {
		return err
	}

	tableName, err := e.safeTableName(schema.Entity)
	if err != nil {
		return err
	}

	// Get old values for audit
	var oldValues map[string]interface{}
	if schema.Entity.UseAuditLog && userID != nil {
		oldValues, _ = e.Get(tenantID, entityCode, recordID)
	}

	var sql string
	if schema.Entity.UseSoftDelete {
		sql = fmt.Sprintf("UPDATE %s SET deleted_at = $1 WHERE tenant_id = $2 AND id = $3",
			tableName)
		if err := e.db.Exec(sql, time.Now(), tenantID, recordID).Error; err != nil {
			return fmt.Errorf("failed to delete record: %w", err)
		}
	} else {
		sql = fmt.Sprintf("DELETE FROM %s WHERE tenant_id = $1 AND id = $2", tableName)
		if err := e.db.Exec(sql, tenantID, recordID).Error; err != nil {
			return fmt.Errorf("failed to delete record: %w", err)
		}
	}

	// Create audit log
	if schema.Entity.UseAuditLog && userID != nil {
		e.createAuditLog(tenantID, userID, schema.Entity, recordID, "delete", oldValues, nil)
	}

	return nil
}

// BulkDelete deletes multiple records
func (e *DataEngine) BulkDelete(tenantID uuid.UUID, entityCode string, recordIDs []uuid.UUID, userID *uuid.UUID) error {
	for _, id := range recordIDs {
		if err := e.Delete(tenantID, entityCode, id, userID); err != nil {
			return err
		}
	}
	return nil
}

// =============================================================================
// HELPER METHODS
// =============================================================================

func (e *DataEngine) safeTableName(entity *models.Entity) (string, error) {
	tableName := entity.TableName
	if tableName == "" {
		tableName = fmt.Sprintf("data_%s", entity.Code)
	}
	// Validate and quote table name
	if err := security.ValidateIdentifier(tableName); err != nil {
		return "", fmt.Errorf("invalid table name: %w", err)
	}
	return security.QuoteIdentifier(tableName), nil
}

func (e *DataEngine) getTableName(entity *models.Entity) string {
	if entity.TableName != "" {
		return entity.TableName
	}
	return fmt.Sprintf("data_%s", entity.Code)
}

func (e *DataEngine) isValidField(fields []models.Field, fieldCode string) bool {
	for _, f := range fields {
		if f.Code == fieldCode {
			return true
		}
	}
	// Also allow system fields
	return isSystemField(fieldCode)
}

func isSystemField(fieldCode string) bool {
	systemFields := []string{"id", "tenant_id", "created_at", "updated_at", "deleted_at"}
	for _, sf := range systemFields {
		if sf == fieldCode {
			return true
		}
	}
	return false
}

func (e *DataEngine) getSearchableColumns(fields []models.Field) []string {
	var cols []string
	for _, f := range fields {
		if f.InSearch {
			// Only search text-like fields
			if f.FieldType != nil {
				switch f.FieldType.Code {
				case "string", "text", "richtext", "email", "phone", "url":
					// Validate the column name before adding
					if err := security.ValidateIdentifier(f.Code); err == nil {
						cols = append(cols, f.Code)
					}
				}
			}
		}
	}
	return cols
}

func (e *DataEngine) validateAndFilterData(fields []models.Field, data map[string]interface{}, isCreate bool) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for _, field := range fields {
		// Skip system fields
		if field.IsSystem || field.IsAuto || field.IsPrimary {
			continue
		}

		// Validate field code
		if err := security.ValidateIdentifier(field.Code); err != nil {
			continue
		}

		value, exists := data[field.Code]

		// Check required
		if field.IsRequired && isCreate && (!exists || value == nil || value == "") {
			return nil, fmt.Errorf("field '%s' is required", field.Name)
		}

		if !exists {
			continue
		}

		// Validation based on field type
		if err := e.validateFieldValue(&field, value); err != nil {
			return nil, err
		}

		result[field.Code] = value
	}

	return result, nil
}

func (e *DataEngine) validateFieldValue(field *models.Field, value interface{}) error {
	if value == nil {
		return nil
	}

	// String length validation
	if strVal, ok := value.(string); ok {
		if field.MinLength != nil && len(strVal) < *field.MinLength {
			return fmt.Errorf("field '%s' must be at least %d characters", field.Name, *field.MinLength)
		}
		if field.MaxLength != nil && len(strVal) > *field.MaxLength {
			return fmt.Errorf("field '%s' must be at most %d characters", field.Name, *field.MaxLength)
		}
		// Regex pattern validation
		if field.RegexPattern != "" {
			// Note: In production, compile and cache regex patterns
			// For now, we skip complex regex validation
		}
	}

	// Numeric validation
	if numVal, ok := value.(float64); ok {
		if field.MinValue != nil && numVal < *field.MinValue {
			return fmt.Errorf("field '%s' must be at least %v", field.Name, *field.MinValue)
		}
		if field.MaxValue != nil && numVal > *field.MaxValue {
			return fmt.Errorf("field '%s' must be at most %v", field.Name, *field.MaxValue)
		}
	}

	return nil
}

func (e *DataEngine) createAuditLog(tenantID uuid.UUID, userID *uuid.UUID, entity *models.Entity, recordID uuid.UUID, action string, oldValues, newValues map[string]interface{}) {
	// Find changed fields
	var changedFields []string
	if oldValues != nil && newValues != nil {
		for key, newVal := range newValues {
			if oldVal, exists := oldValues[key]; exists {
				if fmt.Sprintf("%v", oldVal) != fmt.Sprintf("%v", newVal) {
					changedFields = append(changedFields, key)
				}
			}
		}
	}

	log := models.AuditLog{
		ID:            uuid.New(),
		TenantID:      tenantID,
		UserID:        userID,
		EntityID:      &entity.ID,
		EntityCode:    entity.Code,
		RecordID:      &recordID,
		Action:        action,
		OldValues:     oldValues,
		NewValues:     newValues,
		ChangedFields: changedFields,
		CreatedAt:     time.Now(),
	}

	e.db.Create(&log)
}
