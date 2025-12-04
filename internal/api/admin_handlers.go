// Package api - Admin handlers for Genesis management
package api

import (
	"net/http"

	"github.com/aethra/genesis/internal/auth"
	"github.com/aethra/genesis/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AdminHandler contains admin API handlers
type AdminHandler struct {
	db *gorm.DB
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(db *gorm.DB) *AdminHandler {
	return &AdminHandler{db: db}
}

// =============================================================================
// TENANT MANAGEMENT
// =============================================================================

// ListTenants returns all tenants
// GET /admin/tenants
func (h *AdminHandler) ListTenants(c *gin.Context) {
	var tenants []models.Tenant
	if err := h.db.Find(&tenants).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tenants)
}

// ListTenantsPublic returns active tenants (public - for login page)
// GET /api/tenants
func (h *AdminHandler) ListTenantsPublic(c *gin.Context) {
	var tenants []struct {
		ID   uuid.UUID `json:"id"`
		Name string    `json:"name"`
	}
	if err := h.db.Table("tenants").Select("id, name").Where("is_active = ?", true).Find(&tenants).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"tenants": tenants})
}

// GetTenant returns a single tenant
// GET /admin/tenants/:id
func (h *AdminHandler) GetTenant(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var tenant models.Tenant
	if err := h.db.First(&tenant, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
		return
	}
	c.JSON(http.StatusOK, tenant)
}

// CreateTenant creates a new tenant
// POST /admin/tenants
func (h *AdminHandler) CreateTenant(c *gin.Context) {
	var input struct {
		Code     string                 `json:"code" binding:"required"`
		Name     string                 `json:"name" binding:"required"`
		Domain   string                 `json:"domain"`
		Settings map[string]interface{} `json:"settings"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenant := models.Tenant{
		ID:       uuid.New(),
		Code:     input.Code,
		Name:     input.Name,
		Domain:   input.Domain,
		Settings: models.JSONB(input.Settings),
		IsActive: true,
	}

	if err := h.db.Create(&tenant).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, tenant)
}

// UpdateTenant updates a tenant
// PUT /admin/tenants/:id
func (h *AdminHandler) UpdateTenant(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var tenant models.Tenant
	if err := h.db.First(&tenant, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
		return
	}

	var input struct {
		Code     *string                `json:"code"`
		Name     *string                `json:"name"`
		Domain   *string                `json:"domain"`
		Settings map[string]interface{} `json:"settings"`
		IsActive *bool                  `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Code != nil {
		tenant.Code = *input.Code
	}
	if input.Name != nil {
		tenant.Name = *input.Name
	}
	if input.Domain != nil {
		tenant.Domain = *input.Domain
	}
	if input.Settings != nil {
		tenant.Settings = models.JSONB(input.Settings)
	}
	if input.IsActive != nil {
		tenant.IsActive = *input.IsActive
	}

	if err := h.db.Save(&tenant).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tenant)
}

// DeleteTenant deletes a tenant
// DELETE /admin/tenants/:id
func (h *AdminHandler) DeleteTenant(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.db.Delete(&models.Tenant{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "tenant deleted"})
}

// =============================================================================
// MODULE MANAGEMENT
// =============================================================================

// ListModules returns all modules (optionally filtered by tenant_id query param)
// GET /admin/modules?tenant_id=xxx
func (h *AdminHandler) ListModules(c *gin.Context) {
	var modules []models.Module
	query := h.db.Order("display_order")

	if tenantIDStr := c.Query("tenant_id"); tenantIDStr != "" {
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
			return
		}
		query = query.Where("tenant_id = ?", tenantID)
	}

	if err := query.Find(&modules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, modules)
}

// GetModule returns a single module
// GET /admin/modules/:id
func (h *AdminHandler) GetModule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var module models.Module
	if err := h.db.First(&module, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "module not found"})
		return
	}
	c.JSON(http.StatusOK, module)
}

