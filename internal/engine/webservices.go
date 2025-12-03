// Package engine - Web Service Client for External APIs
package engine

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/aethra/genesis/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WebServiceClient manages web service calls
type WebServiceClient struct {
	db            *gorm.DB
	clients       map[uuid.UUID]*http.Client
	mu            sync.RWMutex
	encryptionKey []byte
	rateLimiters  map[uuid.UUID]*RateLimiter
}

// RateLimiter simple token bucket rate limiter
type RateLimiter struct {
	tokens     int
	maxTokens  int
	refillRate int
	lastRefill time.Time
	mu         sync.Mutex
}

// WebServiceRequest represents a request to be made
type WebServiceRequest struct {
	EndpointID  uuid.UUID
	PathParams  map[string]string
	QueryParams map[string]string
	Headers     map[string]string
	Body        interface{}
}

// WebServiceResponse represents the response from a web service
type WebServiceResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	ParsedBody interface{}
	Duration   time.Duration
}

// NewWebServiceClient creates a new web service client
func NewWebServiceClient(db *gorm.DB, encryptionKey string) *WebServiceClient {
	key := make([]byte, 32)
	copy(key, []byte(encryptionKey))

	return &WebServiceClient{
		db:            db,
		clients:       make(map[uuid.UUID]*http.Client),
		encryptionKey: key,
		rateLimiters:  make(map[uuid.UUID]*RateLimiter),
	}
}

// Call makes a request to a web service endpoint
func (wsc *WebServiceClient) Call(ctx context.Context, req *WebServiceRequest) (*WebServiceResponse, error) {
	// Load endpoint configuration
	var endpoint models.WebServiceEndpoint
	if err := wsc.db.Preload("WebService").First(&endpoint, "id = ?", req.EndpointID).Error; err != nil {
		return nil, fmt.Errorf("endpoint not found: %w", err)
	}

	if !endpoint.IsActive {
		return nil, fmt.Errorf("endpoint is not active")
	}

	if endpoint.WebService == nil {
		return nil, fmt.Errorf("web service not found for endpoint")
	}

	if !endpoint.WebService.IsActive {
		return nil, fmt.Errorf("web service is not active")
	}

	// Check rate limit
	if err := wsc.checkRateLimit(endpoint.WebService); err != nil {
		return nil, err
	}

	// Build request
	httpReq, err := wsc.buildRequest(ctx, &endpoint, req)
	if err != nil {
		return nil, err
	}

	// Get or create HTTP client
	client := wsc.getClient(endpoint.WebService)

	// Execute with retries
	start := time.Now()
	resp, err := wsc.executeWithRetry(client, httpReq, endpoint.WebService)
	duration := time.Since(start)

	if err != nil {
		wsc.updateCallResult(endpoint.WebService.ID, false, err.Error())
		return nil, err
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse response
	var parsedBody interface{}
	switch endpoint.ResponseType {
	case "json":
		if err := json.Unmarshal(body, &parsedBody); err != nil {
			parsedBody = nil // Parsing failed, keep raw body
		}
	case "xml":
		if err := xml.Unmarshal(body, &parsedBody); err != nil {
			parsedBody = nil
		}
	}

	// Check success
	success := wsc.isSuccessful(resp.StatusCode, &endpoint)
	if !success {
		wsc.updateCallResult(endpoint.WebService.ID, false, string(body))
	} else {
		wsc.updateCallResult(endpoint.WebService.ID, true, "")
	}

	return &WebServiceResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       body,
		ParsedBody: parsedBody,
		Duration:   duration,
	}, nil
}

// CallByCode makes a request using service and endpoint codes
func (wsc *WebServiceClient) CallByCode(ctx context.Context, tenantID uuid.UUID, serviceCode, endpointCode string, req *WebServiceRequest) (*WebServiceResponse, error) {
	var service models.WebService
	if err := wsc.db.First(&service, "tenant_id = ? AND code = ?", tenantID, serviceCode).Error; err != nil {
		return nil, fmt.Errorf("web service not found: %w", err)
	}

	var endpoint models.WebServiceEndpoint
	if err := wsc.db.First(&endpoint, "web_service_id = ? AND code = ?", service.ID, endpointCode).Error; err != nil {
		return nil, fmt.Errorf("endpoint not found: %w", err)
	}

	req.EndpointID = endpoint.ID
	return wsc.Call(ctx, req)
}

