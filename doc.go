/*
Package icap implements the Internet Content Adaptation Protocol (ICAP) as
defined in RFC 3507.

ICAP is a protocol that allows edge devices such as proxies to offload tasks
to dedicated servers. It is commonly used for content filtering, antivirus
scanning, and other content adaptation services.

This library provides both server and client implementations of the ICAP protocol.
It allows Go programs to:
  - Create ICAP servers that can process and modify HTTP requests and responses
  - Bridge between ICAP and HTTP to serve local content
  - Handle REQMOD and RESPMOD ICAP methods

Basic usage example:

	package main

	import (
		"fmt"
		"github.com/intra-sh/icap"
		"net/http"
	)

	func main() {
		icap.HandleFunc("/example", exampleHandler)
		fmt.Println("Starting ICAP server on port 1344...")
		if err := icap.ListenAndServe(":1344", nil); err != nil {
			fmt.Println("Error starting server:", err)
		}
	}

	func exampleHandler(w icap.ResponseWriter, req *icap.Request) {
		h := w.Header()
		h.Set("ISTag", "\"GOLANG-ICAP\"")

		switch req.Method {
		case "OPTIONS":
			h.Set("Methods", "REQMOD, RESPMOD")
			h.Set("Allow", "204")
			w.WriteHeader(200, nil, false)
		case "REQMOD", "RESPMOD":
			w.WriteHeader(204, nil, false) // No modifications
		default:
			w.WriteHeader(405, nil, false)
		}
	}
*/
package icap