// CreateModule creates a new module
// POST /admin/modules
func (h *AdminHandler) CreateModule(c *gin.Context) {
	var input struct {
		TenantID     string `json:"tenant_id" binding:"required"`
		Code         string `json:"code" binding:"required"`
		Name         string `json:"name" binding:"required"`
		Description  string `json:"description"`
		Icon         string `json:"icon"`
		Color        string `json:"color"`
		DisplayOrder int    `json:"display_order"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID, err := uuid.Parse(input.TenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}

	module := models.Module{
		ID:           uuid.New(),
		TenantID:     tenantID,
		Code:         input.Code,
		Name:         input.Name,
		Description:  input.Description,
		Icon:         input.Icon,
		Color:        input.Color,
		DisplayOrder: input.DisplayOrder,
		IsActive:     true,
	}

	if err := h.db.Create(&module).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, module)
}

// UpdateModule updates a module
// PUT /admin/modules/:id
func (h *AdminHandler) UpdateModule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var module models.Module
	if err := h.db.First(&module, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "module not found"})
		return
	}

	var input struct {
		Code         *string `json:"code"`
		Name         *string `json:"name"`
		Description  *string `json:"description"`
		Icon         *string `json:"icon"`
		Color        *string `json:"color"`
		DisplayOrder *int    `json:"display_order"`
		IsActive     *bool   `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Code != nil {
		module.Code = *input.Code
	}
	if input.Name != nil {
		module.Name = *input.Name
	}
	if input.Description != nil {
		module.Description = *input.Description
	}
	if input.Icon != nil {
		module.Icon = *input.Icon
	}
	if input.Color != nil {
		module.Color = *input.Color
	}
	if input.DisplayOrder != nil {
		module.DisplayOrder = *input.DisplayOrder
	}
	if input.IsActive != nil {
		module.IsActive = *input.IsActive
	}

	if err := h.db.Save(&module).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, module)
}

// DeleteModule deletes a module
// DELETE /admin/modules/:id
func (h *AdminHandler) DeleteModule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.db.Delete(&models.Module{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "module deleted"})
}

// =============================================================================
// ENTITY MANAGEMENT
// =============================================================================

// ListEntities returns all entities (optionally filtered by module_id query param)
// GET /admin/entities?module_id=xxx
func (h *AdminHandler) ListEntities(c *gin.Context) {
	var entities []models.Entity
	query := h.db.Order("display_order")

	if moduleIDStr := c.Query("module_id"); moduleIDStr != "" {
		moduleID, err := uuid.Parse(moduleIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid module_id"})
			return
		}
		query = query.Where("module_id = ?", moduleID)
	}

	if err := query.Find(&entities).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, entities)
}

// GetEntity returns a single entity with its fields
// GET /admin/entities/:id
func (h *AdminHandler) GetEntity(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var entity models.Entity
	if err := h.db.First(&entity, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "entity not found"})
		return
	}

	// Get fields for this entity
	var fields []models.Field
	h.db.Where("entity_id = ?", id).Order("display_order").Find(&fields)

	c.JSON(http.StatusOK, gin.H{
		"entity": entity,
		"fields": fields,
	})
}

// CreateEntity creates a new entity
// POST /admin/entities
func (h *AdminHandler) CreateEntity(c *gin.Context) {
	var input struct {
		ModuleID     string `json:"module_id" binding:"required"`
		Code         string `json:"code" binding:"required"`
		Name         string `json:"name" binding:"required"`
		NamePlural   string `json:"name_plural"`
		Description  string `json:"description"`
		TableName    string `json:"table_name"`
		Icon         string `json:"icon"`
		Color        string `json:"color"`
		DisplayOrder int    `json:"display_order"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	moduleID, err := uuid.Parse(input.ModuleID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid module_id"})
		return
	}

	// Get module to find tenant_id
	var module models.Module
	if err := h.db.First(&module, "id = ?", moduleID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "module not found"})
		return
	}

	// Generate table name if not provided
	tableName := input.TableName
	if tableName == "" {
		tableName = "data_" + input.Code
	}

	entity := models.Entity{
		ID:           uuid.New(),
		TenantID:     module.TenantID,
		ModuleID:     &moduleID,
		Code:         input.Code,
		Name:         input.Name,
		NamePlural:   input.NamePlural,
		Description:  input.Description,
		TableName:    tableName,
		Icon:         input.Icon,
		Color:        input.Color,
		DisplayOrder: input.DisplayOrder,
		IsActive:     true,
	}

	if err := h.db.Create(&entity).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, entity)
}

// UpdateEntity updates an entity
// PUT /admin/entities/:id
func (h *AdminHandler) UpdateEntity(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var entity models.Entity
	if err := h.db.First(&entity, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "entity not found"})
		return
	}

	var input struct {
		Code         *string `json:"code"`
		Name         *string `json:"name"`
		NamePlural   *string `json:"name_plural"`
		Description  *string `json:"description"`
		Icon         *string `json:"icon"`
		Color        *string `json:"color"`
		DisplayOrder *int    `json:"display_order"`
		IsActive     *bool   `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Code != nil {
		entity.Code = *input.Code
	}
	if input.Name != nil {
		entity.Name = *input.Name
	}
	if input.NamePlural != nil {
		entity.NamePlural = *input.NamePlural
	}
	if input.Description != nil {
		entity.Description = *input.Description
	}
	if input.Icon != nil {
		entity.Icon = *input.Icon
	}
	if input.Color != nil {
		entity.Color = *input.Color
	}
	if input.DisplayOrder != nil {
		entity.DisplayOrder = *input.DisplayOrder
	}
	if input.IsActive != nil {
		entity.IsActive = *input.IsActive
	}

	if err := h.db.Save(&entity).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, entity)
}

