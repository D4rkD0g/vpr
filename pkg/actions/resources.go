// Package actions implements action handlers for the VPR engine.
// This file implements the ensure_resource_exists action for verifying
// and creating required resources needed for PoC execution.
package actions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	
	execContext "vpr/pkg/context"
	"vpr/pkg/poc"
)

// resourceHandler implements the ensure_resource_exists action
func resourceHandler(ctx *execContext.ExecutionContext, action *poc.Action) (interface{}, error) {
	if action.Type != "ensure_resource_exists" {
		return nil, fmt.Errorf("invalid action type for resourceHandler: %s", action.Type)
	}
	
	// Get required parameters
	params := action.Parameters
	if params == nil {
		return nil, fmt.Errorf("ensure_resource_exists requires parameters")
	}
	
	// Get resource type
	resourceType, err := getRequiredParamStr(ctx, params, "resource_type")
	if err != nil {
		return nil, err
	}
	
	// Get check endpoint
	checkEndpoint, err := getRequiredParamStr(ctx, params, "check_endpoint")
	if err != nil {
		return nil, err
	}
	
	// Get check method (optional, default to GET)
	checkMethod := "GET"
	if methodParam, ok := params["check_method"]; ok {
		if methodStr, ok := methodParam.(string); ok {
			resolvedMethod, err := ctx.Substitute(methodStr)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve check_method: %w", err)
			}
			checkMethod = strings.ToUpper(resolvedMethod)
		}
	}
	
	// Get check body (optional)
	var checkBody io.Reader
	if bodyParam, ok := params["check_body"]; ok {
		if bodyStr, ok := bodyParam.(string); ok {
			resolvedBody, err := ctx.Substitute(bodyStr)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve check_body: %w", err)
			}
			checkBody = strings.NewReader(resolvedBody)
		} else if bodyMap, ok := bodyParam.(map[string]interface{}); ok {
			// If body is a map, convert to JSON
			jsonBody, err := json.Marshal(bodyMap)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal check_body: %w", err)
			}
			checkBody = bytes.NewReader(jsonBody)
		}
	}
	
	// Get check headers (optional)
	checkHeaders := make(map[string]string)
	if headersParam, ok := params["check_headers"]; ok {
		if headersMap, ok := headersParam.(map[string]interface{}); ok {
			for k, v := range headersMap {
				if vStr, ok := v.(string); ok {
					resolvedValue, err := ctx.Substitute(vStr)
					if err != nil {
						return nil, fmt.Errorf("failed to resolve header value: %w", err)
					}
					checkHeaders[k] = resolvedValue
				}
			}
		}
	}
	
	// Get create endpoint (optional)
	createEndpoint := ""
	if createParam, ok := params["create_endpoint"]; ok {
		if createStr, ok := createParam.(string); ok {
			resolvedCreate, err := ctx.Substitute(createStr)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve create_endpoint: %w", err)
			}
			createEndpoint = resolvedCreate
		}
	}
	
	// Get create method (optional, default to POST)
	createMethod := "POST"
	if methodParam, ok := params["create_method"]; ok {
		if methodStr, ok := methodParam.(string); ok {
			resolvedMethod, err := ctx.Substitute(methodStr)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve create_method: %w", err)
			}
			createMethod = strings.ToUpper(resolvedMethod)
		}
	}
	
	// Get create body (optional)
	var createBodyStr string
	var createBody io.Reader
	if bodyParam, ok := params["create_body"]; ok {
		if bodyStr, ok := bodyParam.(string); ok {
			resolvedBody, err := ctx.Substitute(bodyStr)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve create_body: %w", err)
			}
			createBodyStr = resolvedBody
			createBody = strings.NewReader(resolvedBody)
		} else if bodyMap, ok := bodyParam.(map[string]interface{}); ok {
			// If body is a map, convert to JSON
			jsonBody, err := json.Marshal(bodyMap)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal create_body: %w", err)
			}
			createBodyStr = string(jsonBody)
			createBody = bytes.NewReader(jsonBody)
		}
	}
	
	// Get create headers (optional)
	createHeaders := make(map[string]string)
	if headersParam, ok := params["create_headers"]; ok {
		if headersMap, ok := headersParam.(map[string]interface{}); ok {
			for k, v := range headersMap {
				if vStr, ok := v.(string); ok {
					resolvedValue, err := ctx.Substitute(vStr)
					if err != nil {
						return nil, fmt.Errorf("failed to resolve header value: %w", err)
					}
					createHeaders[k] = resolvedValue
				}
			}
		}
	}
	
	// Get content type (optional, default based on body)
	contentType := ""
	if ctParam, ok := params["content_type"]; ok {
		if ctStr, ok := ctParam.(string); ok {
			resolvedCT, err := ctx.Substitute(ctStr)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve content_type: %w", err)
			}
			contentType = resolvedCT
		}
	}
	
	// Get HTTP client
	client, err := getHTTPClient(ctx)
	if err != nil {
		return nil, err
	}
	
	// Result object to return
	result := map[string]interface{}{
		"resource_type": resourceType,
		"exists":        false,
		"created":       false,
	}
	
	// Check if resource exists
	exists, checkResp, err := checkResourceExists(ctx, client, checkMethod, checkEndpoint, checkBody, checkHeaders)
	if err != nil {
		result["check_error"] = err.Error()
		// If check fails and no create endpoint is provided, return error
		if createEndpoint == "" {
			if action.TargetVariable != "" {
				_ = ctx.SetVariable(action.TargetVariable, result)
			}
			return result, fmt.Errorf("resource check failed: %w", err)
		}
	}
	
	// Store check response in result
	if checkResp != nil {
		checkRespBody, _ := io.ReadAll(checkResp.Body)
		checkResp.Body.Close()
		
		result["check_status_code"] = checkResp.StatusCode
		result["check_headers"] = checkResp.Header
		result["check_body"] = string(checkRespBody)
	}
	
	// If resource exists, we're done
	if exists {
		result["exists"] = true
		
		// Store result in target variable if specified
		if action.TargetVariable != "" {
			err = ctx.SetVariable(action.TargetVariable, result)
			if err != nil {
				return result, fmt.Errorf("failed to set target variable: %w", err)
			}
		}
		
		return result, nil
	}
	
	// If resource doesn't exist but we have a create endpoint, create it
	if createEndpoint != "" {
		created, createResp, err := createResource(ctx, client, createMethod, createEndpoint, createBody, createBodyStr, createHeaders, contentType)
		
		// Store create response in result
		if createResp != nil {
			createRespBody, _ := io.ReadAll(createResp.Body)
			createResp.Body.Close()
			
			result["create_status_code"] = createResp.StatusCode
			result["create_headers"] = createResp.Header
			result["create_body"] = string(createRespBody)
		}
		
		if err != nil {
			result["create_error"] = err.Error()
			
			// Store result in target variable if specified
			if action.TargetVariable != "" {
				_ = ctx.SetVariable(action.TargetVariable, result)
			}
			
			return result, fmt.Errorf("resource creation failed: %w", err)
		}
		
		result["created"] = created
		
		// Store result in target variable if specified
		if action.TargetVariable != "" {
			err = ctx.SetVariable(action.TargetVariable, result)
			if err != nil {
				return result, fmt.Errorf("failed to set target variable: %w", err)
			}
		}
		
		return result, nil
	}
	
	// If we get here, resource doesn't exist and we can't create it
	if action.TargetVariable != "" {
		_ = ctx.SetVariable(action.TargetVariable, result)
	}
	
	return result, fmt.Errorf("resource '%s' does not exist and no create endpoint provided", resourceType)
}

