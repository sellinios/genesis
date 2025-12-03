// Package models contains the core Genesis data structures
// These models represent the meta-schema that defines everything dynamically
package models

import (
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// SYSTEM MODELS
// =============================================================================

// Tenant represents a customer/organization in the multi-tenant system
type Tenant struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Code      string    `json:"code" gorm:"uniqueIndex;not null;size:50"`
	Name      string    `json:"name" gorm:"not null;size:255"`
	Domain    string    `json:"domain" gorm:"size:255"`
	Settings  JSONB     `json:"settings" gorm:"type:jsonb;default:'{}'"`
	IsActive  bool      `json:"is_active" gorm:"default:true"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relations
	Users    []User   `json:"users,omitempty" gorm:"foreignKey:TenantID"`
	Modules  []Module `json:"modules,omitempty" gorm:"foreignKey:TenantID"`
	Entities []Entity `json:"entities,omitempty" gorm:"foreignKey:TenantID"`
}

// User represents a system user
type User struct {
	ID           uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TenantID     uuid.UUID  `json:"tenant_id" gorm:"type:uuid;index"`
	Email        string     `json:"email" gorm:"not null;size:255"`
	PasswordHash string     `json:"-" gorm:"size:255"`
	FirstName    string     `json:"first_name" gorm:"size:100"`
	LastName     string     `json:"last_name" gorm:"size:100"`
	AvatarURL    string     `json:"avatar_url"`
	Settings     JSONB      `json:"settings" gorm:"type:jsonb;default:'{}'"`
	IsActive     bool       `json:"is_active" gorm:"default:true"`
	LastLoginAt  *time.Time `json:"last_login_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	// Relations
	Tenant *Tenant `json:"tenant,omitempty" gorm:"foreignKey:TenantID"`
	Roles  []Role  `json:"roles,omitempty" gorm:"many2many:user_roles;"`
}

// =============================================================================
// META-SCHEMA MODELS
// =============================================================================

