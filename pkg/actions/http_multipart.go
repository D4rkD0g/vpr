// Package actions defines the interface and registry for runnable actions.
// This file implements multipart/form-data processing for HTTP requests,
// particularly focusing on file upload support as defined in the DSL spec.
package actions

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	
	"vpr/pkg/context"
	"vpr/pkg/poc"
)

// HTTPMultipartProcessor processes multipart form data for HTTP requests
// This is used internally by the buildHTTPRequest function in http.go
func processMultipartForm(ctx *context.ExecutionContext, action *poc.Action, req *http.Request) error {
	if action.Request.Multipart == nil {
		return fmt.Errorf("body_type is 'multipart' but no multipart configuration provided")
	}
	
	// Create multipart buffer and writer
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	
	// Process file uploads
	if action.Request.Multipart.Files != nil && len(action.Request.Multipart.Files) > 0 {
		for _, fileUpload := range action.Request.Multipart.Files {
			if err := addFileToMultipart(ctx, w, fileUpload); err != nil {
				w.Close()
				return fmt.Errorf("failed to add file to multipart form: %w", err)
			}
		}
	}
	
	// Process form fields
	if action.Request.Multipart.Data != nil {
		for key, value := range action.Request.Multipart.Data {
			// Resolve any variables in field values
			resolvedValue, err := ctx.Substitute(value)
			if err != nil {
				w.Close()
				return fmt.Errorf("failed to resolve field value for '%s': %w", key, err)
			}
			
			if err := w.WriteField(key, resolvedValue); err != nil {
				w.Close()
				return fmt.Errorf("failed to add field '%s' to multipart form: %w", key, err)
			}
		}
	}
	
	// Close the writer
	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}
	
	// Set the request body and content type
	req.Body = io.NopCloser(&b)
	req.ContentLength = int64(b.Len())
	req.Header.Set("Content-Type", w.FormDataContentType())
	
	return nil
}

// addFileToMultipart adds a file to a multipart form
func addFileToMultipart(ctx *context.ExecutionContext, w *multipart.Writer, fileUpload poc.FileUpload) error {
	// Validate required fields
	if fileUpload.ParameterName == "" {
		return fmt.Errorf("file upload requires 'parameter_name'")
	}
	
	if fileUpload.Filename == "" {
		return fmt.Errorf("file upload requires 'filename'")
	}
	
	if fileUpload.LocalPath == "" {
		return fmt.Errorf("file upload requires 'local_path'")
	}
	
	// Resolve any variables in paths and names
	resolvedLocalPath, err := ctx.Substitute(fileUpload.LocalPath)
	if err != nil {
		return fmt.Errorf("failed to resolve local_path: %w", err)
	}
	
	resolvedFilename, err := ctx.Substitute(fileUpload.Filename)
	if err != nil {
		return fmt.Errorf("failed to resolve filename: %w", err)
	}
	
	resolvedParamName, err := ctx.Substitute(fileUpload.ParameterName)
	if err != nil {
		return fmt.Errorf("failed to resolve parameter_name: %w", err)
	}
	
	// Use the resolved path directly
	filePath := resolvedLocalPath
	
	// Verify the file exists
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("file not found or not accessible at '%s': %w", filePath, err)
	}
	
	if fileInfo.IsDir() {
		return fmt.Errorf("'%s' is a directory, not a file", filePath)
	}
	
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file '%s': %w", filePath, err)
	}
	defer file.Close()
	
	// Create form file writer
	fileWriter, err := w.CreateFormFile(resolvedParamName, resolvedFilename)
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}
	
	// Copy file content to form writer
	if _, err := io.Copy(fileWriter, file); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}
	
	// Log successful file addition
	slog.Debug("Added file to multipart form", 
		"parameter", resolvedParamName,
		"filename", resolvedFilename,
		"size", fileInfo.Size())
	
	return nil
}

// Modify the buildHTTPRequest function in http.go to use this functionality 
// by adding an integration function:

// handleMultipartRequest processes a multipart form request
// This is called from buildHTTPRequest when body_type is 'multipart'
func handleMultipartRequest(ctx *context.ExecutionContext, action *poc.Action, req *http.Request) error {
	return processMultipartForm(ctx, action, req)
}