// checkResourceExists checks if a resource exists
func checkResourceExists(ctx *execContext.ExecutionContext, client *http.Client, method, endpoint string, body io.Reader, headers map[string]string) (bool, *http.Response, error) {
	// Create request
	req, err := http.NewRequest(method, endpoint, body)
	if err != nil {
		return false, nil, fmt.Errorf("failed to create check request: %w", err)
	}
	
	// Set headers
	req.Header.Set("User-Agent", "VPR-PoCRunner/1.0")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	
	// Apply authentication if context is set
	authContext := ctx.GetAuthenticationContext()
	err = ctx.ApplyAuthentication(req, authContext)
	if err != nil {
		return false, nil, fmt.Errorf("failed to apply authentication: %w", err)
	}
	
	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return false, nil, fmt.Errorf("check request failed: %w", err)
	}
	
	// Check if resource exists (2xx status code)
	exists := resp.StatusCode >= 200 && resp.StatusCode < 300
	
	return exists, resp, nil
}

// createResource creates a resource
func createResource(ctx *execContext.ExecutionContext, client *http.Client, method, endpoint string, body io.Reader, bodyStr string, headers map[string]string, contentType string) (bool, *http.Response, error) {
	// If no content type specified, try to guess
	if contentType == "" {
		// If body starts with { or [, assume JSON
		if strings.HasPrefix(strings.TrimSpace(bodyStr), "{") || strings.HasPrefix(strings.TrimSpace(bodyStr), "[") {
			contentType = "application/json"
		} else if strings.Contains(bodyStr, "&") && strings.Contains(bodyStr, "=") {
			// If body contains & and =, assume form data
			contentType = "application/x-www-form-urlencoded"
		} else {
			// Default to text/plain
			contentType = "text/plain"
		}
	}
	
	// Handle multipart form data separately
	if strings.HasPrefix(contentType, "multipart/form-data") {
		formDataMap := make(map[string]string)
		// Parse the form data string into a map
		parts := strings.Split(bodyStr, "&")
		for _, part := range parts {
			kv := strings.SplitN(part, "=", 2)
			if len(kv) == 2 {
				formDataMap[kv[0]] = kv[1]
			}
		}
		return createMultipartResource(ctx, client, method, endpoint, formDataMap, headers)
	}
	
	// Create request
	req, err := http.NewRequest(method, endpoint, body)
	if err != nil {
		return false, nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("User-Agent", "VPR-PoCRunner/1.0")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	
	// Apply authentication if context is set
	authContext := ctx.GetAuthenticationContext()
	err = ctx.ApplyAuthentication(req, authContext)
	if err != nil {
		return false, nil, fmt.Errorf("failed to apply authentication: %w", err)
	}
	
	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return false, nil, fmt.Errorf("create request failed: %w", err)
	}
	
	// Check if creation was successful (2xx status code)
	created := resp.StatusCode >= 200 && resp.StatusCode < 300
	
	return created, resp, nil
}

