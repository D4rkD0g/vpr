// Package actions implements action handlers for the VPR engine.
// This file implements the authenticate action to support various authentication flows
// such as form login, OAuth2, and other authentication mechanisms.
package actions

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
	
	"vpr/pkg/context"
	"vpr/pkg/credentials"
	"vpr/pkg/poc"
	"vpr/pkg/utils"
)

// authenticationHandler implements the authenticate action
func authenticationHandler(ctx *context.ExecutionContext, action *poc.Action) (interface{}, error) {
	if action.Type != "authenticate" {
		return nil, fmt.Errorf("invalid action type for authenticationHandler: %s", action.Type)
	}
	
	// Validate required fields
	if action.AuthType == "" {
		return nil, fmt.Errorf("authenticate action requires 'auth_type' field")
	}
	
	// Determine authentication method
	switch strings.ToLower(action.AuthType) {
	case "form":
		return formAuthentication(ctx, action)
	case "oauth2_client_credentials":
		return oauth2ClientCredentialsAuthentication(ctx, action)
	case "oauth2_password":
		return oauth2PasswordAuthentication(ctx, action)
	case "basic":
		return basicAuthentication(ctx, action)
	case "api_key":
		return apiKeyAuthentication(ctx, action)
	default:
		return nil, fmt.Errorf("unsupported authentication type: %s", action.AuthType)
	}
}

// formAuthentication handles form-based authentication
func formAuthentication(ctx *context.ExecutionContext, action *poc.Action) (interface{}, error) {
	// Get required parameters
	params := action.Parameters
	if params == nil {
		return nil, fmt.Errorf("form authentication requires parameters")
	}
	
	// Get login URL
	loginURL, err := getRequiredParam(ctx, params, "login_url")
	if err != nil {
		return nil, err
	}
	
	// Get username and password fields
	usernameField, err := getRequiredParam(ctx, params, "username_field")
	if err != nil {
		return nil, err
	}
	
	passwordField, err := getRequiredParam(ctx, params, "password_field")
	if err != nil {
		return nil, err
	}
	
	// Get username and password values
	username, err := getRequiredParam(ctx, params, "username")
	if err != nil {
		return nil, err
	}
	
	password, err := getRequiredParam(ctx, params, "password")
	if err != nil {
		return nil, err
	}
	
	// Create form data
	formData := url.Values{}
	formData.Set(usernameField, username)
	formData.Set(passwordField, password)
	
	// Add any additional form fields
	if additionalFields, ok := params["additional_fields"].(map[string]interface{}); ok {
		for key, value := range additionalFields {
			if strValue, ok := value.(string); ok {
				// Resolve any variables
				resolvedValue, err := ctx.Substitute(strValue)
				if err != nil {
					return nil, fmt.Errorf("failed to resolve field value: %w", err)
				}
				formData.Set(key, resolvedValue)
			}
		}
	}
	
	// Get an HTTP client
	client, err := utils.GetHTTPClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get HTTP client: %w", err)
	}
	
	// Create request
	req, err := http.NewRequest("POST", loginURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "VPR-PoCRunner/1.0")
	
	// Add any custom headers
	if customHeaders, ok := params["headers"].(map[string]interface{}); ok {
		for key, value := range customHeaders {
			if strValue, ok := value.(string); ok {
				// Resolve any variables
				resolvedValue, err := ctx.Substitute(strValue)
				if err != nil {
					return nil, fmt.Errorf("failed to resolve header value: %w", err)
				}
				req.Header.Set(key, resolvedValue)
			}
		}
	}
	
	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("authentication request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Process response
	return processAuthResponse(ctx, action, resp)
}

// oauth2ClientCredentialsAuthentication handles OAuth2 client credentials flow
func oauth2ClientCredentialsAuthentication(ctx *context.ExecutionContext, action *poc.Action) (interface{}, error) {
	// Get required parameters
	params := action.Parameters
	if params == nil {
		return nil, fmt.Errorf("OAuth2 client credentials authentication requires parameters")
	}
	
	// Get token URL
	tokenURL, err := getRequiredParam(ctx, params, "token_url")
	if err != nil {
		return nil, err
	}
	
	// Get client credentials
	clientID, err := getRequiredParam(ctx, params, "client_id")
	if err != nil {
		return nil, err
	}
	
	clientSecret, err := getRequiredParam(ctx, params, "client_secret")
	if err != nil {
		return nil, err
	}
	
	// Get scope (optional)
	scope := ""
	if scopeParam, ok := params["scope"]; ok {
		if scopeStr, ok := scopeParam.(string); ok {
			scope, err = ctx.Substitute(scopeStr)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve scope: %w", err)
			}
		}
	}
	
	// Create form data
	formData := url.Values{}
	formData.Set("grant_type", "client_credentials")
	formData.Set("client_id", clientID)
	formData.Set("client_secret", clientSecret)
	if scope != "" {
		formData.Set("scope", scope)
	}
	
	// Create HTTP client
	client, err := getHTTPClient(ctx)
	if err != nil {
		return nil, err
	}
	
	// Create request
	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "VPR-PoCRunner/1.0")
	
	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("authentication request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Process response
	return processAuthResponse(ctx, action, resp)
}