// buildRequest constructs the HTTP request
func (wsc *WebServiceClient) buildRequest(ctx context.Context, endpoint *models.WebServiceEndpoint, req *WebServiceRequest) (*http.Request, error) {
	service := endpoint.WebService

	// Build URL
	fullURL := service.BaseURL + wsc.resolvePath(endpoint.Path, req.PathParams)

	// Add query parameters
	if len(req.QueryParams) > 0 || endpoint.QueryParams != nil {
		parsedURL, err := url.Parse(fullURL)
		if err != nil {
			return nil, fmt.Errorf("invalid URL: %w", err)
		}
		query := parsedURL.Query()

		// Add default query params from endpoint config
		if endpoint.QueryParams != nil {
			for k, v := range endpoint.QueryParams {
				if strVal, ok := v.(string); ok {
					query.Set(k, strVal)
				}
			}
		}

		// Add request-specific query params (override defaults)
		for k, v := range req.QueryParams {
			query.Set(k, v)
		}

		parsedURL.RawQuery = query.Encode()
		fullURL = parsedURL.String()
	}

	// Build body
	var bodyReader io.Reader
	if req.Body != nil && (endpoint.Method == "POST" || endpoint.Method == "PUT" || endpoint.Method == "PATCH") {
		bodyBytes, err := wsc.buildBody(endpoint, req.Body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create request
	httpReq, err := http.NewRequestWithContext(ctx, endpoint.Method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	wsc.setHeaders(httpReq, service, endpoint, req.Headers)

	// Set authentication
	if err := wsc.setAuth(httpReq, service); err != nil {
		return nil, err
	}

	return httpReq, nil
}

// resolvePath replaces path parameters like /users/{id} with actual values
func (wsc *WebServiceClient) resolvePath(path string, params map[string]string) string {
	result := path
	re := regexp.MustCompile(`\{(\w+)\}`)
	matches := re.FindAllStringSubmatch(path, -1)

	for _, match := range matches {
		if len(match) == 2 {
			paramName := match[1]
			if value, exists := params[paramName]; exists {
				result = strings.Replace(result, match[0], url.PathEscape(value), 1)
			}
		}
	}

	return result
}

// buildBody constructs the request body
func (wsc *WebServiceClient) buildBody(endpoint *models.WebServiceEndpoint, body interface{}) ([]byte, error) {
	switch endpoint.RequestBodyType {
	case "json":
		return json.Marshal(body)
	case "form":
		if formData, ok := body.(map[string]string); ok {
			values := url.Values{}
			for k, v := range formData {
				values.Set(k, v)
			}
			return []byte(values.Encode()), nil
		}
		return nil, fmt.Errorf("form body must be map[string]string")
	case "xml":
		return xml.Marshal(body)
	case "raw":
		if str, ok := body.(string); ok {
			return []byte(str), nil
		}
		if bytes, ok := body.([]byte); ok {
			return bytes, nil
		}
		return nil, fmt.Errorf("raw body must be string or []byte")
	default:
		return json.Marshal(body)
	}
}

// setHeaders sets all required headers
func (wsc *WebServiceClient) setHeaders(req *http.Request, service *models.WebService, endpoint *models.WebServiceEndpoint, customHeaders map[string]string) {
	// Default Content-Type based on request body type
	if endpoint.RequestBodyType == "json" {
		req.Header.Set("Content-Type", "application/json")
	} else if endpoint.RequestBodyType == "form" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else if endpoint.RequestBodyType == "xml" {
		req.Header.Set("Content-Type", "application/xml")
	}

	// Accept header based on response type
	switch endpoint.ResponseType {
	case "json":
		req.Header.Set("Accept", "application/json")
	case "xml":
		req.Header.Set("Accept", "application/xml")
	}

	// Service-level default headers
	if service.DefaultHeaders != nil {
		for k, v := range service.DefaultHeaders {
			if strVal, ok := v.(string); ok {
				req.Header.Set(k, strVal)
			}
		}
	}

	// Endpoint-specific headers
	if endpoint.Headers != nil {
		for k, v := range endpoint.Headers {
			if strVal, ok := v.(string); ok {
				req.Header.Set(k, strVal)
			}
		}
	}

	// Custom headers from request (highest priority)
	for k, v := range customHeaders {
		req.Header.Set(k, v)
	}
}

// setAuth sets authentication on the request
func (wsc *WebServiceClient) setAuth(req *http.Request, service *models.WebService) error {
	if service.AuthType == "" || service.AuthType == "none" {
		return nil
	}

	authConfig := service.AuthConfig
	if authConfig == nil {
		return nil
	}

	switch service.AuthType {
	case "basic":
		username, _ := authConfig["username"].(string)
		passwordEnc, _ := authConfig["password_encrypted"].(string)
		password, err := wsc.decrypt(passwordEnc)
		if err != nil {
			return fmt.Errorf("failed to decrypt basic auth password: %w", err)
		}
		req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+password)))

	case "bearer":
		tokenEnc, _ := authConfig["token_encrypted"].(string)
		token, err := wsc.decrypt(tokenEnc)
		if err != nil {
			return fmt.Errorf("failed to decrypt bearer token: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)

	case "api_key":
		headerName, _ := authConfig["header_name"].(string)
		keyEnc, _ := authConfig["key_encrypted"].(string)
		key, err := wsc.decrypt(keyEnc)
		if err != nil {
			return fmt.Errorf("failed to decrypt API key: %w", err)
		}
		if headerName == "" {
			headerName = "X-API-Key"
		}
		req.Header.Set(headerName, key)

	case "custom_header":
		if headers, ok := authConfig["headers"].(map[string]interface{}); ok {
			for k, v := range headers {
				if strVal, ok := v.(string); ok {
					req.Header.Set(k, strVal)
				}
			}
		}
	}

	return nil
}

// getClient returns or creates an HTTP client for the service
func (wsc *WebServiceClient) getClient(service *models.WebService) *http.Client {
	wsc.mu.RLock()
	if client, exists := wsc.clients[service.ID]; exists {
		wsc.mu.RUnlock()
		return client
	}
	wsc.mu.RUnlock()

	// Create new client
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: !service.SSLVerify,
		},
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(service.TimeoutRead) * time.Millisecond,
	}

	wsc.mu.Lock()
	wsc.clients[service.ID] = client
	wsc.mu.Unlock()

	return client
}

