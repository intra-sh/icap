// Copyright 2011 Andy Balholm. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package icap

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

const serverAddr = "localhost:11344"

// REQMOD example 2 from RFC 3507, adjusted for order of headers, etc.
func TestREQMOD2(t *testing.T) {
	request :=
		"REQMOD icap://icap-server.net/server?arg=87 ICAP/1.0\r\n" +
			"Host: icap-server.net\r\n" +
			"Encapsulated: req-hdr=0, req-body=154\r\n" +
			"\r\n" +
			"POST /origin-resource/form.pl HTTP/1.1\r\n" +
			"Host: www.origin-server.com\r\n" +
			"Accept: text/html, text/plain\r\n" +
			"Accept-Encoding: compress\r\n" +
			"Cache-Control: no-cache\r\n" +
			"\r\n" +
			"1e\r\n" +
			"I am posting this information.\r\n" +
			"0\r\n" +
			"\r\n"
	resp :=
		"ICAP/1.0 200 OK\r\n" +
			"Connection: close\r\n" +
			"Date: Mon, 10 Jan 2000  09:55:21 GMT\r\n" +
			"Encapsulated: req-hdr=0, req-body=231\r\n" +
			"Istag: \"W3E4R7U9-L2E4-2\"\r\n" +
			"Server: ICAP-Server-Software/1.0\r\n" +
			"\r\n" +
			"POST /origin-resource/form.pl HTTP/1.1\r\n" +
			"Accept: text/html, text/plain, image/gif\r\n" +
			"Accept-Encoding: gzip, compress\r\n" +
			"Cache-Control: no-cache\r\n" +
			"Host: www.origin-server.com\r\n" +
			"Via: 1.0 icap-server.net (ICAP Example ReqMod Service 1.1)\r\n" +
			"\r\n" +
			"2d\r\n" +
			"I am posting this information.  ICAP powered!\r\n" +
			"0\r\n" +
			"\r\n"

	HandleFunc("/server", HandleREQMOD2)
	go ListenAndServe(serverAddr, nil)

	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		t.Fatalf("could not connect to ICAP server on localhost: %s", err)
	}

	io.WriteString(conn, request)
	respBuffer := make([]byte, len(resp))
	_, err = io.ReadFull(conn, respBuffer)

	if err != nil {
		t.Fatalf("error while reading response: %v", err)
	}

	response := string(respBuffer)
	checkString("Response", response, resp, t)
}

func HandleREQMOD2(w ResponseWriter, req *Request) {
	w.Header().Set("Date", "Mon, 10 Jan 2000  09:55:21 GMT")
	w.Header().Set("Server", "ICAP-Server-Software/1.0")
	w.Header().Set("ISTag", "\"W3E4R7U9-L2E4-2\"")

	req.Request.Header.Set("Via", "1.0 icap-server.net (ICAP Example ReqMod Service 1.1)")
	req.Request.Header.Set("Accept", "text/html, text/plain, image/gif")
	req.Request.Header.Set("Accept-Encoding", "gzip, compress")

	body, _ := io.ReadAll(req.Request.Body)
	newBody := string(body) + "  ICAP powered!"

	w.WriteHeader(200, req.Request, true)
	io.WriteString(w, newBody)
}

// Test case for modifying an ICAP response by adding headers
func TestResponseModification(t *testing.T) {
	// Define the HTTP response headers and body separately
	httpBody := "This is a test response body."
	httpHeaders := fmt.Sprintf("HTTP/1.1 200 OK\r\n"+
		"Content-Type: text/plain\r\n"+
		"Content-Length: %d\r\n"+
		"\r\n", len(httpBody))

	// Calculate the length of HTTP headers for the Encapsulated header
	httpHeadersLen := len(httpHeaders)

	xReqUrl := "https://www.example.com/example.html"

	// Build the complete ICAP request with computed Encapsulated value
	request := fmt.Sprintf("RESPMOD icap://icap-server.net/modify ICAP/1.0\r\n"+
		"Host: icap-server.net\r\n"+
		"X-ICAP-Request-URL: %s\r\n"+
		"Encapsulated: res-hdr=0, res-body=%d\r\n"+
		"\r\n"+
		"%s"+
		"%x\r\n"+
		"%s\r\n"+
		"0\r\n"+
		"\r\n", xReqUrl, httpHeadersLen, httpHeaders, len(httpBody), httpBody)

	// Register a new handler for response modification
	HandleFunc("/modify", handleResponseModification)
	go ListenAndServeDebug(reqTestServerAddr, nil)

	// Give the server a moment to get ready
	time.Sleep(100 * time.Millisecond)

	conn, err := net.Dial("tcp", reqTestServerAddr)
	if err != nil {
		t.Fatalf("could not connect to ICAP server: %s", err)
	}
	defer conn.Close()

	io.WriteString(conn, request)

	// Set a deadline to prevent the test from hanging
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	// Read the full response
	reader := bufio.NewReader(conn)
	respBuffer := make([]byte, 1024) // Use a larger buffer size
	n, err := reader.Read(respBuffer)
	if err != nil {
		t.Fatalf("error while reading response: %v", err)
	}

	fullResponse := string(respBuffer[:n])

	// Verify the response contains expected data
	if !strings.Contains(fullResponse, "ICAP/1.0 200 OK") {
		t.Errorf("Response doesn't contain expected status code:\n%s", fullResponse)
	}

	if !strings.Contains(fullResponse, "X-Icap-Modified: true") {
		t.Errorf("Response doesn't contain X-ICAP-Modified header:\n%s", fullResponse)
	}

	if !strings.Contains(fullResponse, "This is a successful modification response body") {
		t.Errorf("Response doesn't contain modified body:\n%s", fullResponse)
	}
	if !strings.Contains(fullResponse, xReqUrl) {
		t.Errorf("Response doesn't contain x-icap-request-url [%s]:\n%s", xReqUrl, fullResponse)
	}
}

func handleResponseModification(w ResponseWriter, req *Request) {
	w.Header().Set("Date", "Mon, 10 Jan 2000 09:55:21 GMT")
	w.Header().Set("Server", "ICAP-Test-Server/1.0")

	// Add custom headers to the response
	req.Response.Header.Set("X-ICAP-Modified", "true")
	req.Response.Header.Set("Via", "1.0 icap-server.net (ICAP Test Server)")

	originalBody := make([]byte, req.Response.ContentLength)
	_, err := req.Response.Body.Read(originalBody)
	if err != nil {
		// Handle error
		w.WriteHeader(500, nil, false)
		return
	}

	// Example: modify body content
	modifiedBody := bytes.Replace(
		originalBody,
		[]byte("test"),
		[]byte("successful modification"),
		-1,
	)

	// Update content length if it exists
	if req.Response.ContentLength > 0 {
		req.Response.ContentLength = int64(len(modifiedBody))
	}

	// Return the modified response
	w.WriteHeader(200, req.Response, true)
	w.Write(modifiedBody)
}
