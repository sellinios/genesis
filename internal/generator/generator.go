// Package generator - Code generator for high-performance entity handlers
package generator

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Generator generates optimized code from entity definitions
type Generator struct {
	db *gorm.DB
}

// EntityDef represents an entity definition for code generation
type EntityDef struct {
	ID         uuid.UUID
	TenantID   uuid.UUID
	ModuleCode string
	Code       string
	Name       string
	NamePlural string
	Fields     []FieldDef
	Relations  []RelationDef
}

// FieldDef represents a field definition
type FieldDef struct {
	Code         string
	Name         string
	FieldType    string
	GoType       string
	JSType       string
	IsRequired   bool
	InList       bool
	InDetail     bool
	InForm       bool
	DisplayOrder int
}

// RelationDef represents a relation definition
type RelationDef struct {
	Name           string
	TargetEntity   string
	RelationType   string
	SourceField    string
	TargetField    string
}

// NewGenerator creates a new code generator
func NewGenerator(db *gorm.DB) *Generator {
	return &Generator{db: db}
}

// LoadEntities loads all entity definitions from database
func (g *Generator) LoadEntities(tenantID uuid.UUID) ([]EntityDef, error) {
	var entities []EntityDef

	rows, err := g.db.Raw(`
		SELECT
			e.id, e.tenant_id, m.code as module_code, e.code, e.name, e.name_plural
		FROM entities e
		JOIN modules m ON m.id = e.module_id
		WHERE e.tenant_id = ? AND e.is_active = true
		ORDER BY m.display_order, e.display_order
	`, tenantID).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var e EntityDef
		if err := rows.Scan(&e.ID, &e.TenantID, &e.ModuleCode, &e.Code, &e.Name, &e.NamePlural); err != nil {
			return nil, err
		}

		// Load fields
		e.Fields, err = g.loadFields(e.ID)
		if err != nil {
			return nil, err
		}

		// Load relations
		e.Relations, err = g.loadRelations(e.ID)
		if err != nil {
			return nil, err
		}

		entities = append(entities, e)
	}

	return entities, nil
}

func (g *Generator) loadFields(entityID uuid.UUID) ([]FieldDef, error) {
	var fields []FieldDef

	rows, err := g.db.Raw(`
		SELECT
			f.code, f.name, ft.code as field_type,
			f.is_required, f.in_list, f.in_detail, f.in_form, f.display_order
		FROM fields f
		JOIN field_types ft ON ft.id = f.field_type_id
		WHERE f.entity_id = ?
		ORDER BY f.display_order
	`, entityID).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var f FieldDef
		if err := rows.Scan(&f.Code, &f.Name, &f.FieldType, &f.IsRequired, &f.InList, &f.InDetail, &f.InForm, &f.DisplayOrder); err != nil {
			return nil, err
		}
		f.GoType = fieldTypeToGo(f.FieldType)
		f.JSType = fieldTypeToJS(f.FieldType)
		fields = append(fields, f)
	}

	return fields, nil
}

func (g *Generator) loadRelations(entityID uuid.UUID) ([]RelationDef, error) {
	var relations []RelationDef

	rows, err := g.db.Raw(`
		SELECT
			r.name, te.code as target_entity, r.relation_type,
			r.source_field_code, r.target_field_code
		FROM relations r
		JOIN entities te ON te.id = r.target_entity_id
		WHERE r.source_entity_id = ?
	`, entityID).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var r RelationDef
		if err := rows.Scan(&r.Name, &r.TargetEntity, &r.RelationType, &r.SourceField, &r.TargetField); err != nil {
			return nil, err
		}
		relations = append(relations, r)
	}

	return relations, nil
}

