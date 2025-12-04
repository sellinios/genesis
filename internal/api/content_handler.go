// Package api - Content handler for serving Aethra articles from Genesis
// This provides the same article serving functionality as Aethra's public API
package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ContentHandler handles public content endpoints (articles, categories)
type ContentHandler struct {
	db        *gorm.DB
	aethraDB  *sql.DB // Direct connection to aethra_internal database
}

// NewContentHandler creates a new content handler
func NewContentHandler(db *gorm.DB) *ContentHandler {
	return &ContentHandler{db: db}
}

// NewContentHandlerWithAethra creates a content handler with Aethra database connection
func NewContentHandlerWithAethra(db *gorm.DB, aethraDB *sql.DB) *ContentHandler {
	return &ContentHandler{
		db:       db,
		aethraDB: aethraDB,
	}
}

// SetAethraDB sets the Aethra database connection
func (h *ContentHandler) SetAethraDB(aethraDB *sql.DB) {
	h.aethraDB = aethraDB
}

// getDomain extracts domain from request (query param, header, or referer)
func (h *ContentHandler) getDomain(c *gin.Context) string {
	domain := c.Query("domain")
	if domain == "" {
		domain = c.GetHeader("X-Original-Host")
	}

	// Try to extract from referer
	if domain == "" {
		referer := c.GetHeader("Referer")
		if referer != "" {
			if strings.Contains(referer, "kairos.gr") {
				domain = "kairos.gr"
			} else if strings.Contains(referer, "wfy24.com") {
				domain = "wfy24.com"
			}
		}
	}

	// Default domain
	if domain == "" {
		domain = "kairos.gr"
	}

	return domain
}

// transformImageURL transforms image URLs from various sources to local media paths
func transformImageURL(url string) string {
	if url == "" {
		return url
	}

	// Old local upload - needs media proxy
	if strings.HasPrefix(url, "/uploads/") {
		return "/api/media" + url + ".webp"
	}

	// Intranet URL - convert to media proxy (without /api/)
	if strings.HasPrefix(url, "https://intranet.aethra.dev/uploads/") {
		if !strings.HasSuffix(url, ".webp") {
			url = url + ".webp"
		}
		return strings.Replace(url, "https://intranet.aethra.dev/uploads/", "/api/media/uploads/", 1)
	}

	// Intranet URL - convert to media proxy (with /api/)
	if strings.HasPrefix(url, "https://intranet.aethra.dev/api/uploads/") {
		if !strings.HasSuffix(url, ".webp") {
			url = url + ".webp"
		}
		return strings.Replace(url, "https://intranet.aethra.dev/api/uploads/", "/api/media/uploads/", 1)
	}

	// Convert Google Cloud Storage URLs to local media paths
	if strings.HasPrefix(url, "https://storage.googleapis.com/aethra-media/") {
		return strings.Replace(url, "https://storage.googleapis.com/aethra-media/", "/", 1)
	}

	return url
}

// transformContentURLs transforms URLs within HTML content
func transformContentURLs(content string) string {
	// Replace intranet URLs with local media path
	content = strings.ReplaceAll(content, "https://intranet.aethra.dev/api/media/", "/media/")
	content = strings.ReplaceAll(content, "/api/media/", "/media/")
	// Replace GCS URLs with local media
	content = strings.ReplaceAll(content, "https://storage.googleapis.com/aethra-media/", "/")
	return content
}

