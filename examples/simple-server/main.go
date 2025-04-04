// This example demonstrates how to create a basic ICAP server that handles
// REQMOD and RESPMOD requests.
package main

import (
	"fmt"
	"os"

	"github.com/intra-sh/icap"
)

var ISTag = "\"GOLANG\""

func main() {
	// Handle REQMOD requests
	icap.HandleFunc("/reqmod", reqmodHandler)

	// Handle RESPMOD requests
	icap.HandleFunc("/respmod", respmodHandler)

	// Start the server
	fmt.Println("Starting ICAP server on port 1344...")
	if err := icap.ListenAndServe(":1344", nil); err != nil {
		fmt.Println("Error starting server:", err)
		os.Exit(1)
	}
}

// reqmodHandler handles REQMOD requests
func reqmodHandler(w icap.ResponseWriter, req *icap.Request) {
	h := w.Header()
	h.Set("ISTag", ISTag)
	h.Set("Service", "ICAP Go Reqmod Service")

	switch req.Method {
	case "OPTIONS":
		h.Set("Methods", "REQMOD")
		h.Set("Allow", "204")
		h.Set("Preview", "0")
		h.Set("Transfer-Preview", "*")
		w.WriteHeader(200, nil, false)
		fmt.Println("OPTIONS request processed")
	case "REQMOD":
		// Modify the request
		if req.Request != nil {
			// Add a custom header to the HTTP request
			req.Request.Header.Add("X-ICAP-Processed", "true")

			// Log some information
			fmt.Printf("Processing request to: %s\n", req.Request.URL)
		}
		w.WriteHeader(200, req.Request, false)
		fmt.Println("REQMOD request processed")
	default:
		w.WriteHeader(405, nil, false)
		fmt.Println("Invalid request method:", req.Method)
	}
}

// respmodHandler handles RESPMOD requests
func respmodHandler(w icap.ResponseWriter, req *icap.Request) {
	h := w.Header()
	h.Set("ISTag", ISTag)
	h.Set("Service", "ICAP Go Respmod Service")

	switch req.Method {
	case "OPTIONS":
		h.Set("Methods", "RESPMOD")
		h.Set("Allow", "204")
		h.Set("Preview", "0")
		h.Set("Transfer-Preview", "*")
		w.WriteHeader(200, nil, false)
		fmt.Println("OPTIONS request processed")
	case "RESPMOD":
		// Process the response
		if req.Response != nil {
			// Add a custom header to the HTTP response
			req.Response.Header.Add("X-ICAP-Processed", "true")
			fmt.Println("Processing response from:", req.Request.URL)
		}
		w.WriteHeader(200, req.Response, false)
		fmt.Println("RESPMOD request processed")
	default:
		w.WriteHeader(405, nil, false)
		fmt.Println("Invalid request method:", req.Method)
	}
}
