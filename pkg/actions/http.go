// Package actions implements various action handlers for the VPR engine.
// This file specifically contains the implementation of the HTTP request action,
// which is one of the core actions defined in the DSL specification.
package actions

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"time"
	
	"vpr/pkg/context"
	"vpr/pkg/extractors"
	"vpr/pkg/poc"
	"vpr/pkg/utils"
)

// HTTPResponse represents the result of an HTTP request
type HTTPResponse struct {
	StatusCode    int                 `json:"status_code"`
	Headers       map[string][]string `json:"headers"`
	Body          string              `json:"body"`
	ResponseTime  float64             `json:"response_time_ms"`
	ContentLength int64               `json:"content_length"`
	URL           string              `json:"url"`
}

// httpRequestHandler executes an HTTP request
func httpRequestHandler(ctx *context.ExecutionContext, action *poc.Action) (interface{}, error) {
	if action.Type != "http_request" {
		return nil, fmt.Errorf("invalid action type for httpRequestHandler: %s", action.Type)
	}
	
	// 处理认证上下文
	if action.AuthenticationContext != "" {
		// 如果指定了认证上下文，则在此请求中使用该上下文
		if err := ctx.SetAuthenticationContext(action.AuthenticationContext); err != nil {
			return nil, fmt.Errorf("failed to set authentication context: %w", err)
		}
	}
	
	// Validate required parameters
	if action.Request == nil {
		return nil, fmt.Errorf("http_request action requires a 'request' section")
	}
	
	if action.Request.URL == "" {
		return nil, fmt.Errorf("http_request action requires a URL")
	}
	
	// Get HTTP client
	client, err := utils.GetHTTPClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get HTTP client: %w", err)
	}
	
	// Build request
	req, err := buildHTTPRequest(ctx, action)
	if err != nil {
		return nil, fmt.Errorf("failed to build HTTP request: %w", err)
	}
	
	// Log the request
	logHTTPRequest(req, action)
	
	// Execute the request
	startTime := time.Now()
	resp, err := client.Do(req)
	elapsedMs := float64(time.Since(startTime).Microseconds()) / 1000.0
	
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Process the response
	httpResp, err := processHTTPResponse(resp, elapsedMs)
	if err != nil {
		return nil, fmt.Errorf("failed to process HTTP response: %w", err)
	}
	
	// Handle response actions if any
	if action.ResponseActions != nil && len(action.ResponseActions) > 0 {
		// Create a map to store the HTTP response data for extractors
		respData := map[string]interface{}{
			"status_code":    httpResp.StatusCode,
			"headers":        httpResp.Headers,
			"body":           httpResp.Body,
			"response_time":  httpResp.ResponseTime,
			"content_length": httpResp.ContentLength,
			"url":            httpResp.URL,
		}
		
		log.Printf("DEBUG: Processing %d response actions", len(action.ResponseActions))
		
		// Get the extractor registry
		extractorRegistryObj := ctx.GetExtractorRegistry()
		if extractorRegistryObj == nil {
			return nil, fmt.Errorf("extractor registry not available")
		}
		
		// Type assert to the correct type
		extractorRegistry, ok := extractorRegistryObj.(*extractors.ExtractorRegistry)
		if !ok {
			return nil, fmt.Errorf("extractor registry is not of correct type")
		}
		
		// Execute each response action
		for i, responseAction := range action.ResponseActions {
			// Get the handler for this extractor type
			handler := extractorRegistry.Get(responseAction.Type)
			if handler == nil {
				return nil, fmt.Errorf("unknown extractor type: %s", responseAction.Type)
			}
			
			log.Printf("DEBUG: Executing response action %d: %s", i+1, responseAction.Type)
			
			// Execute the extractor with the HTTP response data
			// Need to get address of responseAction to convert from value to pointer type
			_, err := handler(ctx, &responseAction, respData)
			if err != nil {
				return nil, fmt.Errorf("response action failed: %w", err)
			}
		}
	}
	
	// Store the HTTP response in context for later checks
	lastResponseVar := map[string]interface{}{
		"status_code":    httpResp.StatusCode,
		"headers":        httpResp.Headers,
		"body":           httpResp.Body,
		"response_time":  httpResp.ResponseTime,
		"content_length": httpResp.ContentLength,
		"url":            httpResp.URL,
	}
	
	if err := ctx.SetVariable("last_http_response", lastResponseVar); err != nil {
		log.Printf("WARNING: Failed to store last_http_response in context: %v", err)
		// Continue even if setting the variable fails
	} else {
		log.Printf("DEBUG: Stored HTTP response in context as last_http_response (status=%d, url=%s)", 
			httpResp.StatusCode, httpResp.URL)
	}
	
	return httpResp, nil
}

