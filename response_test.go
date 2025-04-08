// Copyright 2011 Andy Balholm. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package icap

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

const serverAddr = "localhost:11344"

// REQMOD example 2 from RFC 3507, adjusted for order of headers, etc.
func TestRequestModification1(t *testing.T) {
	response, err := SimulateRequestHandling("REQMOD",
		[]string{
			"POST /origin-resource/form.pl HTTP/1.1",
			"Host: www.origin-server.com",
			"Accept: text/html, text/plain",
			"Accept-Encoding: compress",
			"Cache-Control: no-cache",
		},
		"I am posting this information.",
		"",
		handleResponseModification1)
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
	if err != nil {
		t.Errorf("Response handling has errors:\n%s", err)
	}
	checkString("Response", response, resp, t)
}

func handleResponseModification1(w ResponseWriter, req *Request) {
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
func TestResponseModification2(t *testing.T) {
	xReqUrl := "https://www.example.com/example.html"
	fullResponse, err := SimulateRequestHandling("RESPMOD",
		[]string{
			"HTTP/1.1 200 OK",
			"Content-Type: text/plain",
		},
		"This is a test response body.",
		xReqUrl,
		handleResponseModification2)
	if err != nil {
		t.Errorf("Response handling has errors:\n%s", err)
	}

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

func handleResponseModification2(w ResponseWriter, req *Request) {
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