// oauth2PasswordAuthentication handles OAuth2 password flow
func oauth2PasswordAuthentication(ctx *context.ExecutionContext, action *poc.Action) (interface{}, error) {
	// Get required parameters
	params := action.Parameters
	if params == nil {
		return nil, fmt.Errorf("OAuth2 password authentication requires parameters")
	}
	
	// Get token URL
	tokenURL, err := getRequiredParam(ctx, params, "token_url")
	if err != nil {
		return nil, err
	}
	
	// Get client credentials
	clientID, err := getRequiredParam(ctx, params, "client_id")
	if err != nil {
		return nil, err
	}
	
	// Client secret is optional for public clients
	clientSecret := ""
	if secretParam, ok := params["client_secret"]; ok {
		if secretStr, ok := secretParam.(string); ok {
			clientSecret, err = ctx.Substitute(secretStr)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve client_secret: %w", err)
			}
		}
	}
	
	// Get username and password
	username, err := getRequiredParam(ctx, params, "username")
	if err != nil {
		return nil, err
	}
	
	password, err := getRequiredParam(ctx, params, "password")
	if err != nil {
		return nil, err
	}
	
	// Get scope (optional)
	scope := ""
	if scopeParam, ok := params["scope"]; ok {
		if scopeStr, ok := scopeParam.(string); ok {
			scope, err = ctx.Substitute(scopeStr)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve scope: %w", err)
			}
		}
	}
	
	// Create form data
	formData := url.Values{}
	formData.Set("grant_type", "password")
	formData.Set("client_id", clientID)
	if clientSecret != "" {
		formData.Set("client_secret", clientSecret)
	}
	formData.Set("username", username)
	formData.Set("password", password)
	if scope != "" {
		formData.Set("scope", scope)
	}
	
	// Create HTTP client
	client, err := getHTTPClient(ctx)
	if err != nil {
		return nil, err
	}
	
	// Create request
	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "VPR-PoCRunner/1.0")
	
	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("authentication request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Process response
	return processAuthResponse(ctx, action, resp)
}

// basicAuthentication handles HTTP Basic authentication
func basicAuthentication(ctx *context.ExecutionContext, action *poc.Action) (interface{}, error) {
	// Get required parameters
	params := action.Parameters
	if params == nil {
		return nil, fmt.Errorf("basic authentication requires parameters")
	}
	
	// Get username and password
	username, err := getRequiredParam(ctx, params, "username")
	if err != nil {
		return nil, err
	}
	
	password, err := getRequiredParam(ctx, params, "password")
	if err != nil {
		return nil, err
	}
	
	// For Basic auth, we don't actually make a request
	// We just store the credentials for future use
	
	// Create credentials map
	credentials := map[string]string{
		"username": username,
		"password": password,
		"type":     "basic",
	}
	
	// If user to associate with is specified
	userContext := action.UserContext
	if userContext == "" {
		// If no user context specified, use active authentication context
		userContext = ctx.GetAuthenticationContext()
		// If still empty, error
		if userContext == "" {
			return nil, fmt.Errorf("no user context specified for authentication")
		}
	}
	
	// Store the credentials in the context
	credentialsPath := fmt.Sprintf("users.%s.credentials", userContext)
	err = ctx.SetVariable(credentialsPath, credentials)
	if err != nil {
		return nil, fmt.Errorf("failed to store credentials: %w", err)
	}
	
	// Set authentication context
	err = ctx.SetAuthenticationContext(userContext)
	if err != nil {
		return nil, fmt.Errorf("failed to set authentication context: %w", err)
	}
	
	return map[string]interface{}{
		"authenticated": true,
		"type":          "basic",
		"username":      username,
		"user_context":  userContext,
	}, nil
}

// apiKeyAuthentication handles API key authentication
func apiKeyAuthentication(ctx *context.ExecutionContext, action *poc.Action) (interface{}, error) {
	// Get required parameters
	params := action.Parameters
	if params == nil {
		return nil, fmt.Errorf("API key authentication requires parameters")
	}
	
	// Get API key
	apiKey, err := getRequiredParam(ctx, params, "api_key")
	if err != nil {
		return nil, err
	}
	
	// Get API key header (optional, default to X-API-Key)
	apiKeyHeader := "X-API-Key"
	if headerParam, ok := params["api_key_header"]; ok {
		if headerStr, ok := headerParam.(string); ok {
			apiKeyHeader, err = ctx.Substitute(headerStr)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve api_key_header: %w", err)
			}
		}
	}
	
	// For API key auth, we don't actually make a request
	// We just store the credentials for future use
	
	// Create credentials map
	credentials := map[string]string{
		"api_key":       apiKey,
		"api_key_header": apiKeyHeader,
		"type":          "api_key",
	}
	
	// If user to associate with is specified
	userContext := action.UserContext
	if userContext == "" {
		// If no user context specified, use active authentication context
		userContext = ctx.GetAuthenticationContext()
		// If still empty, error
		if userContext == "" {
			return nil, fmt.Errorf("no user context specified for authentication")
		}
	}
	
	// Store the credentials in the context
	credentialsPath := fmt.Sprintf("users.%s.credentials", userContext)
	err = ctx.SetVariable(credentialsPath, credentials)
	if err != nil {
		return nil, fmt.Errorf("failed to store credentials: %w", err)
	}
	
	// Set authentication context
	err = ctx.SetAuthenticationContext(userContext)
	if err != nil {
		return nil, fmt.Errorf("failed to set authentication context: %w", err)
	}
	
	return map[string]interface{}{
		"authenticated": true,
		"type":          "api_key",
		"api_key_header": apiKeyHeader,
		"user_context":  userContext,
	}, nil
}