// GenerateGoHandler generates optimized Go handler code for an entity
func (g *Generator) GenerateGoHandler(entity EntityDef) (string, error) {
	tmpl := template.Must(template.New("handler").Funcs(template.FuncMap{
		"title":     strings.Title,
		"lower":     strings.ToLower,
		"snakeCase": toSnakeCase,
	}).Parse(goHandlerTemplate))

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, entity); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// GenerateReactComponent generates optimized React component for an entity
func (g *Generator) GenerateReactComponent(entity EntityDef, componentType string) (string, error) {
	var tmplStr string
	switch componentType {
	case "list":
		tmplStr = reactListTemplate
	case "form":
		tmplStr = reactFormTemplate
	case "detail":
		tmplStr = reactDetailTemplate
	default:
		return "", fmt.Errorf("unknown component type: %s", componentType)
	}

	tmpl := template.Must(template.New("react").Funcs(template.FuncMap{
		"title":     strings.Title,
		"lower":     strings.ToLower,
		"camelCase": toCamelCase,
	}).Parse(tmplStr))

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, entity); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// Helper functions
func fieldTypeToGo(ft string) string {
	switch ft {
	case "string", "email", "url", "phone", "text", "richtext", "enum":
		return "string"
	case "integer":
		return "int64"
	case "decimal":
		return "float64"
	case "boolean":
		return "bool"
	case "date", "datetime":
		return "time.Time"
	case "uuid", "relation":
		return "uuid.UUID"
	case "json":
		return "map[string]any"
	default:
		return "string"
	}
}

func fieldTypeToJS(ft string) string {
	switch ft {
	case "string", "email", "url", "phone", "text", "richtext", "enum", "date", "datetime":
		return "string"
	case "integer", "decimal":
		return "number"
	case "boolean":
		return "boolean"
	default:
		return "string"
	}
}

func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	for i := range parts {
		if i == 0 {
			parts[i] = strings.ToLower(parts[i])
		} else {
			parts[i] = strings.Title(parts[i])
		}
	}
	return strings.Join(parts, "")
}

// Go Handler Template
const goHandlerTemplate = `// Code generated by Genesis. DO NOT EDIT.
// Entity: {{.Name}}

package generated

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// {{title .Code}} represents the {{.Name}} entity
type {{title .Code}} struct {
	ID        uuid.UUID  ` + "`json:\"id\" gorm:\"primaryKey\"`" + `
	TenantID  uuid.UUID  ` + "`json:\"tenant_id\"`" + `
{{range .Fields}}	{{title .Code}} {{.GoType}} ` + "`json:\"{{.Code}}\"`" + `
{{end}}{{range .Relations}}	{{title .SourceField}} *uuid.UUID ` + "`json:\"{{.SourceField}},omitempty\"`" + `
{{end}}	CreatedAt time.Time  ` + "`json:\"created_at\"`" + `
	UpdatedAt time.Time  ` + "`json:\"updated_at\"`" + `
}

func ({{title .Code}}) TableName() string {
	return "data_{{.Code}}"
}

// {{title .Code}}Handler handles {{.Name}} requests
type {{title .Code}}Handler struct {
	db *gorm.DB
}

func New{{title .Code}}Handler(db *gorm.DB) *{{title .Code}}Handler {
	return &{{title .Code}}Handler{db: db}
}

// List returns all {{.NamePlural}}
func (h *{{title .Code}}Handler) List(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var items []{{title .Code}}
	if err := h.db.Where("tenant_id = ?", tenantID).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": items, "total": len(items)})
}

// Get returns a single {{.Name}}
func (h *{{title .Code}}Handler) Get(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenant_id")

	var item {{title .Code}}
	if err := h.db.Where("id = ? AND tenant_id = ?", id, tenantID).First(&item).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}

	c.JSON(http.StatusOK, item)
}

// Create creates a new {{.Name}}
func (h *{{title .Code}}Handler) Create(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var item {{title .Code}}
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item.ID = uuid.New()
	item.TenantID = uuid.MustParse(tenantID)

	if err := h.db.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, item)
}

// Update updates a {{.Name}}
func (h *{{title .Code}}Handler) Update(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenant_id")

	var item {{title .Code}}
	if err := h.db.Where("id = ? AND tenant_id = ?", id, tenantID).First(&item).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}

	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.db.Save(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, item)
}

// Delete deletes a {{.Name}}
func (h *{{title .Code}}Handler) Delete(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenant_id")

	if err := h.db.Where("id = ? AND tenant_id = ?", id, tenantID).Delete(&{{title .Code}}{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Deleted"})
}
`

