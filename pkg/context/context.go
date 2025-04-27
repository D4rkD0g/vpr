// Package context defines the ExecutionContext which holds the state during PoC execution.
// It manages resolved variables (from context definition and runtime extractions),
// handles variable substitution (`{{...}}`) and function calls within strings,
// evaluates conditional expressions, and potentially manages resources like HTTP clients
// and credential resolution.
package context

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"
	
	"vpr/pkg/credentials"
	"vpr/pkg/poc"
	// govaluate "github.com/Knetic/govaluate" // For expression evaluation
)

// ExecutionContext holds the state during PoC execution. NOT thread-safe by default.
type ExecutionContext struct {
	mu         sync.RWMutex           // For potential future concurrency
	resolved   map[string]interface{} // Stores all resolved context: users, resources, env, vars, loop vars
	pocContext *poc.Context           // Reference to the original PoC context definition
	
	// Authentication and credentials
	credProvider *credentials.Provider // Credential provider for resolving authentication
	activeAuth   string                // Currently active authentication context (user ID)
	
	// Other runtime state
	httpClient   *http.Client          // Reusable HTTP client
	
	// Registries
	extractorRegistry interface{} // Registry for extractors
	
	// Error handling
	lastError    error                 // Most recent error encountered
	lastStatusCode int                 // Most recent HTTP status code
	
	// Execution tracking
	lastStep    string                 // Most recent step ID or description
	lastResponse []byte                // Most recent response body
}

// NewExecutionContext creates a new execution context and initializes the resolved map
// with values defined in the PoC's context section.
func NewExecutionContext(pocCtx *poc.Context) (*ExecutionContext, error) {
	ctx := &ExecutionContext{
		resolved:     make(map[string]interface{}),
		pocContext:   pocCtx, // Keep reference if needed later
		credProvider: credentials.NewProvider(),
	}
	
	// Create default HTTP client with sensible defaults
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Allow up to 10 redirects
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return nil
		},
	}
	ctx.httpClient = client

	// --- Initialize resolved map from pocContext --- 
	// Store structured data for easier access later if needed

	// Variables
	varsMap := make(map[string]interface{})
	for _, v := range pocCtx.Variables {
		varsMap[v.ID] = v // Store the whole ContextVariable struct
	}
	if len(varsMap) > 0 {
		ctx.resolved["variables"] = varsMap
	}

	// Environment
	envMap := make(map[string]interface{})
	for _, e := range pocCtx.Environment {
		envMap[e.ID] = e // Store the whole ContextEnvironment struct
	}
	if len(envMap) > 0 {
		ctx.resolved["environment"] = envMap
	}

	// Users
	usersMap := make(map[string]interface{})
	for _, u := range pocCtx.Users {
		usersMap[u.ID] = u // Store the whole ContextUser struct
	}
	if len(usersMap) > 0 {
		ctx.resolved["users"] = usersMap
	}

	// Resources
	resourcesMap := make(map[string]interface{})
	for _, r := range pocCtx.Resources {
		resourcesMap[r.ID] = r // Store the whole ContextResource struct
	}
	if len(resourcesMap) > 0 {
		ctx.resolved["resources"] = resourcesMap
	}

	// Files
	filesMap := make(map[string]interface{})
	for _, f := range pocCtx.Files {
		filesMap[f.ID] = f // Store the whole ContextFile struct
	}
	if len(filesMap) > 0 {
		ctx.resolved["files"] = filesMap
	}

	// TODO: Pre-resolve credentials specified directly in context?
	// TODO: Initialize runtime-specific context like http clients etc.

	return ctx, nil
}

