// Package models - UI models for dynamic React components
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// UIComponent represents a React component stored in the database
type UIComponent struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	TenantID    *uuid.UUID     `json:"tenant_id" gorm:"type:uuid"` // NULL = system component
	Code        string         `json:"code" gorm:"uniqueIndex;not null"`
	Name        string         `json:"name" gorm:"not null"`
	Description string         `json:"description"`
	Category    string         `json:"category"` // layout, input, display, data

	CodeJSX     string         `json:"code_jsx" gorm:"not null"` // The actual React component code
	PropsSchema datatypes.JSON `json:"props_schema" gorm:"default:'{}'"`

	Dependencies []string `json:"dependencies" gorm:"type:text[]"`

	IsSystem  bool `json:"is_system" gorm:"default:false"`
	IsActive  bool `json:"is_active" gorm:"default:true"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Tenant *Tenant `json:"tenant,omitempty" gorm:"foreignKey:TenantID"`
}

func (UIComponent) TableName() string {
	return "ui_components"
}

// UILayout represents a page layout with slots
type UILayout struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	TenantID    *uuid.UUID     `json:"tenant_id" gorm:"type:uuid"`
	Code        string         `json:"code" gorm:"not null"`
	Name        string         `json:"name" gorm:"not null"`
	Description string         `json:"description"`

	CodeJSX string         `json:"code_jsx" gorm:"not null"` // Layout with {children}, {sidebar}, etc.
	Slots   datatypes.JSON `json:"slots" gorm:"default:'[\"content\"]'"`

	IsDefault bool `json:"is_default" gorm:"default:false"`
	IsSystem  bool `json:"is_system" gorm:"default:false"`
	IsActive  bool `json:"is_active" gorm:"default:true"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Tenant *Tenant `json:"tenant,omitempty" gorm:"foreignKey:TenantID"`
}

func (UILayout) TableName() string {
	return "ui_layouts"
}

// UIPage represents a dynamic page
type UIPage struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	TenantID    *uuid.UUID     `json:"tenant_id" gorm:"type:uuid"`
	Code        string         `json:"code" gorm:"not null"`
	Name        string         `json:"name" gorm:"not null"`
	Description string         `json:"description"`

	Route    string     `json:"route" gorm:"not null"` // '/dashboard', '/m/:module/:entity'
	LayoutID *uuid.UUID `json:"layout_id" gorm:"type:uuid"`

	PageType string     `json:"page_type" gorm:"not null"` // static, entity_list, entity_detail, entity_form, custom
	EntityID *uuid.UUID `json:"entity_id" gorm:"type:uuid"`

	CodeJSX    string         `json:"code_jsx"`                       // Custom page code
	Components datatypes.JSON `json:"components" gorm:"default:'[]'"` // Component instances with props

	Title              string `json:"title"`
	Icon               string `json:"icon"`
	RequiresAuth       bool   `json:"requires_auth" gorm:"default:true"`
	RequiredPermission string `json:"required_permission"`

	IsSystem     bool `json:"is_system" gorm:"default:false"`
	IsActive     bool `json:"is_active" gorm:"default:true"`
	DisplayOrder int  `json:"display_order" gorm:"default:0"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Tenant *Tenant   `json:"tenant,omitempty" gorm:"foreignKey:TenantID"`
	Layout *UILayout `json:"layout,omitempty" gorm:"foreignKey:LayoutID"`
	Entity *Entity   `json:"entity,omitempty" gorm:"foreignKey:EntityID"`
}

func (UIPage) TableName() string {
	return "ui_pages"
}

// UITheme represents a visual theme
type UITheme struct {
	ID       uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	TenantID *uuid.UUID     `json:"tenant_id" gorm:"type:uuid"`
	Code     string         `json:"code" gorm:"not null"`
	Name     string         `json:"name" gorm:"not null"`

	Variables datatypes.JSON `json:"variables" gorm:"not null"`
	CustomCSS string         `json:"custom_css"`

	IsDefault bool `json:"is_default" gorm:"default:false"`
	IsActive  bool `json:"is_active" gorm:"default:true"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Tenant *Tenant `json:"tenant,omitempty" gorm:"foreignKey:TenantID"`
}

func (UITheme) TableName() string {
	return "ui_themes"
}
