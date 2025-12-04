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
	pageData, err := r.GetPageData(tenantID, route)
	if err != nil {
		return "", err
	}
	pageData.User = user

	return r.GenerateHTML(pageData)
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

        // API helper
        const api = {
            token: localStorage.getItem('token'),
            tenantId: PAGE_DATA.tenant?.id,

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
                const res = await fetch(endpoint, options);
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
                    setData(result || []);
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
                        <Navigation modules={modules} entities={entities} currentPath={window.location.pathname} />
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
                                {entity && <Button onClick={handleCreate}>+ Create {entity.name}</Button>}
                            </div>
                        </header>
                        <div className="content">
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
	// Default CSS
	css := `
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: #0f0f1a;
            min-height: 100vh;
            color: #fff;
        }
        .app { display: flex; min-height: 100vh; }

        /* Sidebar */
        .sidebar {
            width: 260px;
            background: linear-gradient(180deg, #1a1a2e 0%, #16213e 100%);
            border-right: 1px solid rgba(255,255,255,0.1);
            display: flex;
            flex-direction: column;
        }
        .sidebar-header {
            padding: 20px;
            border-bottom: 1px solid rgba(255,255,255,0.1);
        }
        .logo {
            font-size: 24px;
            font-weight: bold;
            background: linear-gradient(90deg, #00d4ff, #7b2cbf);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }
        .tenant-name {
            font-size: 12px;
            color: rgba(255,255,255,0.5);
            margin-top: 4px;
        }
        .nav { flex: 1; padding: 20px 0; }
        .nav-section { padding: 0 20px; margin-bottom: 20px; }
        .nav-section-title {
            font-size: 11px;
            text-transform: uppercase;
            color: rgba(255,255,255,0.4);
            margin-bottom: 10px;
            letter-spacing: 1px;
        }
        .nav-item {
            display: flex;
            align-items: center;
            gap: 12px;
            padding: 12px 20px;
            color: rgba(255,255,255,0.7);
            text-decoration: none;
            cursor: pointer;
            transition: all 0.2s;
            border-left: 3px solid transparent;
        }
        .nav-item:hover {
            background: rgba(255,255,255,0.05);
            color: #fff;
        }
        .nav-item.active {
            background: rgba(0,212,255,0.1);
            color: #00d4ff;
            border-left-color: #00d4ff;
        }
        .nav-icon { width: 20px; text-align: center; }
        .sidebar-footer {
            padding: 20px;
            border-top: 1px solid rgba(255,255,255,0.1);
        }
        .user-info { display: flex; align-items: center; gap: 12px; }
        .user-avatar {
            width: 36px;
            height: 36px;
            border-radius: 50%;
            background: linear-gradient(135deg, #00d4ff, #7b2cbf);
            display: flex;
            align-items: center;
            justify-content: center;
            font-weight: bold;
        }
        .user-name { font-size: 14px; font-weight: 500; }
        .user-role { font-size: 11px; color: rgba(255,255,255,0.5); }

        /* Main Content */
        .main { flex: 1; display: flex; flex-direction: column; }
        .header {
            padding: 20px 30px;
            background: rgba(255,255,255,0.02);
            border-bottom: 1px solid rgba(255,255,255,0.1);
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .page-title { font-size: 24px; font-weight: 600; }
        .content { flex: 1; padding: 30px; overflow-y: auto; }

        /* Stats Grid */
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        .stat-card {
            background: rgba(255,255,255,0.03);
            border: 1px solid rgba(255,255,255,0.1);
            border-radius: 12px;
            padding: 24px;
        }
        .stat-value {
            font-size: 36px;
            font-weight: 700;
            background: linear-gradient(90deg, #00d4ff, #7b2cbf);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }
        .stat-label { font-size: 14px; color: rgba(255,255,255,0.5); margin-top: 4px; }

        /* Buttons */
        .btn {
            padding: 10px 20px;
            border: none;
            border-radius: 8px;
            font-size: 14px;
            font-weight: 500;
            cursor: pointer;
            transition: all 0.2s;
            display: inline-flex;
            align-items: center;
            gap: 8px;
        }
        .btn-primary {
            background: linear-gradient(90deg, #00d4ff, #7b2cbf);
            color: #fff;
        }
        .btn-primary:hover {
            transform: translateY(-2px);
            box-shadow: 0 8px 24px rgba(0,212,255,0.3);
        }
        .btn-secondary { background: rgba(255,255,255,0.1); color: #fff; }
        .btn-secondary:hover { background: rgba(255,255,255,0.15); }
        .btn-danger { background: #ef4444; color: #fff; }
        .btn-sm { padding: 6px 12px; font-size: 12px; }

        /* Tables */
        .table-container {
            background: rgba(255,255,255,0.03);
            border: 1px solid rgba(255,255,255,0.1);
            border-radius: 12px;
            overflow: hidden;
        }
        table { width: 100%; border-collapse: collapse; }
        th, td {
            padding: 16px 20px;
            text-align: left;
            border-bottom: 1px solid rgba(255,255,255,0.05);
        }
        th {
            background: rgba(255,255,255,0.03);
            font-size: 12px;
            text-transform: uppercase;
            color: rgba(255,255,255,0.5);
        }
        tr:hover td { background: rgba(255,255,255,0.02); }
        .actions { display: flex; gap: 8px; }

        /* Forms */
        .form-group { margin-bottom: 20px; }
        .form-label {
            display: block;
            font-size: 13px;
            color: rgba(255,255,255,0.7);
            margin-bottom: 8px;
        }
        .form-input {
            width: 100%;
            padding: 12px 16px;
            background: rgba(255,255,255,0.05);
            border: 1px solid rgba(255,255,255,0.1);
            border-radius: 8px;
            color: #fff;
            font-size: 14px;
        }
        .form-input:focus { outline: none; border-color: #00d4ff; }
        .form-actions { display: flex; gap: 12px; justify-content: flex-end; margin-top: 24px; }
        .required { color: #ef4444; }

        /* Modal */
        .modal-overlay {
            position: fixed;
            inset: 0;
            background: rgba(0,0,0,0.7);
            display: flex;
            align-items: center;
            justify-content: center;
            z-index: 1000;
        }
        .modal {
            background: #1a1a2e;
            border: 1px solid rgba(255,255,255,0.1);
            border-radius: 16px;
            width: 100%;
            max-width: 500px;
            max-height: 90vh;
            overflow-y: auto;
        }
        .modal-header {
            padding: 20px 24px;
            border-bottom: 1px solid rgba(255,255,255,0.1);
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .modal-title { font-size: 18px; font-weight: 600; }
        .modal-close {
            background: none;
            border: none;
            color: rgba(255,255,255,0.5);
            font-size: 24px;
            cursor: pointer;
        }
        .modal-body { padding: 24px; }

        /* Empty State */
        .empty-state { text-align: center; padding: 60px 20px; color: rgba(255,255,255,0.5); }
        .empty-icon { font-size: 48px; margin-bottom: 16px; }
        .empty-title { font-size: 18px; color: #fff; margin-bottom: 8px; }
        .empty-text { margin-bottom: 20px; }

        /* Loading */
        .loading { display: flex; align-items: center; justify-content: center; padding: 40px; }
        .spinner {
            width: 32px;
            height: 32px;
            border: 3px solid rgba(255,255,255,0.1);
            border-top-color: #00d4ff;
            border-radius: 50%;
            animation: spin 1s linear infinite;
        }
        @keyframes spin { to { transform: rotate(360deg); } }
    `

	return css
}