// Module represents a grouping of entities (CRM, ERP, HRM, etc.)
type Module struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TenantID     uuid.UUID `json:"tenant_id" gorm:"type:uuid;index"`
	Code         string    `json:"code" gorm:"not null;size:50"`
	Name         string    `json:"name" gorm:"not null;size:100"`
	Description  string    `json:"description"`
	Icon         string    `json:"icon" gorm:"size:50"`
	Color        string    `json:"color" gorm:"size:20"`
	DisplayOrder int       `json:"display_order" gorm:"default:0"`
	IsActive     bool      `json:"is_active" gorm:"default:true"`
	Settings     JSONB     `json:"settings" gorm:"type:jsonb;default:'{}'"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Relations
	Tenant   *Tenant  `json:"tenant,omitempty" gorm:"foreignKey:TenantID"`
	Entities []Entity `json:"entities,omitempty" gorm:"foreignKey:ModuleID"`
}

// Entity represents a dynamic "table" - the customer defines what they want
type Entity struct {
	ID            uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TenantID      uuid.UUID  `json:"tenant_id" gorm:"type:uuid;index"`
	ModuleID      *uuid.UUID `json:"module_id" gorm:"type:uuid;index"`
	Code          string     `json:"code" gorm:"not null;size:50"`
	Name          string     `json:"name" gorm:"not null;size:100"`
	NamePlural    string     `json:"name_plural" gorm:"size:100"`
	Description   string     `json:"description"`
	TableName     string     `json:"table_name" gorm:"size:100"`
	Icon          string     `json:"icon" gorm:"size:50"`
	Color         string     `json:"color" gorm:"size:20"`
	IsSystem      bool       `json:"is_system" gorm:"default:false"`
	IsActive      bool       `json:"is_active" gorm:"default:true"`
	AllowCreate   bool       `json:"allow_create" gorm:"default:true"`
	AllowEdit     bool       `json:"allow_edit" gorm:"default:true"`
	AllowDelete   bool       `json:"allow_delete" gorm:"default:true"`
	UseSoftDelete bool       `json:"use_soft_delete" gorm:"default:true"`
	UseTimestamps bool       `json:"use_timestamps" gorm:"default:true"`
	UseAuditLog   bool       `json:"use_audit_log" gorm:"default:true"`
	Settings      JSONB      `json:"settings" gorm:"type:jsonb;default:'{}'"`
	DisplayOrder  int        `json:"display_order" gorm:"default:0"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`

	// Relations
	Tenant *Tenant `json:"tenant,omitempty" gorm:"foreignKey:TenantID"`
	Module *Module `json:"module,omitempty" gorm:"foreignKey:ModuleID"`
	Fields []Field `json:"fields,omitempty" gorm:"foreignKey:EntityID"`
	Views  []View  `json:"views,omitempty" gorm:"foreignKey:EntityID"`
}

// FieldType represents the available field types in Genesis
type FieldType struct {
	ID               uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Code             string    `json:"code" gorm:"uniqueIndex;not null;size:50"`
	Name             string    `json:"name" gorm:"not null;size:100"`
	Category         string    `json:"category" gorm:"size:50"`
	DBType           string    `json:"db_type" gorm:"size:50"`
	DBDefaultLength  *int      `json:"db_default_length"`
	ValidationRules  JSONB     `json:"validation_rules" gorm:"type:jsonb;default:'{}'"`
	DefaultComponent string    `json:"default_component" gorm:"size:50"`
	ComponentOptions JSONB     `json:"component_options" gorm:"type:jsonb;default:'{}'"`
	Settings         JSONB     `json:"settings" gorm:"type:jsonb;default:'{}'"`
	IsSystem         bool      `json:"is_system" gorm:"default:true"`
	CreatedAt        time.Time `json:"created_at"`
}

// Field represents a field/column in an entity
type Field struct {
	ID               uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TenantID         uuid.UUID  `json:"tenant_id" gorm:"type:uuid;index"`
	EntityID         uuid.UUID  `json:"entity_id" gorm:"type:uuid;index"`
	FieldTypeID      *uuid.UUID `json:"field_type_id" gorm:"type:uuid"`
	Code             string     `json:"code" gorm:"not null;size:50"`
	Name             string     `json:"name" gorm:"not null;size:100"`
	Description      string     `json:"description"`
	ColumnName       string     `json:"column_name" gorm:"size:100"`
	ColumnType       string     `json:"column_type" gorm:"size:50"`
	IsRequired       bool       `json:"is_required" gorm:"default:false"`
	IsUnique         bool       `json:"is_unique" gorm:"default:false"`
	IsPrimary        bool       `json:"is_primary" gorm:"default:false"`
	IsAuto           bool       `json:"is_auto" gorm:"default:false"`
	DefaultValue     *string    `json:"default_value"`
	MinLength        *int       `json:"min_length"`
	MaxLength        *int       `json:"max_length"`
	MinValue         *float64   `json:"min_value"`
	MaxValue         *float64   `json:"max_value"`
	RegexPattern     string     `json:"regex_pattern" gorm:"size:255"`
	ValidationRules  JSONB      `json:"validation_rules" gorm:"type:jsonb;default:'{}'"`
	DisplayOrder     int        `json:"display_order" gorm:"default:0"`
	InList           bool       `json:"in_list" gorm:"default:false"`
	InDetail         bool       `json:"in_detail" gorm:"default:true"`
	InForm           bool       `json:"in_form" gorm:"default:true"`
	InSearch         bool       `json:"in_search" gorm:"default:false"`
	InFilter         bool       `json:"in_filter" gorm:"default:false"`
	InSort           bool       `json:"in_sort" gorm:"default:false"`
	Placeholder      string     `json:"placeholder"`
	HelpText         string     `json:"help_text"`
	Component        string     `json:"component" gorm:"size:50"`
	ComponentOptions JSONB      `json:"component_options" gorm:"type:jsonb;default:'{}'"`
	IsSystem         bool       `json:"is_system" gorm:"default:false"`
	IsActive         bool       `json:"is_active" gorm:"default:true"`
	Settings         JSONB      `json:"settings" gorm:"type:jsonb;default:'{}'"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`

	// Relations
	Entity    *Entity    `json:"entity,omitempty" gorm:"foreignKey:EntityID"`
	FieldType *FieldType `json:"field_type,omitempty" gorm:"foreignKey:FieldTypeID"`
}

// Relation represents relationships between entities
type Relation struct {
	ID              uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TenantID        uuid.UUID `json:"tenant_id" gorm:"type:uuid;index"`
	SourceEntityID  uuid.UUID `json:"source_entity_id" gorm:"type:uuid;index"`
	SourceFieldCode string    `json:"source_field_code" gorm:"not null;size:50"`
	TargetEntityID  uuid.UUID `json:"target_entity_id" gorm:"type:uuid;index"`
	TargetFieldCode string    `json:"target_field_code" gorm:"default:'id';size:50"`
	RelationType    string    `json:"relation_type" gorm:"not null;size:20"`
	Name            string    `json:"name" gorm:"size:100"`
	OnDelete        string    `json:"on_delete" gorm:"default:'SET NULL';size:20"`
	IsRequired      bool      `json:"is_required" gorm:"default:false"`
	JunctionTable   string    `json:"junction_table" gorm:"size:100"`
	Settings        JSONB     `json:"settings" gorm:"type:jsonb;default:'{}'"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Relations
	SourceEntity *Entity `json:"source_entity,omitempty" gorm:"foreignKey:SourceEntityID"`
	TargetEntity *Entity `json:"target_entity,omitempty" gorm:"foreignKey:TargetEntityID"`
}