// React List Template
const reactListTemplate = `// Code generated by Genesis. DO NOT EDIT.
// Entity: {{.Name}} - List Component

function {{title .Code}}List({ onSelect, onCreate }) {
  const [data, setData] = React.useState([]);
  const [loading, setLoading] = React.useState(true);

  React.useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/data/{{.Code}}', {
        headers: { 'Authorization': 'Bearer ' + localStorage.getItem('token') }
      });
      const json = await res.json();
      setData(json.data || []);
    } catch (err) {
      console.error(err);
    }
    setLoading(false);
  };

  const handleDelete = async (id) => {
    if (!confirm('Delete this {{.Name}}?')) return;
    await fetch('/api/data/{{.Code}}/' + id, {
      method: 'DELETE',
      headers: { 'Authorization': 'Bearer ' + localStorage.getItem('token') }
    });
    fetchData();
  };

  if (loading) return <div className="loading"><div className="spinner" /></div>;

  return (
    <div className="entity-list">
      <div className="list-header">
        <h2>{{.NamePlural}}</h2>
        <button className="btn btn-primary" onClick={onCreate}>+ New {{.Name}}</button>
      </div>

      {data.length === 0 ? (
        <div className="empty-state">
          <div className="empty-icon">📋</div>
          <div className="empty-title">No {{.NamePlural}} yet</div>
          <button className="btn btn-primary" onClick={onCreate}>Create First {{.Name}}</button>
        </div>
      ) : (
        <table className="data-table">
          <thead>
            <tr>
{{range .Fields}}{{if .InList}}              <th>{{.Name}}</th>
{{end}}{{end}}              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {data.map(row => (
              <tr key={row.id} onClick={() => onSelect(row)}>
{{range .Fields}}{{if .InList}}                <td>{formatValue(row.{{camelCase .Code}}, '{{.FieldType}}')}</td>
{{end}}{{end}}                <td className="actions">
                  <button className="btn btn-sm" onClick={(e) => { e.stopPropagation(); onSelect(row); }}>Edit</button>
                  <button className="btn btn-sm btn-danger" onClick={(e) => { e.stopPropagation(); handleDelete(row.id); }}>Delete</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}

function formatValue(val, type) {
  if (val === null || val === undefined) return '-';
  if (type === 'boolean') return val ? '✓' : '✗';
  if (type === 'date') return new Date(val).toLocaleDateString();
  if (type === 'datetime') return new Date(val).toLocaleString();
  if (type === 'decimal') return Number(val).toFixed(2);
  return String(val);
}
`