// GetArticles retrieves blog articles from Aethra's database
// GET /api/public/articles
func (h *ContentHandler) GetArticles(c *gin.Context) {
	if h.aethraDB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Aethra database not configured",
			"hint":  "Set AETHRA_DB_* environment variables",
		})
		return
	}

	domain := h.getDomain(c)
	category := c.Query("category")
	search := c.Query("search")

	// Get limit and offset from query parameters
	limit := 20 // default
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Build query with optional filters
	query := `
		SELECT a.id, a.title, a.slug, a.summary, a.published_at,
		       a.featured_image, a.views, COALESCE(u.username, 'Unknown') as author,
		       a.category
		FROM articles a
		JOIN websites w ON a.website_id = w.id
		LEFT JOIN users u ON a.author_id = u.id
		WHERE w.domain = $1 AND a.status = 'published'
	`

	args := []interface{}{domain}
	paramCount := 1

	if category != "" {
		paramCount++
		query += fmt.Sprintf(` AND LOWER(a.category) = LOWER($%d)`, paramCount)
		args = append(args, category)
	}

	if search != "" {
		paramCount++
		titleParam := paramCount
		paramCount++
		summaryParam := paramCount
		paramCount++
		contentParam := paramCount

		query += fmt.Sprintf(` AND (LOWER(a.title) LIKE LOWER($%d) OR LOWER(a.summary) LIKE LOWER($%d) OR LOWER(a.content) LIKE LOWER($%d))`, titleParam, summaryParam, contentParam)
		searchPattern := "%" + search + "%"
		args = append(args, searchPattern, searchPattern, searchPattern)
	}

	paramCount++
	limitParam := paramCount
	paramCount++
	offsetParam := paramCount

	query += fmt.Sprintf(`
		ORDER BY a.published_at DESC
		LIMIT $%d OFFSET $%d`, limitParam, offsetParam)

	args = append(args, limit, offset)
	rows, err := h.aethraDB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch articles",
			"details": err.Error(),
			"domain":  domain,
		})
		return
	}
	defer rows.Close()

	articles := []map[string]interface{}{}
	for rows.Next() {
		var article struct {
			ID            int
			Title         string
			Slug          string
			Summary       sql.NullString
			PublishedAt   sql.NullTime
			FeaturedImage sql.NullString
			Views         int
			Author        sql.NullString
			Category      sql.NullString
		}

		err := rows.Scan(&article.ID, &article.Title, &article.Slug,
			&article.Summary, &article.PublishedAt,
			&article.FeaturedImage, &article.Views, &article.Author, &article.Category)
		if err != nil {
			continue
		}

		// Process featured image URL
		featuredImage := transformImageURL(article.FeaturedImage.String)

		articles = append(articles, map[string]interface{}{
			"id":             article.ID,
			"title":          article.Title,
			"slug":           article.Slug,
			"summary":        article.Summary.String,
			"published_at":   article.PublishedAt.Time,
			"featured_image": featuredImage,
			"views":          article.Views,
			"author":         article.Author.String,
			"category":       article.Category.String,
			"created_at":     article.PublishedAt.Time,
		})
	}

	// Get total count with same filters
	var total int
	countQuery := `
		SELECT COUNT(*) FROM articles a
		JOIN websites w ON a.website_id = w.id
		WHERE w.domain = $1 AND a.status = 'published'
	`
	countArgs := []interface{}{domain}
	countParamCount := 1

	if category != "" {
		countParamCount++
		countQuery += fmt.Sprintf(` AND LOWER(a.category) = LOWER($%d)`, countParamCount)
		countArgs = append(countArgs, category)
	}

	if search != "" {
		countParamCount++
		countTitleParam := countParamCount
		countParamCount++
		countSummaryParam := countParamCount
		countParamCount++
		countContentParam := countParamCount

		countQuery += fmt.Sprintf(` AND (LOWER(a.title) LIKE LOWER($%d) OR LOWER(a.summary) LIKE LOWER($%d) OR LOWER(a.content) LIKE LOWER($%d))`, countTitleParam, countSummaryParam, countContentParam)
		searchPattern := "%" + search + "%"
		countArgs = append(countArgs, searchPattern, searchPattern, searchPattern)
	}

	h.aethraDB.QueryRow(countQuery, countArgs...).Scan(&total)

	c.JSON(http.StatusOK, gin.H{
		"articles": articles,
		"total":    total,
	})
}