// =============================================================================
// VIEW MODELS
// =============================================================================

// View represents a UI view (list, detail, form, etc.)
type View struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TenantID  uuid.UUID `json:"tenant_id" gorm:"type:uuid;index"`
	EntityID  uuid.UUID `json:"entity_id" gorm:"type:uuid;index"`
	Code      string    `json:"code" gorm:"not null;size:50"`
	Name      string    `json:"name" gorm:"not null;size:100"`
	ViewType  string    `json:"view_type" gorm:"not null;size:30"`
	IsDefault bool      `json:"is_default" gorm:"default:false"`
	IsActive  bool      `json:"is_active" gorm:"default:true"`
	Settings  JSONB     `json:"settings" gorm:"type:jsonb;default:'{}'"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relations
	Entity   *Entity       `json:"entity,omitempty" gorm:"foreignKey:EntityID"`
	Sections []ViewSection `json:"sections,omitempty" gorm:"foreignKey:ViewID"`
	Fields   []ViewField   `json:"fields,omitempty" gorm:"foreignKey:ViewID"`
}

// ViewSection represents a section in a detail/form view
type ViewSection struct {
	ID               uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	ViewID           uuid.UUID `json:"view_id" gorm:"type:uuid;index"`
	Code             string    `json:"code" gorm:"not null;size:50"`
	Name             string    `json:"name" gorm:"not null;size:100"`
	Description      string    `json:"description"`
	DisplayOrder     int       `json:"display_order" gorm:"default:0"`
	Columns          int       `json:"columns" gorm:"default:2"`
	Collapsible      bool      `json:"collapsible" gorm:"default:false"`
	CollapsedDefault bool      `json:"collapsed_default" gorm:"default:false"`
	Settings         JSONB     `json:"settings" gorm:"type:jsonb;default:'{}'"`
	CreatedAt        time.Time `json:"created_at"`

	// Relations
	View   *View       `json:"view,omitempty" gorm:"foreignKey:ViewID"`
	Fields []ViewField `json:"fields,omitempty" gorm:"foreignKey:SectionID"`
}

// ViewField represents which fields appear in each view
type ViewField struct {
	ID               uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	ViewID           uuid.UUID  `json:"view_id" gorm:"type:uuid;index"`
	SectionID        *uuid.UUID `json:"section_id" gorm:"type:uuid"`
	FieldID          uuid.UUID  `json:"field_id" gorm:"type:uuid;index"`
	DisplayOrder     int        `json:"display_order" gorm:"default:0"`
	Label            string     `json:"label" gorm:"size:100"`
	Width            string     `json:"width" gorm:"size:20"`
	Component        string     `json:"component" gorm:"size:50"`
	ComponentOptions JSONB      `json:"component_options" gorm:"type:jsonb;default:'{}'"`
	IsVisible        bool       `json:"is_visible" gorm:"default:true"`
	IsReadonly       bool       `json:"is_readonly" gorm:"default:false"`
	Settings         JSONB      `json:"settings" gorm:"type:jsonb;default:'{}'"`
	CreatedAt        time.Time  `json:"created_at"`

	// Relations
	View    *View        `json:"view,omitempty" gorm:"foreignKey:ViewID"`
	Section *ViewSection `json:"section,omitempty" gorm:"foreignKey:SectionID"`
	Field   *Field       `json:"field,omitempty" gorm:"foreignKey:FieldID"`
}

// =============================================================================
// PERMISSION MODELS
// =============================================================================

// Role represents a user role
type Role struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TenantID    uuid.UUID `json:"tenant_id" gorm:"type:uuid;index"`
	Code        string    `json:"code" gorm:"not null;size:50"`
	Name        string    `json:"name" gorm:"not null;size:100"`
	Description string    `json:"description"`
	IsSystem    bool      `json:"is_system" gorm:"default:false"`
	IsActive    bool      `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Relations
	Tenant      *Tenant      `json:"tenant,omitempty" gorm:"foreignKey:TenantID"`
	Permissions []Permission `json:"permissions,omitempty" gorm:"foreignKey:RoleID"`
	Users       []User       `json:"users,omitempty" gorm:"many2many:user_roles;"`
}