// DeleteEntity deletes an entity
// DELETE /admin/entities/:id
func (h *AdminHandler) DeleteEntity(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.db.Delete(&models.Entity{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "entity deleted"})
}

// =============================================================================
// FIELD MANAGEMENT
// =============================================================================

// ListFieldTypes returns all available field types
// GET /admin/field-types
func (h *AdminHandler) ListFieldTypes(c *gin.Context) {
	var fieldTypes []models.FieldType
	if err := h.db.Find(&fieldTypes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, fieldTypes)
}

// ListFields returns all fields (optionally filtered by entity_id query param)
// GET /admin/fields?entity_id=xxx
func (h *AdminHandler) ListFields(c *gin.Context) {
	var fields []models.Field
	query := h.db.Order("display_order")

	if entityIDStr := c.Query("entity_id"); entityIDStr != "" {
		entityID, err := uuid.Parse(entityIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid entity_id"})
			return
		}
		query = query.Where("entity_id = ?", entityID)
	}

	if err := query.Find(&fields).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, fields)
}

// GetField returns a single field
// GET /admin/fields/:id
func (h *AdminHandler) GetField(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var field models.Field
	if err := h.db.First(&field, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "field not found"})
		return
	}
	c.JSON(http.StatusOK, field)
}

// CreateField creates a new field
// POST /admin/fields
func (h *AdminHandler) CreateField(c *gin.Context) {
	var input struct {
		EntityID     string                 `json:"entity_id" binding:"required"`
		FieldTypeID  string                 `json:"field_type_id" binding:"required"`
		Code         string                 `json:"code" binding:"required"`
		Name         string                 `json:"name" binding:"required"`
		Description  string                 `json:"description"`
		ColumnName   string                 `json:"column_name"`
		ColumnType   string                 `json:"column_type"`
		DefaultValue *string                `json:"default_value"`
		Placeholder  string                 `json:"placeholder"`
		HelpText     string                 `json:"help_text"`
		IsRequired   bool                   `json:"is_required"`
		IsUnique     bool                   `json:"is_unique"`
		InList       bool                   `json:"in_list"`
		InDetail     bool                   `json:"in_detail"`
		InForm       bool                   `json:"in_form"`
		InSearch     bool                   `json:"in_search"`
		InFilter     bool                   `json:"in_filter"`
		InSort       bool                   `json:"in_sort"`
		DisplayOrder int                    `json:"display_order"`
		Settings     map[string]interface{} `json:"settings"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entityID, err := uuid.Parse(input.EntityID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid entity_id"})
		return
	}

	// Get entity to find tenant_id
	var entity models.Entity
	if err := h.db.First(&entity, "id = ?", entityID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "entity not found"})
		return
	}

	fieldTypeID, err := uuid.Parse(input.FieldTypeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid field_type_id"})
		return
	}

	// Generate column name if not provided
	columnName := input.ColumnName
	if columnName == "" {
		columnName = input.Code
	}

	field := models.Field{
		ID:           uuid.New(),
		TenantID:     entity.TenantID,
		EntityID:     entityID,
		FieldTypeID:  &fieldTypeID,
		Code:         input.Code,
		Name:         input.Name,
		Description:  input.Description,
		ColumnName:   columnName,
		ColumnType:   input.ColumnType,
		DefaultValue: input.DefaultValue,
		Placeholder:  input.Placeholder,
		HelpText:     input.HelpText,
		IsRequired:   input.IsRequired,
		IsUnique:     input.IsUnique,
		InList:       input.InList,
		InDetail:     input.InDetail,
		InForm:       input.InForm,
		InSearch:     input.InSearch,
		InFilter:     input.InFilter,
		InSort:       input.InSort,
		DisplayOrder: input.DisplayOrder,
		Settings:     models.JSONB(input.Settings),
		IsActive:     true,
	}

	if err := h.db.Create(&field).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, field)
}

// UpdateField updates a field
// PUT /admin/fields/:id
func (h *AdminHandler) UpdateField(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var field models.Field
	if err := h.db.First(&field, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "field not found"})
		return
	}

	var input struct {
		Code         *string                `json:"code"`
		Name         *string                `json:"name"`
		Description  *string                `json:"description"`
		DefaultValue *string                `json:"default_value"`
		Placeholder  *string                `json:"placeholder"`
		HelpText     *string                `json:"help_text"`
		IsRequired   *bool                  `json:"is_required"`
		IsUnique     *bool                  `json:"is_unique"`
		InList       *bool                  `json:"in_list"`
		InDetail     *bool                  `json:"in_detail"`
		InForm       *bool                  `json:"in_form"`
		InSearch     *bool                  `json:"in_search"`
		InFilter     *bool                  `json:"in_filter"`
		InSort       *bool                  `json:"in_sort"`
		DisplayOrder *int                   `json:"display_order"`
		Settings     map[string]interface{} `json:"settings"`
		IsActive     *bool                  `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Code != nil {
		field.Code = *input.Code
	}
	if input.Name != nil {
		field.Name = *input.Name
	}
	if input.Description != nil {
		field.Description = *input.Description
	}
	if input.DefaultValue != nil {
		field.DefaultValue = input.DefaultValue
	}
	if input.Placeholder != nil {
		field.Placeholder = *input.Placeholder
	}
	if input.HelpText != nil {
		field.HelpText = *input.HelpText
	}
	if input.IsRequired != nil {
		field.IsRequired = *input.IsRequired
	}
	if input.IsUnique != nil {
		field.IsUnique = *input.IsUnique
	}
	if input.InList != nil {
		field.InList = *input.InList
	}
	if input.InDetail != nil {
		field.InDetail = *input.InDetail
	}
	if input.InForm != nil {
		field.InForm = *input.InForm
	}
	if input.InSearch != nil {
		field.InSearch = *input.InSearch
	}
	if input.InFilter != nil {
		field.InFilter = *input.InFilter
	}
	if input.InSort != nil {
		field.InSort = *input.InSort
	}
	if input.DisplayOrder != nil {
		field.DisplayOrder = *input.DisplayOrder
	}
	if input.Settings != nil {
		field.Settings = models.JSONB(input.Settings)
	}
	if input.IsActive != nil {
		field.IsActive = *input.IsActive
	}

	if err := h.db.Save(&field).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, field)
}

