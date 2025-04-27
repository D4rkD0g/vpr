// Package extractors provides the implementation for data extraction from responses.
// This file specifically implements XML-based extraction using XPath expressions.
package extractors

import (
	"fmt"
	"log/slog"
	"strings"
	
	"vpr/pkg/context"
	"vpr/pkg/poc"
	
	"github.com/antchfx/xmlquery"
)

// extractFromXMLHandler extracts data from XML responses using XPath expressions
func extractFromXMLHandler(ctx *context.ExecutionContext, action *poc.HTTPResponseAction, data interface{}) (interface{}, error) {
	if action.Type != "extract_from_xml" {
		return nil, fmt.Errorf("invalid extractor type for extractFromXMLHandler: %s", action.Type)
	}
	
	// Validate required fields
	if action.XPath == "" {
		return nil, fmt.Errorf("extract_from_xml requires xpath field")
	}
	
	if action.TargetVariable == "" {
		return nil, fmt.Errorf("extract_from_xml requires target_variable field")
	}
	
	slog.Debug("Extracting from XML", 
		"target_variable", action.TargetVariable,
		"xpath", action.XPath)
	
	// Get the source XML content
	var xmlContent string
	
	// Determine source of XML data
	if action.Source != "" {
		// Source specified, try to resolve from context
		sourceData, err := ctx.ResolveVariable(action.Source)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve source variable '%s': %w", action.Source, err)
		}
		
		// Convert source to string
		if strData, ok := sourceData.(string); ok {
			xmlContent = strData
		} else {
			return nil, fmt.Errorf("source data is not a string")
		}
	} else if httpResp, ok := data.(map[string]interface{}); ok {
		// No source specified, use the data from HTTP response
		if bodyStr, ok := httpResp["body"].(string); ok {
			xmlContent = bodyStr
		} else {
			return nil, fmt.Errorf("HTTP response body is not a string")
		}
	} else {
		return nil, fmt.Errorf("data is not an HTTP response")
	}
	
	// Parse XML document
	doc, err := xmlquery.Parse(strings.NewReader(xmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}
	
	// Resolve any variables in the XPath
	xpath, err := ctx.Substitute(action.XPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve xpath: %w", err)
	}
	
	// Execute XPath query
	nodes, err := xmlquery.QueryAll(doc, xpath)
	if err != nil {
		return nil, fmt.Errorf("failed to execute XPath query: %w", err)
	}
	
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no nodes found for XPath: %s", xpath)
	}
	
	var result interface{}
	
	// Check if we're dealing with attribute nodes (XPath like //element/@attr)
	isAttributeXPath := strings.Contains(xpath, "/@")
	
	// Extract the requested data from nodes
	if action.ExtractAll {
		// Extract all matching nodes
		results := make([]string, 0, len(nodes))
		
		for _, node := range nodes {
			if isAttributeXPath {
				// When XPath selects attributes directly (e.g., //user/@id), 
				// the node.Data is the attribute name, but we want the value
				results = append(results, node.InnerText())
			} else if action.Attribute != "" {
				// Extract attribute if specified
				for _, attr := range node.Attr {
					if attr.Name.Local == action.Attribute {
						results = append(results, attr.Value)
						break
					}
				}
			} else {
				// Extract node content
				content := getNodeContent(node)
				results = append(results, content)
			}
		}
		
		result = results
	} else {
		// Extract only the first node
		node := nodes[0]
		
		if isAttributeXPath {
			// When XPath selects attributes directly (e.g., //user/@id),
			// we need to get the attribute value, not the name
			result = node.InnerText()
		} else if action.Attribute != "" {
			// Extract attribute if specified
			var attributeValue string
			attributeFound := false
			
			for _, attr := range node.Attr {
				if attr.Name.Local == action.Attribute {
					attributeValue = attr.Value
					attributeFound = true
					break
				}
			}
			
			if !attributeFound {
				return nil, fmt.Errorf("attribute '%s' not found", action.Attribute)
			}
			
			result = attributeValue
		} else {
			// Extract node content
			result = getNodeContent(node)
		}
	}
	
	// Create a proper variable structure that matches the ContextVariable format
	varStruct := &poc.ContextVariable{
		ID:    action.TargetVariable,
		Value: result,
	}

	// Store the extracted data in the variables map
	varsPath := "variables." + action.TargetVariable
	if err := ctx.SetVariable(varsPath, varStruct); err != nil {
		return nil, fmt.Errorf("failed to set target variable: %w", err)
	}

	slog.Debug("XML extraction successful",
		"target_variable", action.TargetVariable,
		"extracted_value", result)
	
	return result, nil
}

// getNodeContent extracts the content from an XML node
// It returns the inner text (or own text for leaf nodes)
func getNodeContent(node *xmlquery.Node) string {
	switch node.Type {
	case xmlquery.ElementNode:
		if node.FirstChild == nil {
			return ""
		}
		// If the node has only one child and it's a text node, return its content
		if node.FirstChild == node.LastChild && node.FirstChild.Type == xmlquery.TextNode {
			return strings.TrimSpace(node.FirstChild.Data)
		}
		// Otherwise, return the inner XML
		var sb strings.Builder
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if child.Type == xmlquery.TextNode {
				sb.WriteString(child.Data)
			}
		}
		return strings.TrimSpace(sb.String())
	case xmlquery.TextNode, xmlquery.CharDataNode, xmlquery.CommentNode:
		return strings.TrimSpace(node.Data)
	case xmlquery.AttributeNode:
		return strings.TrimSpace(node.Data)
	default:
		return strings.TrimSpace(node.InnerText())
	}
}

func init() {
	// Register XML extractor handler
	MustRegisterExtractor("extract_from_xml", extractFromXMLHandler)
}