// GetArticle retrieves a single article by slug
// GET /api/public/articles/:slug
func (h *ContentHandler) GetArticle(c *gin.Context) {
	if h.aethraDB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Aethra database not configured",
			"hint":  "Set AETHRA_DB_* environment variables",
		})
		return
	}

	slug := c.Param("slug")
	domain := h.getDomain(c)

	query := `
		SELECT a.id, a.title, a.slug, a.summary, a.content, a.published_at,
		       a.featured_image, a.views, COALESCE(u.username, 'Unknown') as author,
		       a.meta_description, a.category, a.place_slug
		FROM articles a
		JOIN websites w ON a.website_id = w.id
		LEFT JOIN users u ON a.author_id = u.id
		WHERE w.domain = $1 AND a.slug = $2 AND a.status = 'published'
	`

	var article struct {
		ID              int
		Title           string
		Slug            string
		Summary         sql.NullString
		Content         sql.NullString
		PublishedAt     sql.NullTime
		FeaturedImage   sql.NullString
		Views           int
		Author          sql.NullString
		MetaDescription sql.NullString
		Category        sql.NullString
		PlaceSlug       sql.NullString
	}

	err := h.aethraDB.QueryRow(query, domain, slug).Scan(
		&article.ID, &article.Title, &article.Slug,
		&article.Summary, &article.Content, &article.PublishedAt,
		&article.FeaturedImage, &article.Views, &article.Author,
		&article.MetaDescription, &article.Category, &article.PlaceSlug)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Article not found"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch article"})
		return
	}

	// Increment view count
	h.aethraDB.Exec("UPDATE articles SET views = views + 1 WHERE id = $1", article.ID)

	// Process content to update image URLs
	content := transformContentURLs(article.Content.String)

	// Process featured image URL
	featuredImage := transformImageURL(article.FeaturedImage.String)

	c.JSON(http.StatusOK, gin.H{
		"id":               article.ID,
		"title":            article.Title,
		"slug":             article.Slug,
		"summary":          article.Summary.String,
		"content":          content,
		"published_at":     article.PublishedAt.Time,
		"featured_image":   featuredImage,
		"views":            article.Views,
		"author":           article.Author.String,
		"meta_description": article.MetaDescription.String,
		"category":         article.Category.String,
		"place_slug":       article.PlaceSlug.String,
	})
}

// GetCategories retrieves all distinct categories for a domain
// GET /api/public/categories
func (h *ContentHandler) GetCategories(c *gin.Context) {
	if h.aethraDB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Aethra database not configured",
			"hint":  "Set AETHRA_DB_* environment variables",
		})
		return
	}

	domain := h.getDomain(c)

	if domain == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Domain parameter is required"})
		return
	}

	query := `
		SELECT DISTINCT LOWER(TRIM(a.category)) as category
		FROM articles a
		JOIN websites w ON a.website_id = w.id
		WHERE w.domain = $1
		  AND a.status = 'published'
		  AND a.category IS NOT NULL
		  AND TRIM(a.category) != ''
		ORDER BY category
	`

	rows, err := h.aethraDB.Query(query, domain)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch categories",
			"details": err.Error(),
		})
		return
	}
	defer rows.Close()

	categories := []string{}
	for rows.Next() {
		var category string
		if err := rows.Scan(&category); err != nil {
			continue
		}
		categories = append(categories, category)
	}

	c.JSON(http.StatusOK, gin.H{
		"categories": categories,
		"total":      len(categories),
	})
}

// GetContent retrieves static content by slug (for future CMS integration)
// GET /api/content/:slug
func (h *ContentHandler) GetContent(c *gin.Context) {
	slug := c.Param("slug")

	// For now, return placeholder
	// Later this will fetch from the CMS
	content := map[string]interface{}{
		"slug":    slug,
		"title":   "Content Title",
		"content": "This content will be fetched from the CMS",
	}

	c.JSON(http.StatusOK, content)
}