// ResolveVariable resolves path like "variables.myVar.Value" or "environment.target_host.Value"
// It navigates maps and structs within the resolved context data.
func (c *ExecutionContext) ResolveVariable(path string) (interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid variable path: empty path")
	}

	var current interface{} = c.resolved // Start from the root resolved map

	for i, part := range parts {
		if current == nil {
			return nil, fmt.Errorf("cannot resolve path '%s': encountered nil value at '%s'", path, strings.Join(parts[:i], "."))
		}

		currentValue := reflect.ValueOf(current)
		// Dereference pointers if needed
		if currentValue.Kind() == reflect.Ptr {
			currentValue = currentValue.Elem()
		}

		switch currentValue.Kind() {
		case reflect.Map:
			// Check if map key type is string
			if currentValue.Type().Key().Kind() != reflect.String {
				return nil, fmt.Errorf("cannot resolve path '%s': map key is not string at '%s'", path, strings.Join(parts[:i], "."))
			}
			// Look up the part in the map
			keyValue := reflect.ValueOf(part)
			mapValue := currentValue.MapIndex(keyValue)
			if !mapValue.IsValid() {
				// Key not found
				return nil, fmt.Errorf("path part '%s' not found in map context at '%s'", part, strings.Join(parts[:i], "."))
			}
			current = mapValue.Interface() // Update current to the value found in the map

		case reflect.Struct:
			// Look up the part as a struct field name (case-sensitive)
			fieldValue := currentValue.FieldByName(part)
			if !fieldValue.IsValid() {
				// Field not found
				return nil, fmt.Errorf("field '%s' not found in struct context at '%s' (type: %s)", part, strings.Join(parts[:i], "."), currentValue.Type().Name())
			}
			if !fieldValue.CanInterface() {
				return nil, fmt.Errorf("cannot access unexported field '%s' in struct context at '%s'", part, strings.Join(parts[:i], "."))
			}
			current = fieldValue.Interface() // Update current to the field value

		default:
			// Encountered a non-map, non-struct type before reaching the end of the path
			if i < len(parts)-1 { // More parts remaining means error
				return nil, fmt.Errorf("cannot resolve path '%s': encountered non-navigable type '%s' at '%s'", path, currentValue.Kind(), strings.Join(parts[:i], "."))
			}
			// If it's the last part, this is the final value, handled below.
		}
	}

	// After iterating through all parts, 'current' holds the final value
	return current, nil
}

// SetVariable sets a value at a given path within the resolved context map.
// It creates nested maps (map[string]interface{}) as needed along the path.
// Note: This primarily targets the map structure and avoids modifying underlying structs directly.
func (c *ExecutionContext) SetVariable(path string, value interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return fmt.Errorf("invalid variable path: empty path")
	}

	currentMap := c.resolved
	// Iterate through path parts, creating nested maps until the last part
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		next, exists := currentMap[part]

		if !exists {
			// Create a new map if the key doesn't exist
			newMap := make(map[string]interface{})
			currentMap[part] = newMap
			currentMap = newMap
		} else {
			// Check if the existing value is a map
			if nextMap, ok := next.(map[string]interface{}); ok {
				currentMap = nextMap
			} else {
				// Path conflicts with an existing non-map value
				return fmt.Errorf("cannot set variable: path part '%s' conflicts with existing non-map value at '%s'", part, strings.Join(parts[:i+1], "."))
			}
		}
	}

	// Set the final value at the last part of the path
	lastPart := parts[len(parts)-1]
	currentMap[lastPart] = value

	return nil
}

