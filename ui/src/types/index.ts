// Core Types for Genesis No-Code Platform

export interface User {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
  role: string;
  tenant_id: string;
}

export interface Tenant {
  id: string;
  name: string;
  code: string;
}

export interface Module {
  id: string;
  name: string;
  code: string;
  description: string;
  icon: string;
  sort_order: number;
  is_active: boolean;
}

export interface Entity {
  id: string;
  module_id: string;
  name: string;
  code: string;
  table_name: string;
  description: string;
  icon: string;
  sort_order: number;
  is_active: boolean;
}

export interface Field {
  id: string;
  entity_id: string;
  name: string;
  code: string;
  field_type: string;
  field_type_id: string;
  description?: string;
  is_required: boolean;
  is_unique: boolean;
  is_searchable: boolean;
  is_filterable: boolean;
  show_in_list: boolean;
  show_in_form: boolean;
  sort_order: number;
  default_value?: string;
  validation_rules?: string;
  options?: FieldOption[];
  reference_entity?: string;
}

export interface FieldOption {
  label: string;
  value: string;
}

export interface FieldType {
  id: string;
  code: string;
  name: string;
  component: string;
  has_options: boolean;
}

export interface Schema {
  tenant: Tenant;
  modules: Module[];
  entities: Entity[];
}

export interface EntitySchema {
  entity: Entity;
  fields: Field[];
}

// Article types (from Aethra)
export interface Article {
  id: string;
  title: string;
  slug: string;
  excerpt?: string;
  content?: string;
  featured_image?: string;
  category?: string;
  category_id?: string;
  status: 'draft' | 'published' | 'archived';
  tags?: string;
  meta_title?: string;
  meta_description?: string;
  event_date?: string;
  event_location?: string;
  place_slug?: string;
  website_id?: string;
  website_name?: string;
  views?: number;
  author?: string;
  author_id?: string;
  created_at: string;
  updated_at: string;
  published_at?: string;
}

export interface Website {
  id: string;
  name: string;
  domain: string;
}

export interface Category {
  id: string;
  name: string;
  slug: string;
}

// UI Component stored in database
export interface UIComponent {
  id: string;
  code: string;
  name: string;
  category: string;
  code_jsx: string;
  props_schema: string;
}

// API Response types
export interface ApiResponse<T> {
  data?: T;
  error?: string;
  message?: string;
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  limit: number;
  offset: number;
}
