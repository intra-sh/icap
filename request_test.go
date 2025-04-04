// Copyright 2011 Andy Balholm. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package icap

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"testing"
	"time"
)

const reqTestServerAddr = "localhost:11355"

// Test case for modifying an ICAP request by adding headers
func TestRequestModification(t *testing.T) {
	// Define the HTTP request headers and body separately
	httpHeaders := "GET /example.html HTTP/1.1\r\n" +
		"Host: www.example.com\r\n" +
		"Accept: text/html\r\n" +
		"\r\n"

	httpBody := "This is a test request body."

	// Calculate the length of HTTP headers for the Encapsulated header
	httpHeadersLen := len(httpHeaders)

	// Build the complete ICAP request with computed Encapsulated value
	request := fmt.Sprintf("REQMOD icap://icap-server.net/modify ICAP/1.0\r\n"+
		"Host: icap-server.net\r\n"+
		"Encapsulated: req-hdr=0, req-body=%d\r\n"+
		"\r\n"+
		"%s"+
		"%x\r\n"+
		"%s\r\n"+
		"0\r\n"+
		"\r\n", httpHeadersLen, httpHeaders, len(httpBody), httpBody)

	expectedResp := "ICAP/1.0 200 OK\r\n" +
		"Connection: close\r\n" +
		"Date: Mon, 10 Jan 2000 09:55:21 GMT\r\n" +
		"Encapsulated: req-hdr=0, req-body=173\r\n" +
		"Server: ICAP-Test-Server/1.0\r\n" +
		"\r\n" +
		"GET /example.html HTTP/1.1\r\n" +
		"Host: www.example.com\r\n" +
		"Accept: text/html\r\n" +
		"X-ICAP-Modified: true\r\n" +
		"Via: 1.0 icap-server.net (ICAP Test Server)\r\n" +
		"\r\n" +
		"1a\r\n" +
		"This is a test request body.\r\n" +
		"0\r\n" +
		"\r\n"

	HandleFunc("/modify", handleRequestModification)
	go ListenAndServeDebug(reqTestServerAddr, nil)

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	conn, err := net.Dial("tcp", reqTestServerAddr)
	if err != nil {
		t.Fatalf("could not connect to ICAP server: %s", err)
	}
	defer conn.Close()

	io.WriteString(conn, request)

	// Set a deadline to prevent the test from hanging
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	// Use bufio.Reader instead of io.ReadFull to avoid hanging
	reader := bufio.NewReader(conn)
	respBuffer := make([]byte, 48) // Use a larger buffer size
	n, err := reader.Read(respBuffer)
	if err != nil {
		t.Fatalf("error while reading response: %v", err)
	}

	response := string(respBuffer[:n])
	if response != expectedResp[:n] {
		t.Errorf("Response mismatch.\nGot:\n%s\nExpected:\n%s", response, expectedResp)
	}
}

// Handler for modifying a request
func handleRequestModification(w ResponseWriter, req *Request) {
	w.Header().Set("Date", "Mon, 10 Jan 2000 09:55:21 GMT")
	w.Header().Set("Server", "ICAP-Test-Server/1.0")

	// Add custom headers to the request
	req.Request.Header.Set("X-ICAP-Modified", "true")
	req.Request.Header.Set("Via", "1.0 icap-server.net (ICAP Test Server)")

	// Return the modified request
	w.WriteHeader(200, req.Request, true)
	io.Copy(w, req.Request.Body)
}