// buildHTTPRequest constructs an HTTP request from action parameters
func buildHTTPRequest(ctx *context.ExecutionContext, action *poc.Action) (*http.Request, error) {
	// Resolve the URL - substitute variables if needed
	resolvedURL, err := ctx.Substitute(action.Request.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve URL: %w", err)
	}
	
	// Determine the method (default to GET)
	method := "GET"
	if action.Request.Method != "" {
		method = strings.ToUpper(action.Request.Method)
	}
	
	// Prepare the request body if applicable
	var body io.Reader
	if action.Request.Body != nil {
		// Convert body to string depending on type
		var bodyStr string
		
		switch v := action.Request.Body.(type) {
		case string:
			bodyStr = v
		case map[string]interface{}, []interface{}:
			// Handle complex body types based on body_type here
			// For now a simple string representation
			bodyStr = fmt.Sprintf("%v", v)
		default:
			bodyStr = fmt.Sprintf("%v", v)
		}
		
		// Resolve any variables in body
		resolvedBody, err := ctx.Substitute(bodyStr)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve request body: %w", err)
		}
		body = bytes.NewBufferString(resolvedBody)
	}
	
	// Create the request
	req, err := http.NewRequest(method, resolvedURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	
	// Add headers if present
	if action.Request.Headers != nil {
		for key, value := range action.Request.Headers {
			// Resolve any variables in header values
			resolvedValue, err := ctx.Substitute(value)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve header '%s': %w", key, err)
			}
			req.Header.Set(key, resolvedValue)
		}
	}
	
	// Apply authentication if an authentication context is specified
	if action.AuthenticationContext != "" {
		if err := ctx.ApplyAuthentication(req, action.AuthenticationContext); err != nil {
			return nil, fmt.Errorf("failed to apply authentication for context '%s': %w", 
				action.AuthenticationContext, err)
		}
	} else {
		// Apply the current active authentication context (if any)
		if err := ctx.ApplyAuthentication(req, ""); err != nil {
			return nil, fmt.Errorf("failed to apply active authentication context: %w", err)
		}
	}
	
	// Set default headers if not already set
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "VPR-PoCRunner/1.0")
	}
	
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "*/*")
	}
	
	// Handle special body types
	if action.Request.BodyType != "" {
		switch strings.ToLower(action.Request.BodyType) {
		case "json":
			if req.Header.Get("Content-Type") == "" {
				req.Header.Set("Content-Type", "application/json")
			}
		case "form":
			if req.Header.Get("Content-Type") == "" {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
		case "multipart":
			// Handle multipart form data
			err := handleMultipartRequest(ctx, action, req)
			if err != nil {
				return nil, fmt.Errorf("failed to process multipart form: %w", err)
			}
			// Content-Type is set by the multipart processor
		}
	}
	
	return req, nil
}

// processHTTPResponse converts an HTTP response to our internal representation
func processHTTPResponse(resp *http.Response, elapsedMs float64) (*HTTPResponse, error) {
	// Read the body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	
	// Create the response object
	httpResp := &HTTPResponse{
		StatusCode:    resp.StatusCode,
		Headers:       resp.Header,
		Body:          string(bodyBytes),
		ResponseTime:  elapsedMs,
		ContentLength: resp.ContentLength,
		URL:           resp.Request.URL.String(),
	}
	
	// Log the response
	logHTTPResponse(httpResp)
	
	return httpResp, nil
}

// logHTTPRequest logs an HTTP request at debug level
func logHTTPRequest(req *http.Request, action *poc.Action) {
	slog.Debug("HTTP Request",
		"method", req.Method,
		"url", req.URL.String(),
		"headers", fmt.Sprintf("%v", req.Header),
		"has_body", action.Request.Body != nil,
	)
}

// logHTTPResponse logs an HTTP response at debug level
func logHTTPResponse(resp *HTTPResponse) {
	// Prepare a preview of the body (truncated if large)
	bodyPreview := resp.Body
	if len(bodyPreview) > 500 {
		bodyPreview = bodyPreview[:500] + "... [truncated]"
	}
	
	// Log the response at debug level
	slog.Debug("HTTP response details", 
		"status_code", resp.StatusCode,
		"content_length", resp.ContentLength,
		"response_time_ms", resp.ResponseTime,
		"body_preview", bodyPreview,
	)
}

// 注：init函数已移除，避免重复注册http_request处理程序
// http_request.go文件已经注册了HTTPRequestHandler，它调用了httpRequestHandler
