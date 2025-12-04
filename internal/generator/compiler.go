// Package generator - JSX to JS compiler for database-stored components
package generator

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CompiledComponent represents a pre-compiled component
type CompiledComponent struct {
	ID           uuid.UUID
	Code         string
	SourceJSX    string
	CompiledJS   string
	Hash         string
	CompiledAt   time.Time
}

// ComponentCache caches compiled components in memory
type ComponentCache struct {
	mu         sync.RWMutex
	components map[string]*CompiledComponent // key: code
	byHash     map[string]*CompiledComponent // key: hash (for dedup)
}

// Compiler compiles and caches JSX components
type Compiler struct {
	db    *gorm.DB
	cache *ComponentCache
	gen   *Generator
}

// NewCompiler creates a new JSX compiler
func NewCompiler(db *gorm.DB) *Compiler {
	return &Compiler{
		db: db,
		cache: &ComponentCache{
			components: make(map[string]*CompiledComponent),
			byHash:     make(map[string]*CompiledComponent),
		},
		gen: NewGenerator(db),
	}
}

// CompileAll compiles all components and caches them
func (c *Compiler) CompileAll(tenantID uuid.UUID) error {
	// Load existing UI components from database
	rows, err := c.db.Raw(`
		SELECT id, code, code_jsx
		FROM ui_components
		WHERE (tenant_id = ? OR tenant_id IS NULL) AND is_active = true
	`, tenantID).Rows()
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id uuid.UUID
		var code, jsx string
		if err := rows.Scan(&id, &code, &jsx); err != nil {
			return err
		}

		compiled, err := c.compileJSX(jsx)
		if err != nil {
			return fmt.Errorf("compile %s: %w", code, err)
		}

		c.cacheComponent(&CompiledComponent{
			ID:         id,
			Code:       code,
			SourceJSX:  jsx,
			CompiledJS: compiled,
			Hash:       hashCode(jsx),
			CompiledAt: time.Now(),
		})
	}

	return nil
}

// GenerateEntityComponents generates and compiles components for all entities
func (c *Compiler) GenerateEntityComponents(tenantID uuid.UUID) error {
	entities, err := c.gen.LoadEntities(tenantID)
	if err != nil {
		return err
	}

	for _, entity := range entities {
		// Generate List component
		listJSX, err := c.gen.GenerateReactComponent(entity, "list")
		if err != nil {
			return err
		}
		if err := c.saveComponent(tenantID, entity.Code+"_list", entity.Name+" List", "entity", listJSX); err != nil {
			return err
		}

		// Generate Form component
		formJSX, err := c.gen.GenerateReactComponent(entity, "form")
		if err != nil {
			return err
		}
		if err := c.saveComponent(tenantID, entity.Code+"_form", entity.Name+" Form", "entity", formJSX); err != nil {
			return err
		}

		// Generate Detail component
		detailJSX, err := c.gen.GenerateReactComponent(entity, "detail")
		if err != nil {
			return err
		}
		if err := c.saveComponent(tenantID, entity.Code+"_detail", entity.Name+" Detail", "entity", detailJSX); err != nil {
			return err
		}
	}

	return nil
}

// saveComponent saves a component to the database
func (c *Compiler) saveComponent(tenantID uuid.UUID, code, name, category, jsx string) error {
	compiled, err := c.compileJSX(jsx)
	if err != nil {
		return err
	}

	hash := hashCode(jsx)

	// Upsert component
	result := c.db.Exec(`
		INSERT INTO ui_components (tenant_id, code, name, category, code_jsx, is_system, is_active)
		VALUES (?, ?, ?, ?, ?, false, true)
		ON CONFLICT (code) DO UPDATE SET
			name = EXCLUDED.name,
			code_jsx = EXCLUDED.code_jsx,
			updated_at = CURRENT_TIMESTAMP
	`, tenantID, code, name, category, jsx)

	if result.Error != nil {
		return result.Error
	}

	// Get the component ID
	var id uuid.UUID
	c.db.Raw("SELECT id FROM ui_components WHERE code = ?", code).Scan(&id)

	// Cache it
	c.cacheComponent(&CompiledComponent{
		ID:         id,
		Code:       code,
		SourceJSX:  jsx,
		CompiledJS: compiled,
		Hash:       hash,
		CompiledAt: time.Now(),
	})

	return nil
}

// compileJSX transforms JSX to JavaScript
// Note: In production, this would use a proper JSX transformer
// For now, we do simple transformations that work with Babel in browser
func (c *Compiler) compileJSX(jsx string) (string, error) {
	// The JSX will be compiled by Babel in the browser
	// We just wrap it properly here
	return jsx, nil
}

// GetComponent returns a cached component
func (c *Compiler) GetComponent(code string) *CompiledComponent {
	c.cache.mu.RLock()
	defer c.cache.mu.RUnlock()
	return c.cache.components[code]
}

// GetAllComponents returns all cached components
func (c *Compiler) GetAllComponents() []*CompiledComponent {
	c.cache.mu.RLock()
	defer c.cache.mu.RUnlock()

	result := make([]*CompiledComponent, 0, len(c.cache.components))
	for _, comp := range c.cache.components {
		result = append(result, comp)
	}
	return result
}

// GetComponentBundle returns all components as a single JS bundle
func (c *Compiler) GetComponentBundle() string {
	c.cache.mu.RLock()
	defer c.cache.mu.RUnlock()

	var bundle string
	bundle += "// Genesis Component Bundle - Generated\n"
	bundle += "const GenesisComponents = {};\n\n"

	for code, comp := range c.cache.components {
		bundle += fmt.Sprintf("// Component: %s\n", code)
		bundle += comp.CompiledJS
		bundle += fmt.Sprintf("\nGenesisComponents['%s'] = %s;\n\n", code, extractFunctionName(comp.CompiledJS))
	}

	return bundle
}

func (c *Compiler) cacheComponent(comp *CompiledComponent) {
	c.cache.mu.Lock()
	defer c.cache.mu.Unlock()
	c.cache.components[comp.Code] = comp
	c.cache.byHash[comp.Hash] = comp
}

// InvalidateCache clears the component cache
func (c *Compiler) InvalidateCache() {
	c.cache.mu.Lock()
	defer c.cache.mu.Unlock()
	c.cache.components = make(map[string]*CompiledComponent)
	c.cache.byHash = make(map[string]*CompiledComponent)
}

func hashCode(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:8])
}

func extractFunctionName(js string) string {
	// Extract function name from "function Foo(...)" or use anonymous
	// Simple implementation - looks for "function X("
	start := 0
	for i := 0; i < len(js)-9; i++ {
		if js[i:i+8] == "function" && js[i+8] == ' ' {
			start = i + 9
			break
		}
	}
	if start == 0 {
		return "Anonymous"
	}
	end := start
	for end < len(js) && js[end] != '(' && js[end] != ' ' {
		end++
	}
	if end > start {
		return js[start:end]
	}
	return "Anonymous"
}
