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
	"vpr/pkg/utils"
)

// resourceHandler implements the ensure_resource_exists action
func resourceHandler(ctx *execContext.ExecutionContext, action *poc.Action) (interface{}, error) {
	if action.Type != "ensure_resource_exists" {
		return nil, fmt.Errorf("invalid action type for resourceHandler: %s", action.Type)
	}
	
	// Initialize parameters
	params := action.Parameters
	if params == nil {
		params = make(map[string]interface{})
	}
	
	// Get resource ID from action.Resource field (spec compliant) or from parameters
	resourceID := action.Resource
	if resourceID == "" {
		// Try to get from parameters as fallback
		if resourceParam, ok := params["resource"]; ok {
			if resourceStr, ok := resourceParam.(string); ok {
				var err error
				resourceID, err = ctx.Substitute(resourceStr)
				if err != nil {
					return nil, fmt.Errorf("failed to resolve resource id: %w", err)
				}
			}
		}
		
		if resourceID == "" {
			// Also try resource_id parameter for backward compatibility
			if resourceParam, ok := params["resource_id"]; ok {
				if resourceStr, ok := resourceParam.(string); ok {
					var err error
					resourceID, err = ctx.Substitute(resourceStr)
					if err != nil {
						return nil, fmt.Errorf("failed to resolve resource_id: %w", err)
					}
				}
			}
		}
	}
	
	// Get user context from action.UserContext field (spec compliant) or from parameters
	userContext := action.UserContext
	if userContext == "" {
		// Try to get from parameters as fallback
		if userParam, ok := params["user_context"]; ok {
			if userStr, ok := userParam.(string); ok {
				var err error
				userContext, err = ctx.Substitute(userStr)
				if err != nil {
					return nil, fmt.Errorf("failed to resolve user_context: %w", err)
				}
			}
		}
	}
	
	// Get resource type - either from parameters or default to resourceID if not specified
	resourceType, err := getParamStrWithDefault(ctx, params, "resource_type", resourceID)
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
			// Substitute variables in the body string, including {{resource}} and {{user_context}}
			bodyWithVars := bodyStr
			if resourceID != "" {
				bodyWithVars = strings.ReplaceAll(bodyWithVars, "{{resource}}", resourceID)
			}
			if userContext != "" {
				bodyWithVars = strings.ReplaceAll(bodyWithVars, "{{user_context}}", userContext)
			}
			
			resolvedBody, err := ctx.Substitute(bodyWithVars)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve create_body: %w", err)
			}
			createBodyStr = resolvedBody
			createBody = strings.NewReader(resolvedBody)
		} else if bodyMap, ok := bodyParam.(map[string]interface{}); ok {
			// If body is a map, convert to JSON
			// First, process any string values that might contain variables
			processedMap := make(map[string]interface{})
			for k, v := range bodyMap {
				if vStr, ok := v.(string); ok {
					// Handle special variable substitutions
					if resourceID != "" {
						vStr = strings.ReplaceAll(vStr, "{{resource}}", resourceID)
					}
					if userContext != "" {
						vStr = strings.ReplaceAll(vStr, "{{user_context}}", userContext)
					}
					
					// Regular variable substitution
					resolvedValue, err := ctx.Substitute(vStr)
					if err != nil {
						return nil, fmt.Errorf("failed to resolve body value: %w", err)
					}
					processedMap[k] = resolvedValue
				} else {
					processedMap[k] = v
				}
			}
			
			jsonBody, err := json.Marshal(processedMap)
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
					// Handle special variable substitutions
					if resourceID != "" {
						vStr = strings.ReplaceAll(vStr, "{{resource}}", resourceID)
					}
					if userContext != "" {
						vStr = strings.ReplaceAll(vStr, "{{user_context}}", userContext)
					}
					
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
	client, err := utils.GetHTTPClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get HTTP client: %w", err)
	}
	
	// Result object to return
	result := map[string]interface{}{
		"resource_type": resourceType,
		"resource_id":   resourceID,
		"user_context":  userContext,
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
		
		// Try to extract resource ID from response if not provided
		if resourceID == "" && len(checkRespBody) > 0 {
			var respObj map[string]interface{}
			if err := json.Unmarshal(checkRespBody, &respObj); err == nil {
				// Look for common ID fields
				for _, idField := range []string{"id", "resource_id", "resourceId", "ID"} {
					if id, ok := respObj[idField]; ok {
						if idStr, ok := id.(string); ok && idStr != "" {
							result["resource_id"] = idStr
							break
						}
					}
				}
			}
		}
	}
	
	// If resource exists, we're done
	if exists {
		result["exists"] = true
		
		// Store result in target variable if specified
		targetVar := action.TargetVariable
		// If output_variable is specified in parameters, use that instead (spec compliant)
		if outputVar, ok := params["output_variable"]; ok {
			if outputVarStr, ok := outputVar.(string); ok && outputVarStr != "" {
				targetVar = outputVarStr
			}
		}
		
		if targetVar != "" {
			err = ctx.SetVariable(targetVar, result)
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
			
			// Try to extract resource ID from response if successful
			if err == nil && len(createRespBody) > 0 {
				var respObj map[string]interface{}
				if json.Unmarshal(createRespBody, &respObj) == nil {
					// Look for common ID fields
					for _, idField := range []string{"id", "resource_id", "resourceId", "ID"} {
						if id, ok := respObj[idField]; ok {
							if idStr, ok := id.(string); ok && idStr != "" {
								result["resource_id"] = idStr
								break
							}
						}
					}
				}
			}
		}
		
		if err != nil {
			result["create_error"] = err.Error()
			
			// Store result in target variable if specified
			targetVar := action.TargetVariable
			// If output_variable is specified in parameters, use that instead (spec compliant)
			if outputVar, ok := params["output_variable"]; ok {
				if outputVarStr, ok := outputVar.(string); ok && outputVarStr != "" {
					targetVar = outputVarStr
				}
			}
			
			if targetVar != "" {
				_ = ctx.SetVariable(targetVar, result)
			}
			
			return result, fmt.Errorf("resource creation failed: %w", err)
		}
		
		result["created"] = created
		
		// Store result in target variable if specified
		targetVar := action.TargetVariable
		// If output_variable is specified in parameters, use that instead (spec compliant)
		if outputVar, ok := params["output_variable"]; ok {
			if outputVarStr, ok := outputVar.(string); ok && outputVarStr != "" {
				targetVar = outputVarStr
			}
		}
		
		if targetVar != "" {
			err = ctx.SetVariable(targetVar, result)
			if err != nil {
				return result, fmt.Errorf("failed to set target variable: %w", err)
			}
		}
		
		return result, nil
	}
	
	// If we get here, the resource doesn't exist and we couldn't create it
	return result, fmt.Errorf("resource doesn't exist and no create endpoint provided")
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

// getParamStrWithDefault gets a string parameter from action parameters with a default value
func getParamStrWithDefault(ctx *execContext.ExecutionContext, params map[string]interface{}, paramName, defaultValue string) (string, error) {
	if params == nil {
		return defaultValue, nil
	}
	
	paramValue, ok := params[paramName]
	if !ok {
		return defaultValue, nil
	}
	
	paramStr, ok := paramValue.(string)
	if !ok {
		return defaultValue, nil
	}
	
	// Resolve any variables
	resolvedValue, err := ctx.Substitute(paramStr)
	if err != nil {
		return defaultValue, fmt.Errorf("failed to resolve parameter %s: %w", paramName, err)
	}
	
	return resolvedValue, nil
}

// init registers the action handler
func init() {
	MustRegisterAction("ensure_resource_exists", resourceHandler)
}