// React Form Template
const reactFormTemplate = `// Code generated by Genesis. DO NOT EDIT.
// Entity: {{.Name}} - Form Component

function {{title .Code}}Form({ item, onSave, onCancel }) {
  const [values, setValues] = React.useState(item || {});
  const [loading, setLoading] = React.useState(false);
  const [errors, setErrors] = React.useState({});

  const handleChange = (field, value) => {
    setValues(prev => ({ ...prev, [field]: value }));
    setErrors(prev => ({ ...prev, [field]: null }));
  };

  const validate = () => {
    const errs = {};
{{range .Fields}}{{if .IsRequired}}    if (!values.{{camelCase .Code}}) errs.{{camelCase .Code}} = '{{.Name}} is required';
{{end}}{{end}}    setErrors(errs);
    return Object.keys(errs).length === 0;
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!validate()) return;

    setLoading(true);
    try {
      const method = item?.id ? 'PUT' : 'POST';
      const url = item?.id ? '/api/data/{{.Code}}/' + item.id : '/api/data/{{.Code}}';

      const res = await fetch(url, {
        method,
        headers: {
          'Content-Type': 'application/json',
          'Authorization': 'Bearer ' + localStorage.getItem('token')
        },
        body: JSON.stringify(values)
      });

      if (!res.ok) throw new Error('Save failed');

      const saved = await res.json();
      onSave(saved);
    } catch (err) {
      setErrors({ _form: err.message });
    }
    setLoading(false);
  };

  return (
    <form onSubmit={handleSubmit} className="entity-form">
      <h2>{item?.id ? 'Edit' : 'New'} {{.Name}}</h2>

      {errors._form && <div className="form-error">{errors._form}</div>}

{{range .Fields}}{{if .InForm}}      <div className="form-group">
        <label className="form-label">{{.Name}}{{if .IsRequired}} *{{end}}</label>
{{if eq .FieldType "text"}}        <textarea
          className={'form-input' + (errors.{{camelCase .Code}} ? ' error' : '')}
          value={values.{{camelCase .Code}} || ''}
          onChange={e => handleChange('{{camelCase .Code}}', e.target.value)}
          rows={4}
        />
{{else if eq .FieldType "boolean"}}        <input
          type="checkbox"
          checked={!!values.{{camelCase .Code}}}
          onChange={e => handleChange('{{camelCase .Code}}', e.target.checked)}
        />
{{else if eq .FieldType "integer"}}        <input
          type="number"
          className={'form-input' + (errors.{{camelCase .Code}} ? ' error' : '')}
          value={values.{{camelCase .Code}} || ''}
          onChange={e => handleChange('{{camelCase .Code}}', parseInt(e.target.value))}
        />
{{else if eq .FieldType "decimal"}}        <input
          type="number"
          step="0.01"
          className={'form-input' + (errors.{{camelCase .Code}} ? ' error' : '')}
          value={values.{{camelCase .Code}} || ''}
          onChange={e => handleChange('{{camelCase .Code}}', parseFloat(e.target.value))}
        />
{{else if eq .FieldType "date"}}        <input
          type="date"
          className={'form-input' + (errors.{{camelCase .Code}} ? ' error' : '')}
          value={values.{{camelCase .Code}} || ''}
          onChange={e => handleChange('{{camelCase .Code}}', e.target.value)}
        />
{{else if eq .FieldType "datetime"}}        <input
          type="datetime-local"
          className={'form-input' + (errors.{{camelCase .Code}} ? ' error' : '')}
          value={values.{{camelCase .Code}} || ''}
          onChange={e => handleChange('{{camelCase .Code}}', e.target.value)}
        />
{{else}}        <input
          type="{{if eq .FieldType "email"}}email{{else if eq .FieldType "url"}}url{{else}}text{{end}}"
          className={'form-input' + (errors.{{camelCase .Code}} ? ' error' : '')}
          value={values.{{camelCase .Code}} || ''}
          onChange={e => handleChange('{{camelCase .Code}}', e.target.value)}
        />
{{end}}        {errors.{{camelCase .Code}} && <span className="field-error">{errors.{{camelCase .Code}}}</span>}
      </div>
{{end}}{{end}}
      <div className="form-actions">
        <button type="button" className="btn btn-secondary" onClick={onCancel} disabled={loading}>Cancel</button>
        <button type="submit" className="btn btn-primary" disabled={loading}>
          {loading ? 'Saving...' : 'Save'}
        </button>
      </div>
    </form>
  );
}
`

// React Detail Template
const reactDetailTemplate = `// Code generated by Genesis. DO NOT EDIT.
// Entity: {{.Name}} - Detail Component

function {{title .Code}}Detail({ item, onEdit, onBack }) {
  if (!item) return null;

  return (
    <div className="entity-detail">
      <div className="detail-header">
        <button className="btn btn-ghost" onClick={onBack}>← Back</button>
        <h2>{{.Name}} Details</h2>
        <button className="btn btn-primary" onClick={() => onEdit(item)}>Edit</button>
      </div>

      <div className="detail-grid">
{{range .Fields}}{{if .InDetail}}        <div className="detail-field">
          <label>{{.Name}}</label>
          <span>{formatValue(item.{{camelCase .Code}}, '{{.FieldType}}')}</span>
        </div>
{{end}}{{end}}      </div>
    </div>
  );
}
`
