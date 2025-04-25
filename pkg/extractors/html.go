// Package extractors provides the implementation for data extraction from responses.
// This file specifically implements HTML-based extraction using CSS selectors.
package extractors

import (
	"fmt"
	"log/slog"
	"strings"
	
	"vpr/pkg/context"
	"vpr/pkg/poc"
	
	"github.com/PuerkitoBio/goquery"
)

// extractFromHTMLHandler extracts data from HTML responses using CSS selectors
func extractFromHTMLHandler(ctx *context.ExecutionContext, action *poc.HTTPResponseAction, data interface{}) (interface{}, error) {
	if action.Type != "extract_from_html" {
		return nil, fmt.Errorf("invalid extractor type for extractFromHTMLHandler: %s", action.Type)
	}
	
	// Validate required fields
	if action.CSSSelector == "" && action.XPath == "" {
		return nil, fmt.Errorf("extract_from_html requires either css_selector or xpath field")
	}
	
	if action.TargetVariable == "" {
		return nil, fmt.Errorf("extract_from_html requires target_variable field")
	}
	
	slog.Debug("Extracting from HTML", 
		"target_variable", action.TargetVariable,
		"css_selector", action.CSSSelector,
		"xpath", action.XPath)
	
	// Get the source HTML content
	var htmlContent string
	
	// Determine source of HTML data
	if action.Source != "" {
		// Source specified, try to resolve from context
		sourceData, err := ctx.ResolveVariable(action.Source)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve source variable '%s': %w", action.Source, err)
		}
		
		// Convert source to string
		if strData, ok := sourceData.(string); ok {
			htmlContent = strData
		} else {
			return nil, fmt.Errorf("source data is not a string")
		}
	} else if httpResp, ok := data.(map[string]interface{}); ok {
		// No source specified, use the data from HTTP response
		if bodyStr, ok := httpResp["body"].(string); ok {
			htmlContent = bodyStr
		} else {
			return nil, fmt.Errorf("HTTP response body is not a string")
		}
	} else {
		return nil, fmt.Errorf("data is not an HTTP response")
	}
	
	// Create a goquery document
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}
	
	var result interface{}
	
	// Extract data based on selector type
	if action.CSSSelector != "" {
		// Resolve any variables in the CSS selector
		cssSelector, err := ctx.Substitute(action.CSSSelector)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve css_selector: %w", err)
		}
		
		// Extract data using CSS selector
		result, err = extractWithCSSSelector(doc, cssSelector, action.Attribute, action.ExtractAll)
		if err != nil {
			return nil, fmt.Errorf("failed to extract data using CSS selector: %w", err)
		}
	} else if action.XPath != "" {
		// Resolve any variables in the XPath
		xpath, err := ctx.Substitute(action.XPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve xpath: %w", err)
		}
		
		// Extract data using XPath
		result, err = extractWithXPath(doc, xpath, action.Attribute, action.ExtractAll)
		if err != nil {
			return nil, fmt.Errorf("failed to extract data using XPath: %w", err)
		}
	}
	
	// Set variable with extracted data
	if err := ctx.SetVariable(action.TargetVariable, result); err != nil {
		return nil, fmt.Errorf("failed to set target variable: %w", err)
	}
	
	slog.Info("Extracted data from HTML", 
		"target_variable", action.TargetVariable,
		"result_type", fmt.Sprintf("%T", result))
	
	return result, nil
}

// extractWithCSSSelector extracts data from HTML using CSS selectors
func extractWithCSSSelector(doc *goquery.Document, selector string, attribute string, extractAll bool) (interface{}, error) {
	selection := doc.Find(selector)
	
	if selection.Length() == 0 {
		return nil, fmt.Errorf("no elements found for selector: %s", selector)
	}
	
	// If extractAll is true, return a list of all matches
	if extractAll {
		var results []string
		
		selection.Each(func(i int, s *goquery.Selection) {
			if attribute != "" {
				// Extract attribute if specified
				if val, exists := s.Attr(attribute); exists {
					results = append(results, val)
				}
			} else {
				// Extract text content
				results = append(results, strings.TrimSpace(s.Text()))
			}
		})
		
		return results, nil
	} else {
		// Get only the first match
		s := selection.First()
		
		if attribute != "" {
			// Extract attribute if specified
			if val, exists := s.Attr(attribute); exists {
				return val, nil
			}
			return "", fmt.Errorf("attribute '%s' not found", attribute)
		}
		
		// Extract text content
		return strings.TrimSpace(s.Text()), nil
	}
}

// extractWithXPath extracts data from HTML using XPath expressions
// Note: goquery doesn't support XPath directly, so we use a hybrid approach
// For proper XPath support, you might want to consider adding a dependency like "github.com/antchfx/htmlquery"
func extractWithXPath(doc *goquery.Document, xpath string, attribute string, extractAll bool) (interface{}, error) {
	// This is a simplified implementation that handles only basic XPath expressions
	// For a more complete solution, consider using a dedicated XPath library
	
	// Simple mapping of some XPath expressions to CSS selectors
	cssSelector, err := xpathToCSSSelector(xpath)
	if err != nil {
		return nil, fmt.Errorf("unsupported XPath expression: %s", xpath)
	}
	
	// Use the CSS selector extraction mechanism with the mapped selector
	return extractWithCSSSelector(doc, cssSelector, attribute, extractAll)
}

// xpathToCSSSelector converts simple XPath expressions to CSS selectors
// This is a limited implementation that only handles basic cases
func xpathToCSSSelector(xpath string) (string, error) {
	// Handle basic element selection
	if strings.HasPrefix(xpath, "//") {
		return strings.TrimPrefix(xpath, "//"), nil
	}
	
	// Handle direct child selection
	if strings.Contains(xpath, "/") {
		xpath = strings.ReplaceAll(xpath, "//", " ")
		xpath = strings.ReplaceAll(xpath, "/", " > ")
		return strings.TrimSpace(xpath), nil
	}
	
	// Handle attribute selection (very basic implementation)
	if strings.Contains(xpath, "[@") && strings.Contains(xpath, "]") {
		parts := strings.Split(xpath, "[@")
		element := strings.TrimPrefix(parts[0], "//")
		
		attrPart := strings.TrimSuffix(parts[1], "]")
		if strings.Contains(attrPart, "=") {
			attrParts := strings.Split(attrPart, "=")
			attrName := strings.TrimSpace(attrParts[0])
			attrValue := strings.Trim(strings.TrimSpace(attrParts[1]), "'\"")
			
			return fmt.Sprintf("%s[%s='%s']", element, attrName, attrValue), nil
		}
		
		return fmt.Sprintf("%s[%s]", element, attrPart), nil
	}
	
	return "", fmt.Errorf("cannot convert XPath to CSS selector: %s", xpath)
}

func init() {
	// Register HTML extractor handler
	MustRegisterExtractor("extract_from_html", extractFromHTMLHandler)
}
