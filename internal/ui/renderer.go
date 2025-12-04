// Package ui - UI Renderer Engine
// Renders React components from database
package ui

import (
	"encoding/json"
	"fmt"
	"html/template"
	"strings"

	"github.com/aethra/genesis/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Renderer renders dynamic UI from database
type Renderer struct {
	db *gorm.DB
}

// NewRenderer creates a new UI renderer
func NewRenderer(db *gorm.DB) *Renderer {
	return &Renderer{db: db}
}

// PageData contains all data needed to render a page
type PageData struct {
	Page       *models.UIPage       `json:"page"`
	Layout     *models.UILayout     `json:"layout"`
	Theme      *models.UITheme      `json:"theme"`
	Components []models.UIComponent `json:"components"`
	Tenant     *models.Tenant       `json:"tenant"`
	User       map[string]any       `json:"user"`
	Modules    []models.Module      `json:"modules"`
	Entities   []models.Entity      `json:"entities"`
	Entity     *models.Entity       `json:"entity,omitempty"`
	Fields     []models.Field       `json:"fields,omitempty"`
	Data       any                  `json:"data,omitempty"`
}

// RenderPage renders a complete HTML page with React components
func (r *Renderer) RenderPage(tenantID uuid.UUID, route string, user map[string]any) (string, error) {
	return r.RenderPageWithBasePath(tenantID, route, user, "")
}

// RenderPageWithBasePath renders a complete HTML page with React components and base path support
func (r *Renderer) RenderPageWithBasePath(tenantID uuid.UUID, route string, user map[string]any, basePath string) (string, error) {
	pageData, err := r.GetPageData(tenantID, route)
	if err != nil {
		return "", err
	}
	pageData.User = user

	return r.GenerateHTMLWithBasePath(pageData, basePath)
}

// GetPageData loads all data needed for a page
func (r *Renderer) GetPageData(tenantID uuid.UUID, route string) (*PageData, error) {
	data := &PageData{}

	// Get tenant
	var tenant models.Tenant
	if err := r.db.First(&tenant, "id = ?", tenantID).Error; err != nil {
		return nil, fmt.Errorf("tenant not found: %w", err)
	}
	data.Tenant = &tenant

	// Get page by route
	var page models.UIPage
	if err := r.db.Where("route = ? AND (tenant_id = ? OR tenant_id IS NULL)", route, tenantID).
		Order("tenant_id DESC NULLS LAST").
		First(&page).Error; err != nil {
		// Try dynamic routes
		page, err = r.matchDynamicRoute(tenantID, route)
		if err != nil {
			// Default dashboard for /app
			if route == "/app" {
				page = models.UIPage{
					Code:     "dashboard",
					Name:     "Dashboard",
					Route:    "/app",
					PageType: "static",
					Title:    "Dashboard",
				}
			} else {
				return nil, fmt.Errorf("page not found: %w", err)
			}
		}
	}
	data.Page = &page

	// Get layout
	if page.LayoutID != nil {
		var layout models.UILayout
		if err := r.db.First(&layout, "id = ?", page.LayoutID).Error; err == nil {
			data.Layout = &layout
		}
	}
	if data.Layout == nil {
		// Get default layout
		var layout models.UILayout
		if err := r.db.Where("is_default = ? AND (tenant_id = ? OR tenant_id IS NULL)", true, tenantID).
			Order("tenant_id DESC NULLS LAST").
			First(&layout).Error; err == nil {
			data.Layout = &layout
		}
	}

	// Get theme
	var theme models.UITheme
	if err := r.db.Where("is_default = ? AND (tenant_id = ? OR tenant_id IS NULL)", true, tenantID).
		Order("tenant_id DESC NULLS LAST").
		First(&theme).Error; err == nil {
		data.Theme = &theme
	}

	// Get all components
	var components []models.UIComponent
	r.db.Where("is_active = ? AND (tenant_id = ? OR tenant_id IS NULL)", true, tenantID).
		Find(&components)
	data.Components = components

	// Get modules and entities for navigation
	var modules []models.Module
	r.db.Where("tenant_id = ? AND is_active = ?", tenantID, true).
		Order("display_order").Find(&modules)
	data.Modules = modules

	var entities []models.Entity
	r.db.Where("tenant_id = ? AND is_active = ?", tenantID, true).
		Order("display_order").Find(&entities)
	data.Entities = entities

	// If entity page, load entity and fields
	if page.EntityID != nil {
		var entity models.Entity
		if err := r.db.First(&entity, "id = ?", page.EntityID).Error; err == nil {
			data.Entity = &entity

			var fields []models.Field
			r.db.Where("entity_id = ? AND is_active = ?", entity.ID, true).
				Order("display_order").Find(&fields)
			data.Fields = fields
		}
	}

	return data, nil
}

// matchDynamicRoute matches routes like /app/:module/:entity
func (r *Renderer) matchDynamicRoute(tenantID uuid.UUID, route string) (models.UIPage, error) {
	parts := strings.Split(strings.Trim(route, "/"), "/")

	// Match /app/articles - Special Aethra articles management
	if len(parts) >= 2 && parts[0] == "app" && parts[1] == "articles" {
		if len(parts) >= 3 {
			if parts[2] == "new" {
				return models.UIPage{
					Code:     "articles_form",
					Name:     "Create Article",
					Route:    route,
					PageType: "articles_form",
					Title:    "Create Article",
				}, nil
			}
			// Edit view (article ID)
			return models.UIPage{
				Code:     "articles_edit",
				Name:     "Edit Article",
				Route:    route,
				PageType: "articles_edit",
				Title:    "Edit Article",
			}, nil
		}
		// List view
		return models.UIPage{
			Code:     "articles_list",
			Name:     "Articles",
			Route:    route,
			PageType: "articles_list",
			Title:    "Articles",
		}, nil
	}

	// Match /app/:module/:entity
	if len(parts) >= 3 && parts[0] == "app" {
		moduleCode := parts[1]
		entityCode := parts[2]

		// Find entity
		var entity models.Entity
		if err := r.db.Joins("JOIN modules ON modules.id = entities.module_id").
			Where("entities.tenant_id = ? AND entities.code = ? AND modules.code = ?",
				tenantID, entityCode, moduleCode).
			First(&entity).Error; err != nil {
			return models.UIPage{}, err
		}

		// Check if detail or form view
		if len(parts) >= 4 {
			if parts[3] == "new" {
				return models.UIPage{
					Code:     entityCode + "_form",
					Name:     "Create " + entity.Name,
					Route:    route,
					PageType: "entity_form",
					EntityID: &entity.ID,
					Title:    "Create " + entity.Name,
				}, nil
			}
			// Detail view
			return models.UIPage{
				Code:     entityCode + "_detail",
				Name:     entity.Name + " Detail",
				Route:    route,
				PageType: "entity_detail",
				EntityID: &entity.ID,
				Title:    entity.Name,
			}, nil
		}

		// List view
		return models.UIPage{
			Code:     entityCode + "_list",
			Name:     entity.NamePlural,
			Route:    route,
			PageType: "entity_list",
			EntityID: &entity.ID,
			Title:    entity.NamePlural,
		}, nil
	}

	return models.UIPage{}, fmt.Errorf("no matching route")
}

// GenerateHTML generates the complete HTML page
func (r *Renderer) GenerateHTML(data *PageData) (string, error) {
	return r.GenerateHTMLWithBasePath(data, "")
}

// GenerateHTMLWithBasePath generates the complete HTML page with base path support
func (r *Renderer) GenerateHTMLWithBasePath(data *PageData, basePath string) (string, error) {
	// Build component code
	componentCode := r.buildComponentCode(data.Components)

	// Build CSS from theme
	css := r.buildCSS(data.Theme)

	// Serialize data for JavaScript
	jsonData, _ := json.Marshal(data)

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s - Genesis</title>
    <style>%s</style>
    <script src="https://unpkg.com/react@18/umd/react.development.js" crossorigin></script>
    <script src="https://unpkg.com/react-dom@18/umd/react-dom.development.js" crossorigin></script>
    <script src="https://unpkg.com/@babel/standalone/babel.min.js"></script>
</head>
<body>
    <div id="root"></div>

    <script type="text/babel" data-presets="react">
        // Page data from server
        const PAGE_DATA = %s;
        const BASE_PATH = '%s';

        // API helper
        const api = {
            token: localStorage.getItem('token'),
            tenantId: PAGE_DATA.tenant?.id,
            basePath: BASE_PATH,

            async fetch(method, endpoint, body = null) {
                const options = {
                    method,
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': 'Bearer ' + this.token,
                        'X-Tenant-ID': this.tenantId
                    }
                };
                if (body) options.body = JSON.stringify(body);
                const res = await fetch(this.basePath + endpoint, options);
                const data = await res.json();
                if (!res.ok) throw new Error(data.error || 'Request failed');
                return data;
            },

            get: (endpoint) => api.fetch('GET', endpoint),
            post: (endpoint, body) => api.fetch('POST', endpoint, body),
            put: (endpoint, body) => api.fetch('PUT', endpoint, body),
            delete: (endpoint) => api.fetch('DELETE', endpoint)
        };

        // Components from database
        %s

        // Articles Management Component
        function ArticlesManager() {
            const [articles, setArticles] = React.useState([]);
            const [loading, setLoading] = React.useState(true);
            const [websites, setWebsites] = React.useState([]);
            const [categories, setCategories] = React.useState([]);
            const [filters, setFilters] = React.useState({ status: '', website_id: '', search: '' });
            const [modalOpen, setModalOpen] = React.useState(false);
            const [editingArticle, setEditingArticle] = React.useState(null);
            const [formValues, setFormValues] = React.useState({});
            const [total, setTotal] = React.useState(0);

            React.useEffect(() => {
                loadArticles();
                loadWebsites();
                loadCategories();
            }, [filters]);

            const loadArticles = async () => {
                setLoading(true);
                try {
                    const params = new URLSearchParams();
                    if (filters.status) params.append('status', filters.status);
                    if (filters.website_id) params.append('website_id', filters.website_id);
                    if (filters.search) params.append('search', filters.search);
                    const result = await api.get('/api/admin/articles?' + params.toString());
                    setArticles(result.articles || []);
                    setTotal(result.total || 0);
                } catch (err) {
                    console.error('Failed to load articles:', err);
                }
                setLoading(false);
            };

            const loadWebsites = async () => {
                try {
                    const result = await api.get('/api/admin/websites');
                    setWebsites(result.websites || []);
                } catch (err) { console.error(err); }
            };

            const loadCategories = async () => {
                try {
                    const result = await api.get('/api/admin/categories');
                    setCategories(result.categories || []);
                } catch (err) { console.error(err); }
            };

            const handleCreate = () => {
                setEditingArticle(null);
                setFormValues({ status: 'draft', website_id: 4 });
                setModalOpen(true);
            };

            const handleEdit = async (article) => {
                try {
                    const result = await api.get('/api/admin/articles/' + article.id);
                    setEditingArticle(result);
                    setFormValues(result);
                    setModalOpen(true);
                } catch (err) { alert(err.message); }
            };

            const handleDelete = async (article) => {
                if (!confirm('Delete article "' + article.title + '"?')) return;
                try {
                    await api.delete('/api/admin/articles/' + article.id);
                    loadArticles();
                } catch (err) { alert(err.message); }
            };

            const handlePublish = async (article) => {
                try {
                    await api.post('/api/admin/articles/' + article.id + '/publish');
                    loadArticles();
                } catch (err) { alert(err.message); }
            };

            const handleUnpublish = async (article) => {
                try {
                    await api.post('/api/admin/articles/' + article.id + '/unpublish');
                    loadArticles();
                } catch (err) { alert(err.message); }
            };

            const handleSubmit = async () => {
                setLoading(true);
                try {
                    if (editingArticle) {
                        await api.put('/api/admin/articles/' + editingArticle.id, formValues);
                    } else {
                        await api.post('/api/admin/articles', formValues);
                    }
                    setModalOpen(false);
                    loadArticles();
                } catch (err) { alert(err.message); }
                setLoading(false);
            };

            const statusColors = {
                published: 'status-published',
                draft: 'status-draft',
                archived: 'status-archived'
            };

            return (
                <div className="articles-manager">
                    <div className="articles-toolbar">
                        <div className="filters">
                            <select
                                className="filter-select"
                                value={filters.website_id}
                                onChange={e => setFilters({...filters, website_id: e.target.value})}
                            >
                                <option value="">All Websites</option>
                                {websites.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
                            </select>
                            <select
                                className="filter-select"
                                value={filters.status}
                                onChange={e => setFilters({...filters, status: e.target.value})}
                            >
                                <option value="">All Status</option>
                                <option value="published">Published</option>
                                <option value="draft">Draft</option>
                            </select>
                            <input
                                type="text"
                                className="filter-input"
                                placeholder="Search articles..."
                                value={filters.search}
                                onChange={e => setFilters({...filters, search: e.target.value})}
                            />
                        </div>
                        <div className="toolbar-stats">
                            <span className="stat-badge">{total} articles</span>
                        </div>
                    </div>

                    {loading ? (
                        <div className="loading-state">Loading articles...</div>
                    ) : (
                        <div className="articles-grid">
                            {articles.map(article => (
                                <div key={article.id} className="article-card">
                                    <div className="article-image">
                                        {article.featured_image ? (
                                            <img src={article.featured_image} alt="" />
                                        ) : (
                                            <div className="no-image">No Image</div>
                                        )}
                                    </div>
                                    <div className="article-content">
                                        <div className="article-meta">
                                            <span className={'article-status ' + statusColors[article.status]}>{article.status}</span>
                                            <span className="article-website">{article.website_name}</span>
                                        </div>
                                        <h3 className="article-title">{article.title}</h3>
                                        <p className="article-summary">{article.summary?.substring(0, 120)}...</p>
                                        <div className="article-footer">
                                            <span className="article-views">{article.views} views</span>
                                            <span className="article-date">{new Date(article.updated_at).toLocaleDateString()}</span>
                                        </div>
                                        <div className="article-actions">
                                            <button className="btn-icon" onClick={() => handleEdit(article)} title="Edit">Edit</button>
                                            {article.status === 'draft' ? (
                                                <button className="btn-icon btn-publish" onClick={() => handlePublish(article)} title="Publish">Publish</button>
                                            ) : (
                                                <button className="btn-icon btn-unpublish" onClick={() => handleUnpublish(article)} title="Unpublish">Unpublish</button>
                                            )}
                                            <button className="btn-icon btn-delete" onClick={() => handleDelete(article)} title="Delete">Delete</button>
                                        </div>
                                    </div>
                                </div>
                            ))}
                        </div>
                    )}

                    <Modal isOpen={modalOpen} onClose={() => setModalOpen(false)} title={editingArticle ? 'Edit Article' : 'Create Article'}>
                        <div className="article-form">
                            <div className="form-row">
                                <label>Title *</label>
                                <input type="text" value={formValues.title || ''} onChange={e => setFormValues({...formValues, title: e.target.value})} />
                            </div>
                            <div className="form-row">
                                <label>Slug</label>
                                <input type="text" value={formValues.slug || ''} onChange={e => setFormValues({...formValues, slug: e.target.value})} placeholder="auto-generated if empty" />
                            </div>
                            <div className="form-row-half">
                                <div className="form-row">
                                    <label>Website</label>
                                    <select value={formValues.website_id || ''} onChange={e => setFormValues({...formValues, website_id: parseInt(e.target.value)})}>
                                        {websites.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
                                    </select>
                                </div>
                                <div className="form-row">
                                    <label>Category</label>
                                    <select value={formValues.category_id || ''} onChange={e => setFormValues({...formValues, category_id: parseInt(e.target.value)})}>
                                        <option value="">Select category</option>
                                        {categories.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
                                    </select>
                                </div>
                            </div>
                            <div className="form-row">
                                <label>Summary</label>
                                <textarea rows="3" value={formValues.summary || ''} onChange={e => setFormValues({...formValues, summary: e.target.value})} />
                            </div>
                            <div className="form-row">
                                <label>Content</label>
                                <textarea rows="10" value={formValues.content || ''} onChange={e => setFormValues({...formValues, content: e.target.value})} />
                            </div>
                            <div className="form-row">
                                <label>Featured Image URL</label>
                                <input type="text" value={formValues.featured_image || ''} onChange={e => setFormValues({...formValues, featured_image: e.target.value})} />
                            </div>
                            <div className="form-row-half">
                                <div className="form-row">
                                    <label>Meta Title</label>
                                    <input type="text" value={formValues.meta_title || ''} onChange={e => setFormValues({...formValues, meta_title: e.target.value})} />
                                </div>
                                <div className="form-row">
                                    <label>Status</label>
                                    <select value={formValues.status || 'draft'} onChange={e => setFormValues({...formValues, status: e.target.value})}>
                                        <option value="draft">Draft</option>
                                        <option value="published">Published</option>
                                    </select>
                                </div>
                            </div>
                            <div className="form-row">
                                <label>Meta Description</label>
                                <textarea rows="2" value={formValues.meta_description || ''} onChange={e => setFormValues({...formValues, meta_description: e.target.value})} />
                            </div>
                            <div className="form-actions">
                                <button className="btn btn-secondary" onClick={() => setModalOpen(false)}>Cancel</button>
                                <button className="btn btn-primary" onClick={handleSubmit} disabled={loading}>
                                    {loading ? 'Saving...' : (editingArticle ? 'Update Article' : 'Create Article')}
                                </button>
                            </div>
                        </div>
                    </Modal>
                </div>
            );
        }

        // App Component
        function App() {
            const [loading, setLoading] = React.useState(false);
            const [data, setData] = React.useState(PAGE_DATA.data || []);
            const [formValues, setFormValues] = React.useState({});
            const [modalOpen, setModalOpen] = React.useState(false);
            const [editingRecord, setEditingRecord] = React.useState(null);

            const page = PAGE_DATA.page;
            const entity = PAGE_DATA.entity;
            const fields = PAGE_DATA.fields || [];
            const modules = PAGE_DATA.modules || [];
            const entities = PAGE_DATA.entities || [];

            // Check if this is articles page
            const isArticlesPage = page.page_type?.startsWith('articles_');

            // Load data for entity pages
            React.useEffect(() => {
                if (entity && page.page_type === 'entity_list') {
                    loadData();
                }
            }, []);

            const loadData = async () => {
                setLoading(true);
                try {
                    const result = await api.get('/api/data/' + entity.code);
                    setData(result.data || []);
                } catch (err) {
                    console.error('Failed to load data:', err);
                }
                setLoading(false);
            };

            const handleCreate = () => {
                setEditingRecord(null);
                setFormValues({});
                setModalOpen(true);
            };

            const handleEdit = (record) => {
                setEditingRecord(record);
                setFormValues(record);
                setModalOpen(true);
            };

            const handleDelete = async (record) => {
                if (!confirm('Are you sure?')) return;
                try {
                    await api.delete('/api/data/' + entity.code + '/' + record.id);
                    loadData();
                } catch (err) {
                    alert(err.message);
                }
            };

            const handleSubmit = async (values) => {
                setLoading(true);
                try {
                    if (editingRecord) {
                        await api.put('/api/data/' + entity.code + '/' + editingRecord.id, values);
                    } else {
                        await api.post('/api/data/' + entity.code, values);
                    }
                    setModalOpen(false);
                    loadData();
                } catch (err) {
                    alert(err.message);
                }
                setLoading(false);
            };

            const columns = fields.filter(f => f.in_list !== false);

            return (
                <div className="app">
                    <aside className="sidebar">
                        <div className="sidebar-header">
                            <div className="logo">Genesis</div>
                            <div className="tenant-name">{PAGE_DATA.tenant?.name}</div>
                        </div>
                        <Navigation modules={modules} entities={entities} currentPath={window.location.pathname} basePath={BASE_PATH} showArticles={true} />
                        <div className="sidebar-footer">
                            <div className="user-info">
                                <div className="user-avatar">{(PAGE_DATA.user?.first_name || 'U')[0]}</div>
                                <div className="user-details">
                                    <div className="user-name">{PAGE_DATA.user?.first_name} {PAGE_DATA.user?.last_name}</div>
                                    <div className="user-role">User</div>
                                </div>
                            </div>
                        </div>
                    </aside>
                    <main className="main">
                        <header className="header">
                            <h1 className="page-title">{page.title || page.name}</h1>
                            <div className="header-actions">
                                {isArticlesPage && <Button onClick={() => window.dispatchEvent(new CustomEvent('createArticle'))}>+ New Article</Button>}
                                {entity && !isArticlesPage && <Button onClick={handleCreate}>+ Create {entity.name}</Button>}
                            </div>
                        </header>
                        <div className="content">
                            {isArticlesPage && <ArticlesManager />}
                            {page.page_type === 'entity_list' && entity && (
                                <DataGrid
                                    entity={entity}
                                    columns={columns}
                                    data={data}
                                    loading={loading}
                                    onEdit={handleEdit}
                                    onDelete={handleDelete}
                                    onCreate={handleCreate}
                                />
                            )}
                            {page.page_type === 'static' && (
                                <div className="stats-grid">
                                    <StatCard label="Modules" value={modules.length} icon="📦" />
                                    <StatCard label="Entities" value={entities.length} icon="🗃️" />
                                    <StatCard label="Fields" value={fields.length} icon="📝" />
                                </div>
                            )}
                        </div>
                    </main>

                    <Modal isOpen={modalOpen} onClose={() => setModalOpen(false)} title={editingRecord ? 'Edit ' + entity?.name : 'Create ' + entity?.name}>
                        <DynamicForm
                            entity={entity}
                            fields={fields}
                            values={formValues}
                            onChange={setFormValues}
                            onSubmit={handleSubmit}
                            onCancel={() => setModalOpen(false)}
                            loading={loading}
                            mode={editingRecord ? 'edit' : 'create'}
                        />
                    </Modal>
                </div>
            );
        }

        // Render
        const root = ReactDOM.createRoot(document.getElementById('root'));
        root.render(<App />);
    </script>
</body>
</html>`,
		template.HTMLEscapeString(data.Page.Title),
		css,
		string(jsonData),
		basePath,
		componentCode,
	)

	return html, nil
}

// buildComponentCode builds JavaScript from stored components
func (r *Renderer) buildComponentCode(components []models.UIComponent) string {
	var sb strings.Builder
	for _, c := range components {
		sb.WriteString(c.CodeJSX)
		sb.WriteString("\n\n")
	}
	return sb.String()
}

// buildCSS builds CSS from theme
func (r *Renderer) buildCSS(theme *models.UITheme) string {
	// Aethra-style clean light CSS (matching intranet panel)
	css := `
        @import url('https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap');

        :root {
            --primary: #4f46e5;
            --primary-light: #6366f1;
            --primary-dark: #4338ca;
            --success: #10b981;
            --warning: #f59e0b;
            --danger: #ef4444;
            --bg-body: #f9fafb;
            --bg-white: #ffffff;
            --bg-gray: #f3f4f6;
            --border: #e5e7eb;
            --border-light: #f3f4f6;
            --text: #111827;
            --text-secondary: #374151;
            --text-muted: #6b7280;
            --text-light: #9ca3af;
            --shadow-sm: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
            --shadow: 0 1px 3px 0 rgba(0, 0, 0, 0.1), 0 1px 2px -1px rgba(0, 0, 0, 0.1);
            --shadow-lg: 0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -4px rgba(0, 0, 0, 0.1);
        }

        * { box-sizing: border-box; margin: 0; padding: 0; }

        body {
            font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
            background: var(--bg-body);
            min-height: 100vh;
            color: var(--text);
            line-height: 1.5;
        }

        .app { display: flex; min-height: 100vh; }

        /* Sidebar - White with subtle border */
        .sidebar {
            width: 260px;
            background: var(--bg-white);
            border-right: 1px solid var(--border);
            display: flex;
            flex-direction: column;
        }
        .sidebar-header {
            padding: 20px;
            background: linear-gradient(to bottom, #eef2ff, var(--bg-white));
            border-bottom: 1px solid var(--border);
        }
        .logo {
            font-size: 22px;
            font-weight: 700;
            color: var(--primary);
        }
        .tenant-name {
            font-size: 13px;
            color: var(--text-muted);
            margin-top: 4px;
        }

        .nav { flex: 1; padding: 16px 0; overflow-y: auto; }
        .nav-section { padding: 0 12px; margin-bottom: 20px; }
        .nav-section-title {
            font-size: 11px;
            font-weight: 600;
            text-transform: uppercase;
            color: var(--text-light);
            margin-bottom: 8px;
            padding: 0 12px;
            letter-spacing: 0.5px;
        }
        .nav-item {
            display: flex;
            align-items: center;
            gap: 12px;
            padding: 10px 12px;
            margin: 2px 0;
            color: var(--text-secondary);
            text-decoration: none;
            cursor: pointer;
            transition: all 0.15s ease;
            border-radius: 8px;
            font-weight: 500;
            font-size: 14px;
        }
        .nav-item:hover {
            background: var(--bg-gray);
            color: var(--primary);
        }
        .nav-item.active {
            background: #eef2ff;
            color: var(--primary);
        }
        .nav-icon {
            width: 20px;
            height: 20px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 14px;
        }

        .sidebar-footer {
            padding: 16px;
            border-top: 1px solid var(--border);
        }
        .user-info {
            display: flex;
            align-items: center;
            gap: 12px;
            padding: 8px;
            border-radius: 8px;
            transition: background 0.15s;
            cursor: pointer;
        }
        .user-info:hover { background: var(--bg-gray); }
        .user-avatar {
            width: 36px;
            height: 36px;
            border-radius: 8px;
            background: var(--primary);
            color: white;
            display: flex;
            align-items: center;
            justify-content: center;
            font-weight: 600;
            font-size: 14px;
        }
        .user-name { font-size: 14px; font-weight: 500; color: var(--text); }
        .user-role { font-size: 12px; color: var(--text-muted); }

        /* Main Content */
        .main {
            flex: 1;
            display: flex;
            flex-direction: column;
            background: var(--bg-body);
        }
        .header {
            padding: 20px 32px;
            background: var(--bg-white);
            border-bottom: 1px solid var(--border);
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .page-title {
            font-size: 24px;
            font-weight: 700;
            color: var(--text);
        }
        .header-actions {
            display: flex;
            gap: 12px;
            align-items: center;
        }
        .content {
            flex: 1;
            padding: 24px 32px;
            overflow-y: auto;
        }

        /* Stats Grid */
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin-bottom: 24px;
        }
        .stat-card {
            background: var(--bg-white);
            border: 1px solid var(--border);
            border-radius: 12px;
            padding: 20px;
            transition: all 0.2s ease;
            box-shadow: var(--shadow-sm);
        }
        .stat-card:hover {
            transform: translateY(-2px);
            box-shadow: var(--shadow);
        }
        .stat-icon {
            font-size: 28px;
            margin-bottom: 12px;
        }
        .stat-value {
            font-size: 32px;
            font-weight: 700;
            color: var(--text);
        }
        .stat-label {
            font-size: 14px;
            color: var(--text-muted);
            margin-top: 4px;
        }

        /* Buttons */
        .btn {
            padding: 10px 20px;
            border: none;
            border-radius: 8px;
            font-size: 14px;
            font-weight: 500;
            cursor: pointer;
            transition: all 0.15s ease;
            display: inline-flex;
            align-items: center;
            justify-content: center;
            gap: 8px;
            font-family: inherit;
        }
        .btn-primary {
            background: var(--primary);
            color: white;
        }
        .btn-primary:hover {
            background: var(--primary-dark);
        }
        .btn-secondary {
            background: var(--bg-white);
            color: var(--text-secondary);
            border: 1px solid var(--border);
        }
        .btn-secondary:hover {
            background: var(--bg-gray);
        }
        .btn-danger {
            background: var(--danger);
            color: white;
        }
        .btn-danger:hover {
            background: #dc2626;
        }
        .btn-sm { padding: 6px 12px; font-size: 13px; }

        /* Tables */
        .table-container {
            background: var(--bg-white);
            border: 1px solid var(--border);
            border-radius: 12px;
            overflow: hidden;
            box-shadow: var(--shadow-sm);
        }
        table { width: 100%; border-collapse: collapse; }
        th, td { padding: 14px 20px; text-align: left; }
        th {
            background: var(--bg-gray);
            font-size: 12px;
            font-weight: 600;
            text-transform: uppercase;
            color: var(--text-muted);
            letter-spacing: 0.5px;
            border-bottom: 1px solid var(--border);
        }
        td {
            border-bottom: 1px solid var(--border-light);
            font-size: 14px;
            color: var(--text);
        }
        tr:hover td { background: var(--bg-gray); }
        tr:last-child td { border-bottom: none; }
        .actions { display: flex; gap: 8px; justify-content: flex-end; }

        /* Forms */
        .form-group { margin-bottom: 20px; }
        .form-label {
            display: block;
            font-size: 14px;
            font-weight: 500;
            color: var(--text-secondary);
            margin-bottom: 6px;
        }
        .form-input {
            width: 100%;
            padding: 10px 14px;
            background: var(--bg-white);
            border: 1px solid var(--border);
            border-radius: 8px;
            color: var(--text);
            font-size: 14px;
            font-family: inherit;
            transition: all 0.15s;
        }
        .form-input:focus {
            outline: none;
            border-color: var(--primary);
            box-shadow: 0 0 0 3px rgba(79, 70, 229, 0.1);
        }
        .form-input::placeholder { color: var(--text-light); }
        textarea.form-input { min-height: 100px; resize: vertical; }
        select.form-input {
            cursor: pointer;
            appearance: none;
            background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='20' height='20' viewBox='0 0 24 24' fill='none' stroke='%236b7280' stroke-width='2'%3E%3Cpolyline points='6 9 12 15 18 9'%3E%3C/polyline%3E%3C/svg%3E");
            background-repeat: no-repeat;
            background-position: right 12px center;
            padding-right: 40px;
        }
        .form-actions {
            display: flex;
            gap: 12px;
            justify-content: flex-end;
            margin-top: 24px;
            padding-top: 20px;
            border-top: 1px solid var(--border-light);
        }

        /* Modal */
        .modal-overlay {
            position: fixed;
            inset: 0;
            background: rgba(0, 0, 0, 0.5);
            display: flex;
            align-items: center;
            justify-content: center;
            z-index: 1000;
            animation: fadeIn 0.15s ease;
        }
        @keyframes fadeIn {
            from { opacity: 0; }
            to { opacity: 1; }
        }
        .modal {
            background: var(--bg-white);
            border-radius: 16px;
            width: 100%;
            max-width: 560px;
            max-height: 90vh;
            overflow-y: auto;
            box-shadow: var(--shadow-lg);
            animation: slideUp 0.2s ease;
        }
        @keyframes slideUp {
            from { opacity: 0; transform: translateY(10px); }
            to { opacity: 1; transform: translateY(0); }
        }
        .modal-header {
            padding: 20px 24px;
            border-bottom: 1px solid var(--border);
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .modal-title {
            font-size: 18px;
            font-weight: 600;
            color: var(--text);
        }
        .modal-close {
            background: transparent;
            border: none;
            color: var(--text-muted);
            width: 32px;
            height: 32px;
            border-radius: 8px;
            font-size: 18px;
            cursor: pointer;
            transition: all 0.15s;
        }
        .modal-close:hover {
            background: var(--bg-gray);
            color: var(--text);
        }
        .modal-body { padding: 24px; }
        .modal-lg { max-width: 720px; }

        /* Empty State */
        .empty-state {
            text-align: center;
            padding: 60px 40px;
            background: var(--bg-white);
            border: 2px dashed var(--border);
            border-radius: 12px;
        }
        .empty-icon { font-size: 48px; margin-bottom: 16px; }
        .empty-title {
            font-size: 18px;
            font-weight: 600;
            color: var(--text);
            margin-bottom: 8px;
        }
        .empty-text {
            color: var(--text-muted);
            margin-bottom: 20px;
        }

        /* Loading */
        .loading {
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 40px;
            flex-direction: column;
            gap: 12px;
        }
        .spinner {
            width: 32px;
            height: 32px;
            border: 3px solid var(--border);
            border-top-color: var(--primary);
            border-radius: 50%;
            animation: spin 0.6s linear infinite;
        }
        @keyframes spin { to { transform: rotate(360deg); } }

        /* Badges */
        .badge {
            display: inline-flex;
            align-items: center;
            padding: 4px 10px;
            border-radius: 6px;
            font-size: 12px;
            font-weight: 500;
        }
        .badge-success { background: #d1fae5; color: #065f46; }
        .badge-warning { background: #fef3c7; color: #92400e; }
        .badge-danger { background: #fee2e2; color: #991b1b; }
        .badge-primary { background: #e0e7ff; color: #3730a3; }

        /* Card styles */
        .card {
            background: var(--bg-white);
            border: 1px solid var(--border);
            border-radius: 12px;
            box-shadow: var(--shadow-sm);
        }
        .card-header {
            padding: 16px 20px;
            border-bottom: 1px solid var(--border);
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .card-title { font-size: 16px; font-weight: 600; }
        .card-body { padding: 20px; }

        /* Articles Manager */
        .articles-manager { max-width: 100%; }
        .articles-toolbar {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 20px;
            padding: 16px 20px;
            background: var(--bg-white);
            border: 1px solid var(--border);
            border-radius: 12px;
            box-shadow: var(--shadow-sm);
        }
        .filters {
            display: flex;
            gap: 12px;
            align-items: center;
        }
        .filter-select, .filter-input {
            padding: 8px 14px;
            background: var(--bg-white);
            border: 1px solid var(--border);
            border-radius: 8px;
            color: var(--text);
            font-size: 14px;
            font-family: inherit;
            min-width: 140px;
        }
        .filter-select:focus, .filter-input:focus {
            outline: none;
            border-color: var(--primary);
        }
        .stat-badge {
            padding: 6px 14px;
            background: #e0e7ff;
            color: #3730a3;
            border-radius: 20px;
            font-size: 13px;
            font-weight: 500;
        }
        .loading-state {
            text-align: center;
            padding: 40px;
            color: var(--text-muted);
        }
        .articles-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
            gap: 20px;
        }
        .article-card {
            background: var(--bg-white);
            border: 1px solid var(--border);
            border-radius: 12px;
            overflow: hidden;
            transition: all 0.2s ease;
            box-shadow: var(--shadow-sm);
        }
        .article-card:hover {
            transform: translateY(-2px);
            box-shadow: var(--shadow);
        }
        .article-image {
            height: 160px;
            overflow: hidden;
            background: var(--bg-gray);
        }
        .article-image img {
            width: 100%;
            height: 100%;
            object-fit: cover;
        }
        .no-image {
            height: 100%;
            display: flex;
            align-items: center;
            justify-content: center;
            color: var(--text-light);
            font-size: 14px;
        }
        .article-content { padding: 16px; }
        .article-meta {
            display: flex;
            gap: 8px;
            margin-bottom: 10px;
        }
        .article-status {
            padding: 3px 8px;
            border-radius: 4px;
            font-size: 11px;
            font-weight: 600;
            text-transform: uppercase;
        }
        .status-published { background: #d1fae5; color: #065f46; }
        .status-draft { background: #fef3c7; color: #92400e; }
        .status-archived { background: var(--bg-gray); color: var(--text-muted); }
        .article-website {
            padding: 3px 8px;
            background: var(--bg-gray);
            border-radius: 4px;
            font-size: 11px;
            color: var(--text-muted);
        }
        .article-title {
            font-size: 16px;
            font-weight: 600;
            color: var(--text);
            margin-bottom: 6px;
            line-height: 1.4;
            display: -webkit-box;
            -webkit-line-clamp: 2;
            -webkit-box-orient: vertical;
            overflow: hidden;
        }
        .article-summary {
            font-size: 14px;
            color: var(--text-muted);
            margin-bottom: 12px;
            line-height: 1.5;
            display: -webkit-box;
            -webkit-line-clamp: 2;
            -webkit-box-orient: vertical;
            overflow: hidden;
        }
        .article-footer {
            display: flex;
            justify-content: space-between;
            font-size: 12px;
            color: var(--text-light);
            padding-bottom: 12px;
            border-bottom: 1px solid var(--border-light);
            margin-bottom: 12px;
        }
        .article-actions { display: flex; gap: 6px; }
        .article-actions .btn-icon {
            padding: 6px 12px;
            background: var(--bg-gray);
            border: 1px solid var(--border);
            border-radius: 6px;
            color: var(--text-muted);
            font-size: 12px;
            cursor: pointer;
            transition: all 0.15s;
        }
        .article-actions .btn-icon:hover {
            background: var(--primary);
            border-color: var(--primary);
            color: white;
        }
        .article-actions .btn-publish:hover { background: var(--success); border-color: var(--success); }
        .article-actions .btn-unpublish:hover { background: var(--warning); border-color: var(--warning); }
        .article-actions .btn-delete:hover { background: var(--danger); border-color: var(--danger); }

        /* Article Form */
        .article-form { padding: 4px; }
        .article-form .form-row { margin-bottom: 16px; }
        .article-form .form-row label {
            display: block;
            font-size: 14px;
            font-weight: 500;
            color: var(--text-secondary);
            margin-bottom: 6px;
        }
        .article-form input,
        .article-form select,
        .article-form textarea {
            width: 100%;
            padding: 10px 14px;
            background: var(--bg-white);
            border: 1px solid var(--border);
            border-radius: 8px;
            color: var(--text);
            font-size: 14px;
            font-family: inherit;
        }
        .article-form input:focus,
        .article-form select:focus,
        .article-form textarea:focus {
            outline: none;
            border-color: var(--primary);
            box-shadow: 0 0 0 3px rgba(79, 70, 229, 0.1);
        }
        .article-form textarea { min-height: 100px; resize: vertical; }
        .form-row-half {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 16px;
        }
        .article-form .form-actions {
            display: flex;
            gap: 12px;
            justify-content: flex-end;
            margin-top: 20px;
            padding-top: 16px;
            border-top: 1px solid var(--border-light);
        }

        /* Responsive */
        @media (max-width: 1024px) {
            .sidebar { width: 220px; }
            .content { padding: 20px; }
        }
        @media (max-width: 768px) {
            .app { flex-direction: column; }
            .sidebar { width: 100%; border-right: none; border-bottom: 1px solid var(--border); }
            .content { padding: 16px; }
            .articles-grid { grid-template-columns: 1fr; }
            .form-row-half { grid-template-columns: 1fr; }
        }
    `

	return css
}
