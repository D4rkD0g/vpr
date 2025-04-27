// Package extractors provides the registry and implementation of all response data extractors
// supported by the VPR engine. Extractors are used to extract and transform data from
// HTTP responses, storing results in context variables for use in subsequent steps.
package extractors

import (
	"fmt"
	"sync"
	"vpr/pkg/context"
	"vpr/pkg/poc"
)

// ExtractorHandler is the function signature for extractor execution handlers
type ExtractorHandler func(ctx *context.ExecutionContext, action *poc.HTTPResponseAction, data interface{}) (interface{}, error)

// ExtractorRegistry manages the registration and lookup of extractor handlers
type ExtractorRegistry struct {
	mu       sync.RWMutex
	handlers map[string]ExtractorHandler
}

// NewExtractorRegistry creates a new empty extractor registry
func NewExtractorRegistry() *ExtractorRegistry {
	return &ExtractorRegistry{
		handlers: make(map[string]ExtractorHandler),
	}
}

// Register adds a new extractor handler to the registry
func (r *ExtractorRegistry) Register(extractorType string, handler ExtractorHandler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.handlers[extractorType]; exists {
		return fmt.Errorf("extractor handler for type '%s' is already registered", extractorType)
	}

	r.handlers[extractorType] = handler
	return nil
}

// MustRegister adds a new extractor handler to the registry, panicking if it fails
func (r *ExtractorRegistry) MustRegister(extractorType string, handler ExtractorHandler) {
	if err := r.Register(extractorType, handler); err != nil {
		panic(err)
	}
}

// Get retrieves an extractor handler by type
func (r *ExtractorRegistry) Get(extractorType string) ExtractorHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()

	handler, exists := r.handlers[extractorType]
	if !exists {
		return nil
	}

	return handler
}

// Execute runs an extractor using the appropriate handler
func (r *ExtractorRegistry) Execute(ctx *context.ExecutionContext, action *poc.HTTPResponseAction, data interface{}) (interface{}, error) {
	// Sanity check
	if action == nil {
		return nil, fmt.Errorf("cannot execute nil extractor action")
	}

	if action.Type == "" {
		return nil, fmt.Errorf("extractor missing required 'type' field")
	}

	if action.TargetVariable == "" {
		return nil, fmt.Errorf("extractor missing required 'target_variable' field")
	}

	// Get the handler
	handler := r.Get(action.Type)
	if handler == nil {
		return nil, fmt.Errorf("no handler registered for extractor type '%s'", action.Type)
	}

	// Execute the extractor
	extractedValue, err := handler(ctx, action, data)
	if err != nil {
		return nil, err
	}

	// Store the extracted value in the context
	err = ctx.SetVariable(action.TargetVariable, extractedValue)
	if err != nil {
		return nil, fmt.Errorf("failed to set target variable '%s': %w", action.TargetVariable, err)
	}

	return extractedValue, nil
}

// Global instance for convenience
var DefaultRegistry = NewExtractorRegistry()

// RegisterExtractor registers an extractor handler with the default registry
func RegisterExtractor(extractorType string, handler ExtractorHandler) error {
	return DefaultRegistry.Register(extractorType, handler)
}

// MustRegisterExtractor registers an extractor handler with the default registry, panicking if it fails
func MustRegisterExtractor(extractorType string, handler ExtractorHandler) {
	DefaultRegistry.MustRegister(extractorType, handler)
}

// ExecuteExtractor executes an extractor using the default registry
func ExecuteExtractor(ctx *context.ExecutionContext, action *poc.HTTPResponseAction, data interface{}) (interface{}, error) {
	return DefaultRegistry.Execute(ctx, action, data)
}

// InitRegistry creates a new registry with all standard extractors registered
func InitRegistry() *ExtractorRegistry {
	registry := NewExtractorRegistry()
	
	// Register all standard extractors
	registry.MustRegister("extract_from_html", extractFromHTMLHandler)
	registry.MustRegister("extract_from_xml", extractFromXMLHandler)
	registry.MustRegister("extract_from_json", extractFromJSONHandler)
	registry.MustRegister("extract_from_regex", extractFromRegexHandler)
	registry.MustRegister("extract_from_header", extractFromHeaderHandler)
	
	return registry
}

// InitStandardExtractors registers all standard extractors defined in the DSL specification
func InitStandardExtractors() {
	// From Section 10.5 of the DSL specification
	MustRegisterExtractor("extract_from_json", nil)      // TODO: Implement
	MustRegisterExtractor("extract_from_header", nil)    // TODO: Implement
	MustRegisterExtractor("extract_from_body_regex", nil) // TODO: Implement
	MustRegisterExtractor("extract_from_html", nil)      // TODO: Implement
	MustRegisterExtractor("extract_from_xml", nil)       // TODO: Implement
	
	// These handlers will be implemented in separate files
}