// Substitute performs variable substitution in a string, including function calls.
// It replaces all occurrences of {{ ... }} with their resolved values.
func (c *ExecutionContext) Substitute(input string) (string, error) {
	if input == "" {
		return input, nil
	}

	// Quick check if there's any substitution needed
	if !strings.Contains(input, "{{") {
		return input, nil
	}

	result := input
	// Regular expression to find {{ ... }} patterns
	// This simple implementation uses string functions for now
	// TODO: Consider using regexp for more robust parsing
	
	startIdx := 0
	for {
		// Find the next substitution pattern
		openBrace := strings.Index(result[startIdx:], "{{")
		if openBrace == -1 {
			break // No more substitution patterns
		}
		openBrace += startIdx
		
		// Find the closing braces
		closeBrace := strings.Index(result[openBrace:], "}}")
		if closeBrace == -1 {
			return result, fmt.Errorf("unclosed substitution pattern in '%s'", result[openBrace:])
		}
		closeBrace += openBrace
		
		// Extract the pattern content (trimming spaces)
		pattern := strings.TrimSpace(result[openBrace+2:closeBrace])
		
		// Resolve the pattern
		var replacement interface{}
		var err error
		
		// Check if it's a function call
		funcNameEnd := strings.Index(pattern, "(")
		if funcNameEnd != -1 {
			// It's a function call pattern like func_name(arg1, arg2, ...)
			funcName := strings.TrimSpace(pattern[:funcNameEnd])
			
			// Get the raw arguments string and trim spaces
			if !strings.HasSuffix(pattern, ")") {
				return result, fmt.Errorf("invalid function call syntax: missing closing parenthesis in '%s'", pattern)
			}
			argsStr := strings.TrimSpace(pattern[funcNameEnd+1 : len(pattern)-1])
			
			// Parse arguments
			var args []interface{}
			if argsStr != "" {
				// Split by commas, but respect nested parentheses
				// This is a simplified implementation and may not handle all edge cases
				rawArgs := splitArgs(argsStr)
				for _, rawArg := range rawArgs {
					rawArg = strings.TrimSpace(rawArg)
					
					// Check if argument is a nested variable reference
					if strings.HasPrefix(rawArg, "{{") && strings.HasSuffix(rawArg, "}}") {
						// Recursive substitution for nested variables
						resolvedArg, err := c.Substitute(rawArg)
						if err != nil {
							return result, fmt.Errorf("error resolving nested variable in function argument: %w", err)
						}
						args = append(args, resolvedArg)
					} else if strings.HasPrefix(rawArg, "context.") {
						// Directly resolve variable reference
						resolvedArg, err := c.ResolveVariable(rawArg)
						if err != nil {
							return result, fmt.Errorf("error resolving variable in function argument: %w", err)
						}
						args = append(args, resolvedArg)
					} else {
						// It's a literal value
						args = append(args, rawArg)
					}
				}
			}
			
			// Execute the function
			funcImpl, exists := GetFunction(funcName)
			if !exists {
				return result, fmt.Errorf("undefined function: %s", funcName)
			}
			
			replacement, err = funcImpl(args...)
			if err != nil {
				return result, fmt.Errorf("error executing function %s: %w", funcName, err)
			}
		} else {
			// It's a variable reference like context.variables.myVar.value
			replacement, err = c.ResolveVariable(pattern)
			if err != nil {
				return result, fmt.Errorf("error resolving variable %s: %w", pattern, err)
			}
		}
		
		// Convert replacement to string
		var replacementStr string
		switch v := replacement.(type) {
		case string:
			replacementStr = v
		case fmt.Stringer:
			replacementStr = v.String()
		default:
			replacementStr = fmt.Sprintf("%v", v)
		}
		
		// Replace the pattern in the original string
		result = result[:openBrace] + replacementStr + result[closeBrace+2:]
		
		// Update start index for next search
		startIdx = openBrace + len(replacementStr)
		if startIdx >= len(result) {
			break
		}
	}
	
	return result, nil
}

// splitArgs splits a comma-separated argument string, respecting nested parentheses.
// For example: "arg1, func(a, b), arg3" -> ["arg1", "func(a, b)", "arg3"]
func splitArgs(argsStr string) []string {
	var args []string
	var currentArg strings.Builder
	parenLevel := 0
	
	for _, char := range argsStr {
		switch char {
		case '(':
			parenLevel++
			currentArg.WriteRune(char)
		case ')':
			parenLevel--
			currentArg.WriteRune(char)
		case ',':
			if parenLevel == 0 {
				// Top-level comma, split here
				args = append(args, currentArg.String())
				currentArg.Reset()
			} else {
				// Comma inside parentheses, keep as part of the argument
				currentArg.WriteRune(char)
			}
		default:
			currentArg.WriteRune(char)
		}
	}
	
	// Add the last argument if non-empty
	lastArg := currentArg.String()
	if len(lastArg) > 0 {
		args = append(args, lastArg)
	}
	
	// Trim spaces from all arguments
	for i, arg := range args {
		args[i] = strings.TrimSpace(arg)
	}
	
	return args
}