// executeWithRetry executes the request with retry logic
func (wsc *WebServiceClient) executeWithRetry(client *http.Client, req *http.Request, service *models.WebService) (*http.Response, error) {
	maxAttempts := 1
	if service.RetryEnabled {
		maxAttempts = service.RetryMaxAttempts
		if maxAttempts < 1 {
			maxAttempts = 3
		}
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Clone request for retry (body needs to be re-readable)
		reqCopy := req.Clone(req.Context())
		if req.Body != nil {
			body, _ := io.ReadAll(req.Body)
			req.Body = io.NopCloser(bytes.NewReader(body))
			reqCopy.Body = io.NopCloser(bytes.NewReader(body))
		}

		resp, err := client.Do(reqCopy)
		if err != nil {
			lastErr = err
			if attempt < maxAttempts {
				time.Sleep(time.Duration(service.RetryDelayMs) * time.Millisecond)
				continue
			}
			return nil, err
		}

		// Check if we should retry based on status code
		if service.RetryEnabled && attempt < maxAttempts && wsc.shouldRetry(resp.StatusCode, service) {
			resp.Body.Close()
			time.Sleep(time.Duration(service.RetryDelayMs) * time.Millisecond)
			continue
		}

		return resp, nil
	}

	return nil, lastErr
}