// createMultipartResource creates a resource with multipart form data
func createMultipartResource(ctx *execContext.ExecutionContext, client *http.Client, method, endpoint string, formData map[string]string, headers map[string]string) (bool, *http.Response, error) {
	// Create multipart writer
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	
	// Add form fields
	for k, v := range formData {
		err := w.WriteField(k, v)
		if err != nil {
			return false, nil, fmt.Errorf("failed to write form field: %w", err)
		}
	}
	
	// Close writer
	err := w.Close()
	if err != nil {
		return false, nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}
	
	// Create request
	req, err := http.NewRequest(method, endpoint, &b)
	if err != nil {
		return false, nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("User-Agent", "VPR-PoCRunner/1.0")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	
	// Apply authentication if context is set
	authContext := ctx.GetAuthenticationContext()
	err = ctx.ApplyAuthentication(req, authContext)
	if err != nil {
		return false, nil, fmt.Errorf("failed to apply authentication: %w", err)
	}
	
	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return false, nil, fmt.Errorf("create request failed: %w", err)
	}
	
	// Check if creation was successful (2xx status code)
	created := resp.StatusCode >= 200 && resp.StatusCode < 300
	
	return created, resp, nil
}

// getRequiredParamStr gets a required string parameter from action parameters
func getRequiredParamStr(ctx *execContext.ExecutionContext, params map[string]interface{}, paramName string) (string, error) {
	if params == nil {
		return "", fmt.Errorf("missing required parameter: %s (parameters map is nil)", paramName)
	}
	
	paramValue, ok := params[paramName]
	if !ok {
		return "", fmt.Errorf("missing required parameter: %s", paramName)
	}
	
	paramStr, ok := paramValue.(string)
	if !ok {
		return "", fmt.Errorf("parameter %s must be a string", paramName)
	}
	
	// Resolve any variables
	resolvedValue, err := ctx.Substitute(paramStr)
	if err != nil {
		return "", fmt.Errorf("failed to resolve parameter %s: %w", paramName, err)
	}
	
	return resolvedValue, nil
}

// init registers the action handler
func init() {
	MustRegisterAction("ensure_resource_exists", resourceHandler)
}