// EvaluateCondition evaluates an 'if' condition string.
// Conditions in PoC DSL can use variable references and basic operators.
func (c *ExecutionContext) EvaluateCondition(condition string) (bool, error) {
	if condition == "" {
		return true, nil // Empty condition evaluates to true
	}
	
	// Perform variable substitution first
	resolvedCondition, err := c.Substitute(condition)
	if err != nil {
		return false, fmt.Errorf("error substituting variables in condition: %w", err)
	}
	
	// Simple condition evaluation for now
	// TODO: Implement a proper expression evaluator or use a library
	
	// Handle simple equality checks (a == b)
	if strings.Contains(resolvedCondition, "==") {
		parts := strings.Split(resolvedCondition, "==")
		if len(parts) != 2 {
			return false, fmt.Errorf("invalid equality expression: %s", resolvedCondition)
		}
		left := strings.TrimSpace(parts[0])
		right := strings.TrimSpace(parts[1])
		return left == right, nil
	}
	
	// Handle simple inequality checks (a != b)
	if strings.Contains(resolvedCondition, "!=") {
		parts := strings.Split(resolvedCondition, "!=")
		if len(parts) != 2 {
			return false, fmt.Errorf("invalid inequality expression: %s", resolvedCondition)
		}
		left := strings.TrimSpace(parts[0])
		right := strings.TrimSpace(parts[1])
		return left != right, nil
	}
	
	// Handle true/false literals
	resolvedCondition = strings.TrimSpace(resolvedCondition)
	if resolvedCondition == "true" {
		return true, nil
	}
	if resolvedCondition == "false" {
		return false, nil
	}
	
	// If the condition is not empty and not a recognized format, 
	// assume it's a boolean variable or expression that should be truthy
	return resolvedCondition != "" && resolvedCondition != "0" && resolvedCondition != "false", nil
}

// RegisterCredentialResolver registers a credential resolver with this context
func (ctx *ExecutionContext) RegisterCredentialResolver(resolver credentials.CredentialResolver) error {
	if resolver == nil {
		return fmt.Errorf("cannot register nil resolver")
	}
	
	return ctx.credProvider.RegisterResolver(resolver)
}

// ResolveUserCredentials retrieves credentials for a user reference
// This extends the existing GetCredentials method with credential resolver support
func (ctx *ExecutionContext) ResolveUserCredentials(userRef string) (map[string]string, error) {
	// If no user reference provided, return an empty credentials map
	if userRef == "" {
		return map[string]string{}, nil
	}
	
	// Get user from context
	usersMap, ok := ctx.resolved["users"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("users context not properly initialized")
	}
	
	user, ok := usersMap[userRef].(poc.ContextUser)
	if !ok {
		return nil, fmt.Errorf("user '%s' not found in context", userRef)
	}
	
	// If user has no credentials_ref, return an empty map
	if user.CredentialsRef == "" {
		return map[string]string{}, nil
	}
	
	// Resolve credentials using provider
	if !ctx.credProvider.HasResolvers() {
		return nil, fmt.Errorf("no credential resolvers registered")
	}
	
	return ctx.credProvider.ResolveCredentials(user.CredentialsRef)
}

// SetAuthenticationContext sets the active authentication context to the specified user
func (ctx *ExecutionContext) SetAuthenticationContext(userRef string) error {
	// Empty userRef clears authentication context
	if userRef == "" {
		ctx.mu.Lock()
		ctx.activeAuth = ""
		ctx.mu.Unlock()
		return nil
	}
	
	// Verify user exists
	usersMap, ok := ctx.resolved["users"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("users context not properly initialized")
	}
	
	_, ok = usersMap[userRef].(poc.ContextUser)
	if !ok {
		return fmt.Errorf("user '%s' not found in context", userRef)
	}
	
	// Set active authentication context
	ctx.mu.Lock()
	ctx.activeAuth = userRef
	ctx.mu.Unlock()
	
	return nil
}

// GetAuthenticationContext returns the current active authentication context
func (ctx *ExecutionContext) GetAuthenticationContext() string {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	return ctx.activeAuth
}

// ApplyAuthentication applies the credentials of the currently active authentication
// context or the specified user to the provided HTTP request
func (ctx *ExecutionContext) ApplyAuthentication(req *http.Request, userRef string) error {
	// If specific user not provided, use active authentication context
	if userRef == "" {
		ctx.mu.RLock()
		userRef = ctx.activeAuth
		ctx.mu.RUnlock()
		
		// If still empty, no authentication to apply
		if userRef == "" {
			return nil
		}
	}
	
	// Get credentials
	creds, err := ctx.ResolveUserCredentials(userRef)
	if err != nil {
		return fmt.Errorf("failed to get credentials for user '%s': %w", userRef, err)
	}
	
	// If no credentials, nothing to apply
	if len(creds) == 0 {
		return nil
	}
	
	// Apply common credential types
	
	// Bearer token in Authorization header
	if token, ok := creds["bearer_token"]; ok && token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	
	// Basic auth
	if username, hasUser := creds["username"]; hasUser {
		if password, hasPass := creds["password"]; hasPass {
			req.SetBasicAuth(username, password)
		}
	}
	
	// API key
	if apiKey, ok := creds["api_key"]; ok && apiKey != "" {
		// Check if there's a header specified for the API key
		if header, hasHeader := creds["api_key_header"]; hasHeader && header != "" {
			req.Header.Set(header, apiKey)
		} else {
			// Default to X-API-Key if not specified
			req.Header.Set("X-API-Key", apiKey)
		}
	}
	
	// Cookie
	if cookieValue, ok := creds["cookie"]; ok && cookieValue != "" {
		// Check if there's a specific cookie name
		cookieName := "session"
		if name, hasName := creds["cookie_name"]; hasName && name != "" {
			cookieName = name
		}
		
		cookie := &http.Cookie{
			Name:  cookieName,
			Value: cookieValue,
			Path:  "/",
		}
		req.AddCookie(cookie)
	}
	
	// Custom headers - headers must be a JSON object stored as a string
	if headersJson, ok := creds["headers"]; ok && headersJson != "" {
		var headers map[string]string
		// Try to parse the headers JSON
		if err := json.Unmarshal([]byte(headersJson), &headers); err == nil {
			for name, value := range headers {
				req.Header.Set(name, value)
			}
		}
	}
	
	return nil
}

