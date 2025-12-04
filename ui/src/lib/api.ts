// API Client for Genesis
import type { Schema, EntitySchema, Article, Website, Category, User, Tenant } from '../types';

const API_BASE = '';

class ApiClient {
  private token: string | null = null;
  private tenantId: string | null = null;

  setToken(token: string | null) {
    this.token = token;
    if (token) {
      localStorage.setItem('token', token);
    } else {
      localStorage.removeItem('token');
    }
  }

  setTenantId(tenantId: string | null) {
    this.tenantId = tenantId;
    if (tenantId) {
      localStorage.setItem('tenant_id', tenantId);
    } else {
      localStorage.removeItem('tenant_id');
    }
  }

  getToken(): string | null {
    if (!this.token) {
      this.token = localStorage.getItem('token');
    }
    return this.token;
  }

  getTenantId(): string | null {
    if (!this.tenantId) {
      this.tenantId = localStorage.getItem('tenant_id');
    }
    return this.tenantId;
  }

  private async fetch<T>(method: string, endpoint: string, body?: unknown): Promise<T> {
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
    };

    const token = this.getToken();
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    const tenantId = this.getTenantId();
    if (tenantId) {
      headers['X-Tenant-ID'] = tenantId;
    }

    const response = await fetch(`${API_BASE}${endpoint}`, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Request failed' }));
      throw new Error(error.error || error.message || 'Request failed');
    }

    return response.json();
  }

  // Auth
  async login(email: string, password: string, tenantId: string): Promise<{ user: User; tokens: { access_token: string } }> {
    const result = await this.fetch<{ user: User; tokens: { access_token: string } }>('POST', '/auth/login', {
      email,
      password,
      tenant_id: tenantId,
    });
    this.setToken(result.tokens.access_token);
    this.setTenantId(tenantId);
    return result;
  }

  async logout(): Promise<void> {
    try {
      await this.fetch('POST', '/auth/logout');
    } finally {
      this.setToken(null);
    }
  }

  async getMe(): Promise<User> {
    return this.fetch<User>('GET', '/auth/me');
  }

  // Schema
  async getSchema(): Promise<Schema> {
    return this.fetch<Schema>('GET', '/api/schema');
  }

  async getEntitySchema(entityCode: string): Promise<EntitySchema> {
    return this.fetch<EntitySchema>('GET', `/api/schema/${entityCode}`);
  }

  // Dynamic Data
  async getData(entityCode: string, params?: Record<string, string>): Promise<{ data: Record<string, unknown>[]; total: number }> {
    const query = params ? '?' + new URLSearchParams(params).toString() : '';
    return this.fetch('GET', `/api/data/${entityCode}${query}`);
  }

  async getRecord(entityCode: string, id: string): Promise<Record<string, unknown>> {
    return this.fetch('GET', `/api/data/${entityCode}/${id}`);
  }

  async createRecord(entityCode: string, data: Record<string, unknown>): Promise<{ id: string }> {
    return this.fetch('POST', `/api/data/${entityCode}`, data);
  }

  async updateRecord(entityCode: string, id: string, data: Record<string, unknown>): Promise<void> {
    return this.fetch('PUT', `/api/data/${entityCode}/${id}`, data);
  }

  async deleteRecord(entityCode: string, id: string): Promise<void> {
    return this.fetch('DELETE', `/api/data/${entityCode}/${id}`);
  }

  // Admin - Modules
  async getModules(): Promise<{ modules: { id: string; name: string; code: string; icon: string; is_active: boolean }[] }> {
    return this.fetch('GET', '/admin/modules');
  }

  async createModule(data: { name: string; code: string; description?: string; icon?: string }): Promise<{ id: string }> {
    return this.fetch('POST', '/admin/modules', data);
  }

  async updateModule(id: string, data: { name: string; code: string; description?: string; icon?: string }): Promise<void> {
    return this.fetch('PUT', `/admin/modules/${id}`, data);
  }

  async deleteModule(id: string): Promise<void> {
    return this.fetch('DELETE', `/admin/modules/${id}`);
  }

  // Admin - Entities
  async getEntities(): Promise<{ entities: { id: string; module_id: string; name: string; code: string; table_name: string }[] }> {
    return this.fetch('GET', '/admin/entities');
  }

  async createEntity(data: { module_id: string; name: string; code: string; table_name: string; description?: string }): Promise<{ id: string }> {
    return this.fetch('POST', '/admin/entities', data);
  }

  async updateEntity(id: string, data: { name: string; code: string; table_name: string; description?: string }): Promise<void> {
    return this.fetch('PUT', `/admin/entities/${id}`, data);
  }

  async deleteEntity(id: string): Promise<void> {
    return this.fetch('DELETE', `/admin/entities/${id}`);
  }

  // Admin - Fields
  async getFields(entityId?: string): Promise<{ fields: { id: string; entity_id: string; name: string; code: string; field_type: string }[] }> {
    const query = entityId ? `?entity_id=${entityId}` : '';
    return this.fetch('GET', `/admin/fields${query}`);
  }

  async createField(data: { entity_id: string; name: string; code: string; field_type: string; is_required?: boolean }): Promise<{ id: string }> {
    return this.fetch('POST', '/admin/fields', data);
  }

  async updateField(id: string, data: { name: string; code: string; field_type: string; is_required?: boolean }): Promise<void> {
    return this.fetch('PUT', `/admin/fields/${id}`, data);
  }

  async deleteField(id: string): Promise<void> {
    return this.fetch('DELETE', `/admin/fields/${id}`);
  }

  async getFieldTypes(): Promise<{ field_types: { code: string; name: string; component: string }[] }> {
    return this.fetch('GET', '/admin/field-types');
  }

  // Articles (Aethra)
  async getAdminArticles(params?: Record<string, string>): Promise<{ articles: Article[]; total: number }> {
    const query = params ? '?' + new URLSearchParams(params).toString() : '';
    return this.fetch('GET', `/api/admin/articles${query}`);
  }

  async getArticle(id: string): Promise<Article> {
    return this.fetch('GET', `/api/admin/articles/${id}`);
  }

  async createArticle(data: Partial<Article>): Promise<{ id: string }> {
    return this.fetch('POST', '/api/admin/articles', data);
  }

  async updateArticle(id: string, data: Partial<Article>): Promise<void> {
    return this.fetch('PUT', `/api/admin/articles/${id}`, data);
  }

  async deleteArticle(id: string): Promise<void> {
    return this.fetch('DELETE', `/api/admin/articles/${id}`);
  }

  async publishArticle(id: string): Promise<void> {
    return this.fetch('POST', `/api/admin/articles/${id}/publish`);
  }

  async unpublishArticle(id: string): Promise<void> {
    return this.fetch('POST', `/api/admin/articles/${id}/unpublish`);
  }

  async getWebsites(): Promise<{ websites: Website[] }> {
    return this.fetch('GET', '/api/admin/websites');
  }

  async getCategoriesList(): Promise<{ categories: Category[] }> {
    return this.fetch('GET', '/api/admin/categories');
  }

  // Tenants (for setup)
  async getTenants(): Promise<{ tenants: Tenant[] }> {
    return this.fetch('GET', '/admin/tenants');
  }
}

export const api = new ApiClient();
export default api;