// processAuthResponse processes the authentication response
func processAuthResponse(ctx *context.ExecutionContext, action *poc.Action, resp *http.Response) (interface{}, error) {
	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	
	// Check if authentication was successful (2xx status code)
	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	
	// Create result map
	result := map[string]interface{}{
		"status_code":   resp.StatusCode,
		"authenticated": success,
		"headers":       resp.Header,
		"body":          string(body),
	}
	
	// If not successful, return error
	if !success {
		if action.TargetVariable != "" {
			// Still store the result if target variable is specified
			err = ctx.SetVariable(action.TargetVariable, result)
			if err != nil {
				return nil, fmt.Errorf("failed to set target variable: %w", err)
			}
		}
		
		return result, fmt.Errorf("authentication failed with status code %d", resp.StatusCode)
	}
	
	// Extract tokens if response is JSON
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		var tokenData map[string]interface{}
		err = json.Unmarshal(body, &tokenData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse JSON response: %w", err)
		}
		
		// Add token data to result
		for key, value := range tokenData {
			result[key] = value
		}
		
		// Extract tokens into credentials
		credentials := map[string]string{
			"type": strings.ToLower(action.AuthType),
		}
		
		// Common OAuth2 fields
		if accessToken, ok := tokenData["access_token"].(string); ok {
			credentials["bearer_token"] = accessToken
			result["token_type"] = "Bearer" // Default if not specified
			
			// Check for token type
			if tokenType, ok := tokenData["token_type"].(string); ok {
				result["token_type"] = tokenType
			}
		}
		
		// Refresh token if present
		if refreshToken, ok := tokenData["refresh_token"].(string); ok {
			credentials["refresh_token"] = refreshToken
		}
		
		// Expiration if present
		if expiresIn, ok := tokenData["expires_in"].(float64); ok {
			credentials["expires_in"] = fmt.Sprintf("%.0f", expiresIn)
		}
		
		// If user to associate with is specified
		userContext := action.UserContext
		if userContext == "" {
			// If no user context specified, use active authentication context
			userContext = ctx.GetAuthenticationContext()
			// If still empty, use default
			if userContext == "" {
				// 尝试从action参数中获取默认用户
				if action.Parameters != nil {
					if defaultUser, ok := action.Parameters["default_user"]; ok {
						if defaultUserStr, ok := defaultUser.(string); ok {
							userContext = defaultUserStr
						}
					}
				}
				
				// If still empty, error
				if userContext == "" {
					return nil, fmt.Errorf("no user context specified for authentication")
				}
			}
		}
		
		// Store the credentials in the context
		credentialsPath := fmt.Sprintf("users.%s.credentials", userContext)
		err = ctx.SetVariable(credentialsPath, credentials)
		if err != nil {
			return nil, fmt.Errorf("failed to store credentials: %w", err)
		}
		
		// Set authentication context
		err = ctx.SetAuthenticationContext(userContext)
		if err != nil {
			return nil, fmt.Errorf("failed to set authentication context: %w", err)
		}
		
		// Add user context to result
		result["user_context"] = userContext
	}
	
	// Store result in target variable if specified
	if action.TargetVariable != "" {
		err = ctx.SetVariable(action.TargetVariable, result)
		if err != nil {
			return nil, fmt.Errorf("failed to set target variable: %w", err)
		}
	}
	
	return result, nil
}

// getRequiredParam gets a required parameter from action parameters
func getRequiredParam(ctx *context.ExecutionContext, params map[string]interface{}, paramName string) (string, error) {
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

// getHTTPClient gets an HTTP client from the context or creates a new one
func getHTTPClient(ctx *context.ExecutionContext) (*http.Client, error) {
	// Try to get existing client from context
	clientObj, err := ctx.ResolveVariable("http_client")
	if err == nil && clientObj != nil {
		if client, ok := clientObj.(*http.Client); ok {
			return client, nil
		}
	}
	
	// Create a new client
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Allow up to 10 redirects
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return nil
		},
	}
	
	// Store the client in the context for reuse
	err = ctx.SetVariable("http_client", client)
	if err != nil {
		return nil, fmt.Errorf("failed to store HTTP client in context: %w", err)
	}
	
	return client, nil
}

// init registers the action handler
func init() {
	MustRegisterAction("authenticate", authenticationHandler)
}