// CreateLoopContext creates a new context for loop execution with the current item value.
func (c *ExecutionContext) CreateLoopContext(loopVar string, itemValue interface{}) (*ExecutionContext, error) {
	// Create a child context
	childCtx := &ExecutionContext{
		resolved:   make(map[string]interface{}),
		pocContext: c.pocContext,
	}
	
	// Copy all existing resolved values
	c.mu.RLock()
	for k, v := range c.resolved {
		childCtx.resolved[k] = v
	}
	c.mu.RUnlock()
	
	// Add loop variable
	loopPath := fmt.Sprintf("loop.%s", loopVar)
	err := childCtx.SetVariable(loopPath, itemValue)
	if err != nil {
		return nil, fmt.Errorf("failed to set loop variable: %w", err)
	}
	
	// Set the special current_id variable referenced in the spec
	err = childCtx.SetVariable("loop.current_id", itemValue)
	if err != nil {
		return nil, fmt.Errorf("failed to set loop.current_id: %w", err)
	}
	
	return childCtx, nil
}

// SetLastError stores the most recent error encountered during execution
func (ctx *ExecutionContext) SetLastError(err error) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.lastError = err
}

// GetLastError retrieves the most recent error encountered during execution
func (ctx *ExecutionContext) GetLastError() error {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	return ctx.lastError
}

// SetLastStatusCode stores the most recent HTTP status code
func (ctx *ExecutionContext) SetLastStatusCode(code int) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.lastStatusCode = code
}

// GetLastStatusCode retrieves the most recent HTTP status code
func (ctx *ExecutionContext) GetLastStatusCode() int {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	return ctx.lastStatusCode
}

// GetErrorCode attempts to extract an error code from the last error
// This is a utility function for checks that need to verify error codes
func (ctx *ExecutionContext) GetErrorCode() int {
	// First check if we have a status code, which is common for HTTP errors
	if ctx.lastStatusCode > 0 {
		return ctx.lastStatusCode
	}
	
	// If no status code is available, try to extract from the error itself
	// This could be extended based on the specific error types used in the implementation
	return 0
}

// GetHTTPClient returns the current HTTP client
func (ctx *ExecutionContext) GetHTTPClient() *http.Client {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	return ctx.httpClient
}

// SetHTTPClient sets the HTTP client
func (ctx *ExecutionContext) SetHTTPClient(client *http.Client) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.httpClient = client
}

// SetLastStep stores the current step ID or description
func (ctx *ExecutionContext) SetLastStep(step string) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.lastStep = step
}

// GetLastStep retrieves the most recent step ID or description
func (ctx *ExecutionContext) GetLastStep() string {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	return ctx.lastStep
}

// SetLastResponse stores the most recent HTTP response body
func (ctx *ExecutionContext) SetLastResponse(response []byte) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.lastResponse = response
}

// GetLastResponse retrieves the most recent HTTP response body
func (ctx *ExecutionContext) GetLastResponse() []byte {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	return ctx.lastResponse
}

// GetExtractorRegistry returns the extractor registry
func (c *ExecutionContext) GetExtractorRegistry() interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.extractorRegistry
}

// SetExtractorRegistry sets the extractor registry
func (c *ExecutionContext) SetExtractorRegistry(registry interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.extractorRegistry = registry
}
