// Simple test server for VPR functionality validation
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

const htmlResponse = `<!DOCTYPE html>
<html>
<head>
    <title>Test Page</title>
</head>
<body>
    <h1>Test Server</h1>
    <div id="content">
        <p>This is a test paragraph.</p>
        <ul class="items">
            <li class="item">Item 1</li>
            <li class="item">Item 2</li>
            <li class="item">Item 3</li>
        </ul>
    </div>
    <div id="user-info" data-user-id="12345">
        <span class="username">testuser</span>
        <span class="role">admin</span>
    </div>
</body>
</html>`

const xmlResponse = `<?xml version="1.0" encoding="UTF-8"?>
<response>
    <status>success</status>
    <data>
        <user id="12345">
            <username>testuser</username>
            <email>test@example.com</email>
            <roles>
                <role>user</role>
                <role>admin</role>
            </roles>
        </user>
        <items>
            <item id="1">Item One</item>
            <item id="2">Item Two</item>
            <item id="3">Item Three</item>
        </items>
    </data>
</response>`

func main() {
	port := "8081"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	// HTML endpoint
	http.HandleFunc("/html", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, htmlResponse)
	})

	// XML endpoint
	http.HandleFunc("/xml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprint(w, xmlResponse)
	})

	// Echo endpoint
	http.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name == "" {
			name = "Guest"
		}
		response := fmt.Sprintf(`{"message": "Hello, %s!"}`, name)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, response)
	})

	// File upload endpoint
	http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		err := r.ParseMultipartForm(10 << 20) // 10 MB max memory
		if err != nil {
			http.Error(w, "Failed to parse multipart form: "+err.Error(), http.StatusBadRequest)
			return
		}

		file, handler, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Error retrieving file: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Get other form field
		message := r.FormValue("message")
		if message == "" {
			message = "No message provided"
		}

		response := fmt.Sprintf(`{
			"status": "success",
			"message": "%s",
			"file": {
				"name": "%s",
				"size": %d,
				"type": "%s"
			}
		}`, message, handler.Filename, handler.Size, handler.Header.Get("Content-Type"))

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, response)
	})

	log.Printf("Starting test server on port %s...\n", port)
	log.Printf("Endpoints available:\n")
	log.Printf("  - http://localhost:%s/html - HTML content\n", port)
	log.Printf("  - http://localhost:%s/xml - XML content\n", port)
	log.Printf("  - http://localhost:%s/echo?name=YourName - Echo name\n", port)
	log.Printf("  - http://localhost:%s/upload - File upload (POST with multipart form)\n", port)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