// DeleteField deletes a field
// DELETE /admin/fields/:id
func (h *AdminHandler) DeleteField(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.db.Delete(&models.Field{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "field deleted"})
}

// =============================================================================
// USER MANAGEMENT
// =============================================================================

// ListUsers returns all users (optionally filtered by tenant_id query param)
// GET /admin/users?tenant_id=xxx
func (h *AdminHandler) ListUsers(c *gin.Context) {
	var users []models.User
	query := h.db

	if tenantIDStr := c.Query("tenant_id"); tenantIDStr != "" {
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
			return
		}
		query = query.Where("tenant_id = ?", tenantID)
	}

	if err := query.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Remove password hashes from response
	for i := range users {
		users[i].PasswordHash = ""
	}

	c.JSON(http.StatusOK, users)
}

// CreateUser creates a new user
// POST /admin/users
func (h *AdminHandler) CreateUser(c *gin.Context) {
	var input struct {
		TenantID  string                 `json:"tenant_id" binding:"required"`
		Email     string                 `json:"email" binding:"required,email"`
		Password  string                 `json:"password" binding:"required,min=8"`
		FirstName string                 `json:"first_name"`
		LastName  string                 `json:"last_name"`
		Settings  map[string]interface{} `json:"settings"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID, err := uuid.Parse(input.TenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}

	// Hash password with bcrypt
	passwordHash, err := auth.HashPassword(input.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process password"})
		return
	}

	user := models.User{
		ID:           uuid.New(),
		TenantID:     tenantID,
		Email:        input.Email,
		PasswordHash: passwordHash,
		FirstName:    input.FirstName,
		LastName:     input.LastName,
		Settings:     models.JSONB(input.Settings),
		IsActive:     true,
	}

	if err := h.db.Create(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Remove password hash from response
	user.PasswordHash = ""
	c.JSON(http.StatusCreated, user)
}
