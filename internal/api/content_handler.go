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

// =====================================================
// ADMIN ARTICLE MANAGEMENT (CRUD operations)
// =====================================================

// ArticleInput represents the input for creating/updating articles
type ArticleInput struct {
	Title           string   `json:"title" binding:"required"`
	Slug            string   `json:"slug"`
	Summary         string   `json:"summary"`
	Content         string   `json:"content"`
	FeaturedImage   string   `json:"featured_image"`
	Category        string   `json:"category"`
	CategoryID      *int     `json:"category_id"`
	Status          string   `json:"status"`
	Tags            []string `json:"tags"`
	MetaTitle       string   `json:"meta_title"`
	MetaDescription string   `json:"meta_description"`
	EventDate       string   `json:"event_date"`
	EventLocation   string   `json:"event_location"`
	PlaceSlug       string   `json:"place_slug"`
	WebsiteID       int      `json:"website_id"`
}

// GetAdminArticles retrieves all articles for admin (including drafts)
// GET /api/admin/articles
func (h *ContentHandler) GetAdminArticles(c *gin.Context) {
	if h.aethraDB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Aethra database not configured"})
		return
	}

	// Get filters from query params
	status := c.Query("status")
	websiteID := c.Query("website_id")
	category := c.Query("category")
	search := c.Query("search")

	limit := 50
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

	// Build query
	query := `
		SELECT a.id, a.title, a.slug, a.summary, a.status, a.category,
		       a.featured_image, a.views, a.created_at, a.updated_at, a.published_at,
		       w.name as website_name, w.id as website_id,
		       COALESCE(u.username, 'Unknown') as author
		FROM articles a
		JOIN websites w ON a.website_id = w.id
		LEFT JOIN users u ON a.author_id = u.id
		WHERE 1=1
	`

	args := []interface{}{}
	paramCount := 0

	if websiteID != "" {
		paramCount++
		query += fmt.Sprintf(` AND a.website_id = $%d`, paramCount)
		args = append(args, websiteID)
	}

	if status != "" {
		paramCount++
		query += fmt.Sprintf(` AND a.status = $%d`, paramCount)
		args = append(args, status)
	}

	if category != "" {
		paramCount++
		query += fmt.Sprintf(` AND LOWER(a.category) = LOWER($%d)`, paramCount)
		args = append(args, category)
	}

	if search != "" {
		paramCount++
		query += fmt.Sprintf(` AND (LOWER(a.title) LIKE LOWER($%d) OR LOWER(a.summary) LIKE LOWER($%d))`, paramCount, paramCount)
		args = append(args, "%"+search+"%")
	}

	paramCount++
	limitParam := paramCount
	paramCount++
	offsetParam := paramCount

	query += fmt.Sprintf(` ORDER BY a.updated_at DESC LIMIT $%d OFFSET $%d`, limitParam, offsetParam)
	args = append(args, limit, offset)

	rows, err := h.aethraDB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch articles", "details": err.Error()})
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
			Status        sql.NullString
			Category      sql.NullString
			FeaturedImage sql.NullString
			Views         int
			CreatedAt     sql.NullTime
			UpdatedAt     sql.NullTime
			PublishedAt   sql.NullTime
			WebsiteName   string
			WebsiteID     int
			Author        sql.NullString
		}

		err := rows.Scan(&article.ID, &article.Title, &article.Slug,
			&article.Summary, &article.Status, &article.Category,
			&article.FeaturedImage, &article.Views, &article.CreatedAt,
			&article.UpdatedAt, &article.PublishedAt, &article.WebsiteName,
			&article.WebsiteID, &article.Author)
		if err != nil {
			continue
		}

		articles = append(articles, map[string]interface{}{
			"id":             article.ID,
			"title":          article.Title,
			"slug":           article.Slug,
			"summary":        article.Summary.String,
			"status":         article.Status.String,
			"category":       article.Category.String,
			"featured_image": transformImageURL(article.FeaturedImage.String),
			"views":          article.Views,
			"created_at":     article.CreatedAt.Time,
			"updated_at":     article.UpdatedAt.Time,
			"published_at":   article.PublishedAt.Time,
			"website_name":   article.WebsiteName,
			"website_id":     article.WebsiteID,
			"author":         article.Author.String,
		})
	}

	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM articles a WHERE 1=1`
	countArgs := []interface{}{}
	countParamCount := 0

	if websiteID != "" {
		countParamCount++
		countQuery += fmt.Sprintf(` AND a.website_id = $%d`, countParamCount)
		countArgs = append(countArgs, websiteID)
	}
	if status != "" {
		countParamCount++
		countQuery += fmt.Sprintf(` AND a.status = $%d`, countParamCount)
		countArgs = append(countArgs, status)
	}

	h.aethraDB.QueryRow(countQuery, countArgs...).Scan(&total)

	c.JSON(http.StatusOK, gin.H{
		"articles": articles,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// GetAdminArticle retrieves a single article for editing
// GET /api/admin/articles/:id
func (h *ContentHandler) GetAdminArticle(c *gin.Context) {
	if h.aethraDB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Aethra database not configured"})
		return
	}

	id := c.Param("id")

	query := `
		SELECT a.id, a.title, a.slug, a.summary, a.content, a.status,
		       a.featured_image, a.category, a.category_id, a.tags,
		       a.meta_title, a.meta_description, a.event_date, a.event_location,
		       a.place_slug, a.website_id, a.views, a.created_at, a.updated_at, a.published_at,
		       COALESCE(u.username, 'Unknown') as author, a.author_id
		FROM articles a
		LEFT JOIN users u ON a.author_id = u.id
		WHERE a.id = $1
	`

	var article struct {
		ID              int
		Title           string
		Slug            string
		Summary         sql.NullString
		Content         sql.NullString
		Status          sql.NullString
		FeaturedImage   sql.NullString
		Category        sql.NullString
		CategoryID      sql.NullInt64
		Tags            sql.NullString
		MetaTitle       sql.NullString
		MetaDescription sql.NullString
		EventDate       sql.NullTime
		EventLocation   sql.NullString
		PlaceSlug       sql.NullString
		WebsiteID       sql.NullInt64
		Views           int
		CreatedAt       sql.NullTime
		UpdatedAt       sql.NullTime
		PublishedAt     sql.NullTime
		Author          sql.NullString
		AuthorID        sql.NullInt64
	}

	err := h.aethraDB.QueryRow(query, id).Scan(
		&article.ID, &article.Title, &article.Slug, &article.Summary, &article.Content,
		&article.Status, &article.FeaturedImage, &article.Category, &article.CategoryID,
		&article.Tags, &article.MetaTitle, &article.MetaDescription, &article.EventDate,
		&article.EventLocation, &article.PlaceSlug, &article.WebsiteID, &article.Views,
		&article.CreatedAt, &article.UpdatedAt, &article.PublishedAt, &article.Author, &article.AuthorID)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Article not found"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch article", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":               article.ID,
		"title":            article.Title,
		"slug":             article.Slug,
		"summary":          article.Summary.String,
		"content":          article.Content.String,
		"status":           article.Status.String,
		"featured_image":   article.FeaturedImage.String,
		"category":         article.Category.String,
		"category_id":      article.CategoryID.Int64,
		"tags":             article.Tags.String,
		"meta_title":       article.MetaTitle.String,
		"meta_description": article.MetaDescription.String,
		"event_date":       article.EventDate.Time,
		"event_location":   article.EventLocation.String,
		"place_slug":       article.PlaceSlug.String,
		"website_id":       article.WebsiteID.Int64,
		"views":            article.Views,
		"created_at":       article.CreatedAt.Time,
		"updated_at":       article.UpdatedAt.Time,
		"published_at":     article.PublishedAt.Time,
		"author":           article.Author.String,
		"author_id":        article.AuthorID.Int64,
	})
}

// CreateArticle creates a new article
// POST /api/admin/articles
func (h *ContentHandler) CreateArticle(c *gin.Context) {
	if h.aethraDB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Aethra database not configured"})
		return
	}

	var input ArticleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input", "details": err.Error()})
		return
	}

	// Generate slug if not provided
	if input.Slug == "" {
		input.Slug = generateSlug(input.Title)
	}

	// Default status
	if input.Status == "" {
		input.Status = "draft"
	}

	// Default website ID (use Intranet for now)
	if input.WebsiteID == 0 {
		input.WebsiteID = 4 // Intranet
	}

	query := `
		INSERT INTO articles (
			title, slug, summary, content, featured_image, category, category_id,
			status, meta_title, meta_description, event_date, event_location,
			place_slug, website_id, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			NULLIF($11, '')::timestamp with time zone, $12, $13, $14, NOW(), NOW()
		) RETURNING id
	`

	var articleID int
	err := h.aethraDB.QueryRow(query,
		input.Title, input.Slug, input.Summary, input.Content, input.FeaturedImage,
		input.Category, input.CategoryID, input.Status, input.MetaTitle, input.MetaDescription,
		input.EventDate, input.EventLocation, input.PlaceSlug, input.WebsiteID).Scan(&articleID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create article", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":      articleID,
		"message": "Article created successfully",
	})
}

// UpdateArticle updates an existing article
// PUT /api/admin/articles/:id
func (h *ContentHandler) UpdateArticle(c *gin.Context) {
	if h.aethraDB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Aethra database not configured"})
		return
	}

	id := c.Param("id")

	var input ArticleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input", "details": err.Error()})
		return
	}

	// Check if article exists
	var exists bool
	h.aethraDB.QueryRow("SELECT EXISTS(SELECT 1 FROM articles WHERE id = $1)", id).Scan(&exists)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Article not found"})
		return
	}

	query := `
		UPDATE articles SET
			title = $1, slug = $2, summary = $3, content = $4, featured_image = $5,
			category = $6, category_id = $7, status = $8, meta_title = $9, meta_description = $10,
			event_date = NULLIF($11, '')::timestamp with time zone, event_location = $12,
			place_slug = $13, updated_at = NOW()
		WHERE id = $14
	`

	_, err := h.aethraDB.Exec(query,
		input.Title, input.Slug, input.Summary, input.Content, input.FeaturedImage,
		input.Category, input.CategoryID, input.Status, input.MetaTitle, input.MetaDescription,
		input.EventDate, input.EventLocation, input.PlaceSlug, id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update article", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Article updated successfully"})
}

// PublishArticle publishes an article
// POST /api/admin/articles/:id/publish
func (h *ContentHandler) PublishArticle(c *gin.Context) {
	if h.aethraDB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Aethra database not configured"})
		return
	}

	id := c.Param("id")

	_, err := h.aethraDB.Exec(`
		UPDATE articles SET status = 'published', published_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`, id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to publish article"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Article published successfully"})
}

// UnpublishArticle sets article to draft
// POST /api/admin/articles/:id/unpublish
func (h *ContentHandler) UnpublishArticle(c *gin.Context) {
	if h.aethraDB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Aethra database not configured"})
		return
	}

	id := c.Param("id")

	_, err := h.aethraDB.Exec(`
		UPDATE articles SET status = 'draft', updated_at = NOW()
		WHERE id = $1
	`, id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unpublish article"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Article unpublished successfully"})
}

// DeleteArticle deletes an article
// DELETE /api/admin/articles/:id
func (h *ContentHandler) DeleteArticle(c *gin.Context) {
	if h.aethraDB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Aethra database not configured"})
		return
	}

	id := c.Param("id")

	result, err := h.aethraDB.Exec("DELETE FROM articles WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete article"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Article not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Article deleted successfully"})
}

// GetWebsites retrieves all websites for dropdown
// GET /api/admin/websites
func (h *ContentHandler) GetWebsites(c *gin.Context) {
	if h.aethraDB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Aethra database not configured"})
		return
	}

	rows, err := h.aethraDB.Query("SELECT id, name, domain FROM websites ORDER BY name")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch websites"})
		return
	}
	defer rows.Close()

	websites := []map[string]interface{}{}
	for rows.Next() {
		var id int
		var name, domain string
		if err := rows.Scan(&id, &name, &domain); err == nil {
			websites = append(websites, map[string]interface{}{
				"id":     id,
				"name":   name,
				"domain": domain,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"websites": websites})
}

// GetCategoriesList retrieves all categories for dropdown
// GET /api/admin/categories
func (h *ContentHandler) GetCategoriesList(c *gin.Context) {
	if h.aethraDB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Aethra database not configured"})
		return
	}

	rows, err := h.aethraDB.Query("SELECT id, name, slug FROM categories ORDER BY name")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch categories"})
		return
	}
	defer rows.Close()

	categories := []map[string]interface{}{}
	for rows.Next() {
		var id int
		var name, slug string
		if err := rows.Scan(&id, &name, &slug); err == nil {
			categories = append(categories, map[string]interface{}{
				"id":   id,
				"name": name,
				"slug": slug,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"categories": categories})
}

// generateSlug creates a URL-friendly slug from title
func generateSlug(title string) string {
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")
	// Remove special characters (simple approach)
	var result strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	slug = result.String()
	// Remove multiple dashes
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	return strings.Trim(slug, "-")
}
