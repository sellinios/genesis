// Package auth - Permission checking
package auth

import (
	"log"

	"github.com/aethra/genesis/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Action represents a permission action
type Action string

const (
	ActionView   Action = "view"
	ActionCreate Action = "create"
	ActionEdit   Action = "edit"
	ActionDelete Action = "delete"
	ActionExport Action = "export"
	ActionImport Action = "import"
)

// Permission represents a permission entry
type Permission struct {
	ID               uuid.UUID
	TenantID         uuid.UUID
	RoleID           uuid.UUID
	EntityID         uuid.UUID
	CanView          bool
	CanCreate        bool
	CanEdit          bool
	CanDelete        bool
	CanExport        bool
	CanImport        bool
	FieldPermissions map[string]interface{} `gorm:"type:jsonb"`
	RowFilter        map[string]interface{} `gorm:"type:jsonb"`
}

// PermissionService handles permission checks
type PermissionService struct {
	db *gorm.DB
}

// NewPermissionService creates a new permission service
func NewPermissionService(db *gorm.DB) *PermissionService {
	return &PermissionService{db: db}
}

// UserPermission represents computed permissions for a user on an entity
type UserPermission struct {
	CanView          bool
	CanCreate        bool
	CanEdit          bool
	CanDelete        bool
	CanExport        bool
	CanImport        bool
	FieldPermissions map[string]FieldPermission
	RowFilter        map[string]interface{}
}

// FieldPermission represents permissions for a specific field
type FieldPermission struct {
	CanView bool `json:"view"`
	CanEdit bool `json:"edit"`
}

// CheckPermission checks if a user has permission for an action on an entity
func (s *PermissionService) CheckPermission(tenantID, userID uuid.UUID, entityCode string, action Action) (bool, error) {
	perm, err := s.GetUserPermission(tenantID, userID, entityCode)
	if err != nil {
		return false, err
	}

	switch action {
	case ActionView:
		return perm.CanView, nil
	case ActionCreate:
		return perm.CanCreate, nil
	case ActionEdit:
		return perm.CanEdit, nil
	case ActionDelete:
		return perm.CanDelete, nil
	case ActionExport:
		return perm.CanExport, nil
	case ActionImport:
		return perm.CanImport, nil
	default:
		return false, nil
	}
}

// GetUserPermission returns the computed permissions for a user on an entity
func (s *PermissionService) GetUserPermission(tenantID, userID uuid.UUID, entityCode string) (*UserPermission, error) {
	log.Printf("GetUserPermission: tenantID=%s, userID=%s, entityCode=%s", tenantID, userID, entityCode)

	// Get all user's roles
	var roleIDs []uuid.UUID
	err := s.db.Table("user_roles").
		Where("user_id = ?", userID).
		Pluck("role_id", &roleIDs).Error

	if err != nil {
		log.Printf("Error getting roles: %v", err)
		return nil, err
	}

	log.Printf("Found %d roles: %v", len(roleIDs), roleIDs)

	// If user has no roles, return no permissions
	if len(roleIDs) == 0 {
		log.Printf("No roles, returning empty permissions")
		return &UserPermission{}, nil
	}

	// Get entity ID from code
	var entity struct{ ID uuid.UUID }
	err = s.db.Table("entities").
		Select("id").
		Where("tenant_id = ? AND code = ?", tenantID, entityCode).
		First(&entity).Error

	if err != nil {
		log.Printf("Error getting entity: %v", err)
		// If entity not found, deny all
		return &UserPermission{}, nil
	}

	entityID := entity.ID
	log.Printf("Found entity ID: %s", entityID)

	// Get all permissions for user's roles on this entity
	var permissions []models.Permission
	err = s.db.Where("tenant_id = ? AND entity_id = ? AND role_id IN ?", tenantID, entityID, roleIDs).
		Find(&permissions).Error

	if err != nil {
		log.Printf("Error getting permissions: %v", err)
		return nil, err
	}

	log.Printf("Found %d permissions", len(permissions))
	for _, p := range permissions {
		log.Printf("  Permission: can_view=%v, can_create=%v", p.CanView, p.CanCreate)
	}

	// Merge permissions (OR logic - if any role grants permission, user has it)
	result := &UserPermission{
		FieldPermissions: make(map[string]FieldPermission),
	}

	for _, perm := range permissions {
		result.CanView = result.CanView || perm.CanView
		result.CanCreate = result.CanCreate || perm.CanCreate
		result.CanEdit = result.CanEdit || perm.CanEdit
		result.CanDelete = result.CanDelete || perm.CanDelete
		result.CanExport = result.CanExport || perm.CanExport
		result.CanImport = result.CanImport || perm.CanImport

		// Merge field permissions
		if perm.FieldPermissions != nil {
			for field, perms := range perm.FieldPermissions {
				if fp, ok := perms.(map[string]interface{}); ok {
					existing := result.FieldPermissions[field]
					if viewPerm, ok := fp["view"].(bool); ok {
						existing.CanView = existing.CanView || viewPerm
					}
					if editPerm, ok := fp["edit"].(bool); ok {
						existing.CanEdit = existing.CanEdit || editPerm
					}
					result.FieldPermissions[field] = existing
				}
			}
		}

		// Row filters are merged (user sees records matching ANY of their role filters)
		// For simplicity, we use the first non-nil filter
		if result.RowFilter == nil && perm.RowFilter != nil {
			result.RowFilter = perm.RowFilter
		}
	}

	return result, nil
}

// GetRowFilter returns the row filter for a user on an entity
// This can be used to add WHERE conditions to queries
func (s *PermissionService) GetRowFilter(tenantID, userID uuid.UUID, entityCode string) (map[string]interface{}, error) {
	perm, err := s.GetUserPermission(tenantID, userID, entityCode)
	if err != nil {
		return nil, err
	}
	return perm.RowFilter, nil
}

// CanAccessField checks if a user can view a specific field
func (s *PermissionService) CanAccessField(tenantID, userID uuid.UUID, entityCode, fieldCode string) (bool, error) {
	perm, err := s.GetUserPermission(tenantID, userID, entityCode)
	if err != nil {
		return false, err
	}

	// If no field-level permissions defined, allow all
	if len(perm.FieldPermissions) == 0 {
		return perm.CanView, nil
	}

	// Check specific field permission
	if fp, ok := perm.FieldPermissions[fieldCode]; ok {
		return fp.CanView, nil
	}

	// Default to entity-level view permission
	return perm.CanView, nil
}

// CanEditField checks if a user can edit a specific field
func (s *PermissionService) CanEditField(tenantID, userID uuid.UUID, entityCode, fieldCode string) (bool, error) {
	perm, err := s.GetUserPermission(tenantID, userID, entityCode)
	if err != nil {
		return false, err
	}

	// If no field-level permissions defined, use entity-level
	if len(perm.FieldPermissions) == 0 {
		return perm.CanEdit, nil
	}

	// Check specific field permission
	if fp, ok := perm.FieldPermissions[fieldCode]; ok {
		return fp.CanEdit, nil
	}

	// Default to entity-level edit permission
	return perm.CanEdit, nil
}