// shouldRetry checks if the status code should trigger a retry
func (wsc *WebServiceClient) shouldRetry(statusCode int, service *models.WebService) bool {
	if service.RetryOnStatusCodes == nil {
		// Default retry codes
		return statusCode == 500 || statusCode == 502 || statusCode == 503 || statusCode == 504
	}

	for _, code := range service.RetryOnStatusCodes {
		if int(code) == statusCode {
			return true
		}
	}
	return false
}

// isSuccessful checks if the response indicates success
func (wsc *WebServiceClient) isSuccessful(statusCode int, endpoint *models.WebServiceEndpoint) bool {
	if endpoint.SuccessStatusCodes != nil && len(endpoint.SuccessStatusCodes) > 0 {
		for _, code := range endpoint.SuccessStatusCodes {
			if int(code) == statusCode {
				return true
			}
		}
		return false
	}

	// Default: 2xx is success
	return statusCode >= 200 && statusCode < 300
}

// checkRateLimit enforces rate limiting
func (wsc *WebServiceClient) checkRateLimit(service *models.WebService) error {
	if service.RateLimitRequests == nil || service.RateLimitWindowSeconds == nil {
		return nil
	}

	wsc.mu.Lock()
	limiter, exists := wsc.rateLimiters[service.ID]
	if !exists {
		limiter = &RateLimiter{
			tokens:     *service.RateLimitRequests,
			maxTokens:  *service.RateLimitRequests,
			refillRate: *service.RateLimitRequests,
			lastRefill: time.Now(),
		}
		wsc.rateLimiters[service.ID] = limiter
	}
	wsc.mu.Unlock()

	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	// Refill tokens
	now := time.Now()
	elapsed := now.Sub(limiter.lastRefill).Seconds()
	windowSeconds := float64(*service.RateLimitWindowSeconds)

	if elapsed >= windowSeconds {
		limiter.tokens = limiter.maxTokens
		limiter.lastRefill = now
	}

	// Check if we have tokens
	if limiter.tokens <= 0 {
		return fmt.Errorf("rate limit exceeded, try again later")
	}

	limiter.tokens--
	return nil
}

// updateCallResult updates the last call result for a service
func (wsc *WebServiceClient) updateCallResult(serviceID uuid.UUID, success bool, errorMsg string) {
	now := time.Now()
	result := "success"
	if !success {
		result = "failed"
	}

	wsc.db.Model(&models.WebService{}).
		Where("id = ?", serviceID).
		Updates(map[string]interface{}{
			"last_called_at":   &now,
			"last_call_result": result,
			"last_call_error":  errorMsg,
		})
}

// decrypt decrypts encrypted values (same as ConnectionManager)
func (wsc *WebServiceClient) decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	// Use shared decryption from ConnectionManager
	// For now, implement inline (in production, extract to shared crypto package)
	cm := &ConnectionManager{encryptionKey: wsc.encryptionKey}
	return cm.decrypt(ciphertext)
}

// TestEndpoint tests a web service endpoint
func (wsc *WebServiceClient) TestEndpoint(ctx context.Context, endpointID uuid.UUID) (*WebServiceResponse, error) {
	return wsc.Call(ctx, &WebServiceRequest{
		EndpointID: endpointID,
	})
}

// ListServices lists all web services for a tenant
func (wsc *WebServiceClient) ListServices(tenantID uuid.UUID) ([]models.WebService, error) {
	var services []models.WebService
	if err := wsc.db.Where("tenant_id = ?", tenantID).Preload("Endpoints").Find(&services).Error; err != nil {
		return nil, err
	}
	return services, nil
}

// GetService retrieves a web service by ID
func (wsc *WebServiceClient) GetService(serviceID uuid.UUID) (*models.WebService, error) {
	var service models.WebService
	if err := wsc.db.Preload("Endpoints").First(&service, "id = ?", serviceID).Error; err != nil {
		return nil, err
	}
	return &service, nil
}