// Permission represents permissions per entity, per role
type Permission struct {
	ID               uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TenantID         uuid.UUID `json:"tenant_id" gorm:"type:uuid;index"`
	RoleID           uuid.UUID `json:"role_id" gorm:"type:uuid;index"`
	EntityID         uuid.UUID `json:"entity_id" gorm:"type:uuid;index"`
	CanView          bool      `json:"can_view" gorm:"default:false"`
	CanCreate        bool      `json:"can_create" gorm:"default:false"`
	CanEdit          bool      `json:"can_edit" gorm:"default:false"`
	CanDelete        bool      `json:"can_delete" gorm:"default:false"`
	CanExport        bool      `json:"can_export" gorm:"default:false"`
	CanImport        bool      `json:"can_import" gorm:"default:false"`
	FieldPermissions JSONB     `json:"field_permissions" gorm:"type:jsonb;default:'{}'"`
	RowFilter        JSONB     `json:"row_filter" gorm:"type:jsonb"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`

	// Relations
	Role   *Role   `json:"role,omitempty" gorm:"foreignKey:RoleID"`
	Entity *Entity `json:"entity,omitempty" gorm:"foreignKey:EntityID"`
}

// =============================================================================
// ACTION & WORKFLOW MODELS
// =============================================================================

// Action represents a custom action on an entity
type Action struct {
	ID                  uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TenantID            uuid.UUID `json:"tenant_id" gorm:"type:uuid;index"`
	EntityID            uuid.UUID `json:"entity_id" gorm:"type:uuid;index"`
	Code                string    `json:"code" gorm:"not null;size:50"`
	Name                string    `json:"name" gorm:"not null;size:100"`
	Description         string    `json:"description"`
	ActionType          string    `json:"action_type" gorm:"not null;size:30"`
	Icon                string    `json:"icon" gorm:"size:50"`
	Color               string    `json:"color" gorm:"size:20"`
	ShowConditions      JSONB     `json:"show_conditions" gorm:"type:jsonb;default:'{}'"`
	HandlerType         string    `json:"handler_type" gorm:"not null;size:30"`
	HandlerConfig       JSONB     `json:"handler_config" gorm:"type:jsonb;default:'{}'"`
	RequiresConfirm     bool      `json:"requires_confirmation" gorm:"default:false"`
	ConfirmationMessage string    `json:"confirmation_message"`
	IsActive            bool      `json:"is_active" gorm:"default:true"`
	DisplayOrder        int       `json:"display_order" gorm:"default:0"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`

	// Relations
	Entity *Entity `json:"entity,omitempty" gorm:"foreignKey:EntityID"`
}

// Workflow represents an automated process
type Workflow struct {
	ID                uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TenantID          uuid.UUID  `json:"tenant_id" gorm:"type:uuid;index"`
	EntityID          *uuid.UUID `json:"entity_id" gorm:"type:uuid"`
	Code              string     `json:"code" gorm:"not null;size:50"`
	Name              string     `json:"name" gorm:"not null;size:100"`
	Description       string     `json:"description"`
	TriggerType       string     `json:"trigger_type" gorm:"not null;size:30"`
	TriggerConditions JSONB      `json:"trigger_conditions" gorm:"type:jsonb;default:'{}'"`
	ScheduleCron      string     `json:"schedule_cron" gorm:"size:100"`
	IsActive          bool       `json:"is_active" gorm:"default:true"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`

	// Relations
	Entity *Entity        `json:"entity,omitempty" gorm:"foreignKey:EntityID"`
	Steps  []WorkflowStep `json:"steps,omitempty" gorm:"foreignKey:WorkflowID"`
}

