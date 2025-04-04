# ICAP - Internet Content Adaptation Protocol for Go

[![GoDoc](https://godoc.org/github.com/intra-sh/icap?status.svg)](https://godoc.org/github.com/intra-sh/icap)
[![Go Report Card](https://goreportcard.com/badge/github.com/intra-sh/icap)](https://goreportcard.com/report/github.com/intra-sh/icap)

A Go library implementing the Internet Content Adaptation Protocol (ICAP) as defined in [RFC 3507](https://tools.ietf.org/html/rfc3507).

## Overview

ICAP is a lightweight protocol for executing a "remote procedure call" on HTTP messages. It is used to extend transparent proxy servers by providing transformation, antivirus scanning, content filtering, ad blocking, and other value-added services.

This library provides both server and client implementations of the ICAP protocol.

## Installation

```bash
go get github.com/intra-sh/icap
```

## Usage

### Creating a simple ICAP server

```go
package main

import (
	"fmt"
	"github.com/intra-sh/icap"
	"net/http"
)

func main() {
	// Handle REQMOD requests at the /reqmod endpoint
	icap.HandleFunc("/reqmod", reqmodHandler)

	// Start the server
	fmt.Println("Starting ICAP server on port 1344...")
	if err := icap.ListenAndServe(":1344", nil); err != nil {
		fmt.Println("Error starting server:", err)
	}
}

// reqmodHandler handles REQMOD requests
func reqmodHandler(w icap.ResponseWriter, req *icap.Request) {
	h := w.Header()
	h.Set("ISTag", "\"GOLANG\"")
	h.Set("Service", "ICAP Go Service")

	switch req.Method {
	case "OPTIONS":
		h.Set("Methods", "REQMOD")
		h.Set("Allow", "204")
		h.Set("Preview", "0")
		h.Set("Transfer-Preview", "*")
		w.WriteHeader(200, nil, false)
	case "REQMOD":
		// Modify the request here
		if req.Request != nil {
			// For example, add a header to the HTTP request
			req.Request.Header.Add("X-ICAP-Processed", "true")
		}
		w.WriteHeader(200, req.Request, false)
	default:
		w.WriteHeader(405, nil, false)
		fmt.Println("Invalid request method")
	}
}
```

### Using the bridge to serve HTTP content locally

```go
package main

import (
	"github.com/intra-sh/icap"
)

func handler(w icap.ResponseWriter, req *icap.Request) {
	// ServeLocally allows you to use the local HTTP server to generate responses
	icap.ServeLocally(w, req)
}

func main() {
	// Set up an HTTP handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, ICAP bridged to HTTP!")
	})

	// Set up the ICAP handler
	icap.HandleFunc("/icap-bridge", handler)

	// Start the ICAP server
	icap.ListenAndServe(":1344", nil)
}
```

## Working with Request Modification (REQMOD)

Request modification mode allows the ICAP server to examine and modify HTTP requests before they reach the origin server.

### Accessing ICAP Headers

```go
func reqmodHandler(w icap.ResponseWriter, req *icap.Request) {
    // Read ICAP request headers
    preview := req.Header.Get("Preview")
    
    // Set ICAP response headers
    h := w.Header()
    h.Set("ISTag", "\"GOLANG-TAG\"")
    h.Set("Service", "ICAP Go Service")
    
    // Process request based on ICAP method
    switch req.Method {
        // ... handler code
    }
}
```

### Working with Encapsulated HTTP Request

```go
func reqmodHandler(w icap.ResponseWriter, req *icap.Request) {
    // ... ICAP headers handling
    
    if req.Request != nil {
        // Access HTTP request method, URL, and version
        httpMethod := req.Request.Method
        httpURL := req.Request.URL.String()
        
        // Access HTTP request headers
        userAgent := req.Request.Header.Get("User-Agent")
        
        // Modify HTTP request headers
        req.Request.Header.Set("X-ICAP-Modified", "true")
        req.Request.Header.Add("X-ICAP-Processed-Time", time.Now().String())
        
        // Change request destination
        req.Request.Host = "new-destination.com"
        req.Request.URL.Host = "new-destination.com"
        
        // Return modified request
        w.WriteHeader(200, req.Request, false)
    }
}
```

### Modifying HTTP Request Body

```go
func reqmodHandler(w icap.ResponseWriter, req *icap.Request) {
    // ... ICAP and HTTP headers handling
    
    if req.Request != nil && req.Method == "REQMOD" {
        // Read original body
        originalBody, err := io.ReadAll(req.Request.Body)
        if err != nil {
            // Handle error
            w.WriteHeader(500, nil, false)
            return
        }
        
        // Modify body (simple example: convert to uppercase)
        modifiedBody := bytes.ToUpper(originalBody)
        
        // Replace body in request with modified version
        // WriteHeader's third parameter 'true' indicates we'll be writing a body
        w.WriteHeader(200, req.Request, true)
        
        // Write modified body
        w.Write(modifiedBody)
    }
}
```

## Working with Response Modification (RESPMOD)

Response modification mode allows the ICAP server to examine and modify HTTP responses before they reach the client.

### Accessing ICAP Headers in RESPMOD

```go
func respmodHandler(w icap.ResponseWriter, req *icap.Request) {
    // Read ICAP request headers
    preview := req.Header.Get("Preview")
    
    // Set ICAP response headers
    h := w.Header()
    h.Set("ISTag", "\"GOLANG-RESP-TAG\"")
    h.Set("Service", "ICAP Respmod Service")
    
    switch req.Method {
    case "OPTIONS":
        h.Set("Methods", "RESPMOD")
        h.Set("Allow", "204")
        h.Set("Preview", "0")
        w.WriteHeader(200, nil, false)
    case "RESPMOD":
        // Response modification code
    }
}
```

### Working with Encapsulated HTTP Response

```go
func respmodHandler(w icap.ResponseWriter, req *icap.Request) {
    // ... ICAP headers handling
    
    if req.Response != nil {
        // Access HTTP response status
        statusCode := req.Response.StatusCode
        statusText := req.Response.Status
        
        // Access HTTP response headers
        contentType := req.Response.Header.Get("Content-Type")
        
        // Modify HTTP response headers
        req.Response.Header.Set("X-ICAP-Modified", "true")
        req.Response.Header.Add("X-ICAP-Scanned", "clean")
        
        // Access the request URL (that generated this response)
        originalURL := req.Request.URL.String()
        
        // Return modified response
        w.WriteHeader(200, req.Response, false)
    }
}
```

### Modifying HTTP Response Body

```go
func respmodHandler(w icap.ResponseWriter, req *icap.Request) {
    // ... ICAP and HTTP headers handling
    
    if req.Response != nil && req.Method == "RESPMOD" {
        // Read original response body
        originalBody, err := io.ReadAll(req.Response.Body)
        if err != nil {
            // Handle error
            w.WriteHeader(500, nil, false)
            return
        }
        
        // Example: modify HTML content
        modifiedBody := bytes.Replace(
            originalBody, 
            []byte("<title>"), 
            []byte("<title>Modified by ICAP: "), 
            -1,
        )
        
        // Write response with modified body
        w.WriteHeader(200, req.Response, true)
        w.Write(modifiedBody)
    }
}
```

### Handling No Modifications

If you don't need to modify the request or response, you can use 204 status code:

```go
// No modifications needed
w.WriteHeader(204, nil, false)
```

## Status Codes

Common ICAP status codes:
- 200: OK - Successful modification
- 204: No Modifications - Content should not be modified
- 400: Bad Request - Malformed ICAP request
- 404: ICAP Service Not Found
- 405: Method Not Allowed
- 500: Server Error
- 501: Method Not Implemented

## License

BSD - See [LICENSE](LICENSE) for details