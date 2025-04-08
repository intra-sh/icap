// Copyright 2011 Andy Balholm. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package icap

import (
	"io"
	"testing"
)

const reqTestServerAddr = "localhost:11355"

// Test case for modifying an ICAP request by adding headers
func TestRequestModification(t *testing.T) {
	response, err := SimulateRequestHandling("REQMOD",
		[]string{
			"GET /example.html HTTP/1.1",
			"Host: www.example.com",
			"Accept: text/html",
		},
		"This is a test request body.",
		"https://www.example.com/example.html",
		handleRequestModification)

	expectedResp := "ICAP/1.0 200 OK\r\n" +
		"Connection: close\r\n" +
		"Date: Mon, 10 Jan 2000 09:55:21 GMT\r\n" +
		"Encapsulated: req-hdr=0, req-body=163\r\n" +
		"Server: ICAP-Test-Server/1.0\r\n" +
		"X-Original-Url: https://www.example.com/example.html\r\n" +
		"\r\n" +
		"GET https://www.example.com/example.html HTTP/1.1\r\n" +
		"Accept: text/html\r\n" +
		"Host: www.example.com\r\n" +
		"Via: 1.0 icap-server.net (ICAP Test Server)\r\n" +
		"X-Icap-Modified: true\r\n" +
		"\r\n" +
		"1c\r\n" +
		"This is a test request body.\r\n" +
		"0\r\n" +
		"\r\n"

	if err != nil || response != expectedResp {
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