// WorkflowStep represents a step in a workflow
type WorkflowStep struct {
	ID              uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	WorkflowID      uuid.UUID  `json:"workflow_id" gorm:"type:uuid;index"`
	StepOrder       int        `json:"step_order" gorm:"not null"`
	Name            string     `json:"name" gorm:"not null;size:100"`
	StepType        string     `json:"step_type" gorm:"not null;size:30"`
	Config          JSONB      `json:"config" gorm:"type:jsonb;not null"`
	OnSuccessStepID *uuid.UUID `json:"on_success_step_id" gorm:"type:uuid"`
	OnFailureStepID *uuid.UUID `json:"on_failure_step_id" gorm:"type:uuid"`
	CreatedAt       time.Time  `json:"created_at"`

	// Relations
	Workflow      *Workflow     `json:"workflow,omitempty" gorm:"foreignKey:WorkflowID"`
	OnSuccessStep *WorkflowStep `json:"on_success_step,omitempty" gorm:"foreignKey:OnSuccessStepID"`
	OnFailureStep *WorkflowStep `json:"on_failure_step,omitempty" gorm:"foreignKey:OnFailureStepID"`
}

// =============================================================================
// UI MODELS
// =============================================================================

// Menu represents a navigation menu
type Menu struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TenantID  uuid.UUID `json:"tenant_id" gorm:"type:uuid;index"`
	Code      string    `json:"code" gorm:"not null;size:50"`
	Name      string    `json:"name" gorm:"not null;size:100"`
	IsActive  bool      `json:"is_active" gorm:"default:true"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relations
	Tenant *Tenant    `json:"tenant,omitempty" gorm:"foreignKey:TenantID"`
	Items  []MenuItem `json:"items,omitempty" gorm:"foreignKey:MenuID"`
}

// MenuItem represents an item in a menu
type MenuItem struct {
	ID                 uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	MenuID             uuid.UUID  `json:"menu_id" gorm:"type:uuid;index"`
	ParentID           *uuid.UUID `json:"parent_id" gorm:"type:uuid"`
	EntityID           *uuid.UUID `json:"entity_id" gorm:"type:uuid"`
	ModuleID           *uuid.UUID `json:"module_id" gorm:"type:uuid"`
	CustomURL          string     `json:"custom_url" gorm:"size:255"`
	Label              string     `json:"label" gorm:"not null;size:100"`
	Icon               string     `json:"icon" gorm:"size:50"`
	DisplayOrder       int        `json:"display_order" gorm:"default:0"`
	IsActive           bool       `json:"is_active" gorm:"default:true"`
	RequiredPermission string     `json:"required_permission" gorm:"size:50"`
	Settings           JSONB      `json:"settings" gorm:"type:jsonb;default:'{}'"`
	CreatedAt          time.Time  `json:"created_at"`

	// Relations
	Menu     *Menu      `json:"menu,omitempty" gorm:"foreignKey:MenuID"`
	Parent   *MenuItem  `json:"parent,omitempty" gorm:"foreignKey:ParentID"`
	Children []MenuItem `json:"children,omitempty" gorm:"foreignKey:ParentID"`
	Entity   *Entity    `json:"entity,omitempty" gorm:"foreignKey:EntityID"`
	Module   *Module    `json:"module,omitempty" gorm:"foreignKey:ModuleID"`
}

// =============================================================================
// AUDIT MODEL
// =============================================================================

// AuditLog represents an audit trail entry
type AuditLog struct {
	ID            uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TenantID      uuid.UUID  `json:"tenant_id" gorm:"type:uuid;index"`
	UserID        *uuid.UUID `json:"user_id" gorm:"type:uuid"`
	EntityID      *uuid.UUID `json:"entity_id" gorm:"type:uuid"`
	EntityCode    string     `json:"entity_code" gorm:"size:50;index"`
	RecordID      *uuid.UUID `json:"record_id" gorm:"type:uuid"`
	Action        string     `json:"action" gorm:"not null;size:30"`
	OldValues     JSONB      `json:"old_values" gorm:"type:jsonb"`
	NewValues     JSONB      `json:"new_values" gorm:"type:jsonb"`
	ChangedFields []string   `json:"changed_fields" gorm:"type:text[]"`
	IPAddress     string     `json:"ip_address" gorm:"size:45"`
	UserAgent     string     `json:"user_agent"`
	CreatedAt     time.Time  `json:"created_at" gorm:"index"`

	// Relations
	User   *User   `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Entity *Entity `json:"entity,omitempty" gorm:"foreignKey:EntityID"`
}
