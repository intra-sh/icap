package icap

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"time"
)

const reqSimulationServerAddr = "localhost:11355"

func SimulateRequestHandling(icapMethod string, inputHttpHeaders []string, httpBody string, xUrl string, handler func(ResponseWriter, *Request)) (string, error) {
	// Define the HTTP request headers and body separately
	request := ""
	switch icapMethod {
	case "OPTIONS":
		return "", nil
	case "REQMOD":
		httpHeaders := ""
		for _, arg := range inputHttpHeaders {
			httpHeaders = httpHeaders + arg + "\r\n"
		}
		httpHeaders += "\r\n"

		// Calculate the length of HTTP headers for the Encapsulated header
		httpHeadersLen := len(httpHeaders)
		request = fmt.Sprintf("REQMOD icap://icap-server.net/modify ICAP/1.0\r\n"+
			"Host: icap-server.net\r\n"+
			Optional(xUrl != "", fmt.Sprintf("X-Original-URL: %s\r\n", xUrl), "")+
			Optional(httpBody != "", fmt.Sprintf("Encapsulated: req-hdr=0, req-body=%d\r\n", httpHeadersLen), "Encapsulated: req-hdr=0")+
			"\r\n"+
			"%s"+
			"%x\r\n"+
			"%s\r\n"+
			"0\r\n"+
			"\r\n", httpHeaders, len(httpBody), httpBody)

	case "RESPMOD":
		httpHeaders := ""
		for _, arg := range inputHttpHeaders {
			httpHeaders = httpHeaders + arg + "\r\n"
		}
		httpHeaders += fmt.Sprintf("Content-Length: %d\r\n", len(httpBody))
		httpHeaders += "\r\n"

		// Calculate the length of HTTP headers for the Encapsulated header
		httpHeadersLen := len(httpHeaders)

		// Build the complete ICAP request with computed Encapsulated value
		request = fmt.Sprintf("RESPMOD icap://icap-server.net/modify ICAP/1.0\r\n"+
			"Host: icap-server.net\r\n"+
			Optional(xUrl != "", fmt.Sprintf("X-ICAP-Request-URL: %s\r\n", xUrl), "")+
			Optional(httpBody != "", fmt.Sprintf("Encapsulated: res-hdr=0, res-body=%d\r\n", httpHeadersLen), "Encapsulated: res-hdr=0")+
			"\r\n"+
			"%s"+
			"%x\r\n"+
			"%s\r\n"+
			"0\r\n"+
			"\r\n", httpHeaders, len(httpBody), httpBody)
	default:
		return "", nil
	}

	HandleFunc("/modify", handler)
	go ListenAndServeDebug(reqSimulationServerAddr, nil)

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	conn, err := net.Dial("tcp", reqSimulationServerAddr)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	io.WriteString(conn, request)

	// Set a deadline to prevent the test from hanging
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	// Use bufio.Reader instead of io.ReadFull to avoid hanging
	reader := bufio.NewReader(conn)
	respBuffer := make([]byte, 4096) // Use a larger buffer size
	n, err := reader.Read(respBuffer)
	if err != nil {
		return "", err
	}

	response := string(respBuffer[:n])
	return response, nil
}

func Optional(condition bool, a string, b string) string {
	if condition {
		return a
	}
	return b
}
