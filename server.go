// Copyright 2011 Andy Balholm. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Network connections and request dispatch for the ICAP server.

package icap

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"runtime/debug"
	"time"
)

// Handler Objects implementing the Handler interface can be registered
// to serve ICAP requests.
//
// ServeICAP should write reply headers and data to the ResponseWriter
// and then return.
type Handler interface {
	ServeICAP(ResponseWriter, *Request)
}

// The HandlerFunc type is an adapter to allow the use of
// ordinary functions as ICAP handlers.  If f is a function
// with the appropriate signature, HandlerFunc(f) is a
// Handler object that calls f.
type HandlerFunc func(ResponseWriter, *Request)

// ServeICAP calls f(w, r).
func (f HandlerFunc) ServeICAP(w ResponseWriter, r *Request) {
	f(w, r)
}

// A conn represents the server side of an ICAP connection.
type conn struct {
	remoteAddr string            // network address of remote side
	handler    Handler           // request handler
	rwc        net.Conn          // i/o connection
	buf        *bufio.ReadWriter // buffered rwc
}

// Create new connection from rwc.
func newConn(rwc net.Conn, handler Handler) (c *conn, err error) {
	c = new(conn)
	c.remoteAddr = rwc.RemoteAddr().String()
	c.handler = handler
	c.rwc = rwc
	br := bufio.NewReader(rwc)
	bw := bufio.NewWriter(rwc)
	c.buf = bufio.NewReadWriter(br, bw)

	return c, nil
}

// Read next request from connection.
func (c *conn) readRequest() (w *respWriter, err error) {
	var req *Request
	if req, err = ReadRequest(c.buf); err != nil {
		return nil, err
	}

	if req == nil {
		req = new(Request)
	} else {
		req.RemoteAddr = c.remoteAddr
	}

	w = new(respWriter)
	w.conn = c
	w.req = req
	w.header = make(http.Header)
	return w, err
}

// Close the connection.
func (c *conn) close() {
	if c.buf != nil {
		c.buf.Flush()
		c.buf = nil
	}
	if c.rwc != nil {
		c.rwc.Close()
		c.rwc = nil
	}
}

// Serve a new connection.
func (c *conn) serve(debugLevel int) {
	defer func() {
		err := recover()
		if err == nil {
			return
		}

		var buf bytes.Buffer
		fmt.Fprintf(&buf, "icap: panic serving %v: %v\n", c.remoteAddr, err)
		buf.Write(debug.Stack())
		log.Print(buf.String())
	}()
	for {
		var w *respWriter
		w, err := c.readRequest()
		// In a case of parsing error there should be an option to handle a dummy request to not fail the whole service.
		if w == nil {
			c.rwc.Close()

			break
		}
		if err != nil {
			log.Println("error while reading request:", err)
			c.rwc.Close()
			break
		}

		c.handler.ServeICAP(w, w.req)
		w.finishRequest()
	}

	c.close()
}

// A Server defines parameters for running an ICAP server.
type Server struct {
	Addr         string  // TCP address to listen on, ":1344" if empty
	Handler      Handler // handler to invoke
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	DebugLevel   int
}

// ListenAndServe listens on the TCP network address srv.Addr and then
// calls Serve to handle requests on incoming connections.  If
// srv.Addr is blank, ":1344" is used.
func (srv *Server) ListenAndServe() error {
	addr := srv.Addr
	if addr == "" {
		addr = ":1344"
	}
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return srv.Serve(l)
}

// ListenAndServeTLS ---
func (srv *Server) ListenAndServeTLS(cert, key string) error {
	cer, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		return err
	}
	addr := srv.Addr
	if addr == "" {
		addr = ":1344"
	}
	config := &tls.Config{Certificates: []tls.Certificate{cer}}
	l, err := tls.Listen("tcp", addr, config)
	if err != nil {
		return err
	}
	return srv.Serve(l)
}

// Serve accepts incoming connections on the Listener l, creating a
// new service thread for each.  The service threads read requests and
// then call srv.Handler to reply to them.
func (srv *Server) Serve(l net.Listener) error {
	defer l.Close()
	handler := srv.Handler
	if handler == nil {
		handler = DefaultServeMux
	}

	for {
		rw, err := l.Accept()
		if err != nil {
			// Instead of using the deprecated ne.Temporary(), check for specific error types
			// or just log and continue for non-critical errors
			log.Printf("icap: Accept error: %v", err)
			// If this is a temporary error, retry
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// Retry after a small delay
				time.Sleep(5 * time.Millisecond)
				continue
			}
			return err
		}
		if srv.ReadTimeout != 0 {
			if err := rw.SetReadDeadline(time.Now().Add(srv.ReadTimeout)); err != nil {
				log.Printf("icap: SetReadDeadline error: %v", err)
			}
		}
		if srv.WriteTimeout != 0 {
			if err := rw.SetWriteDeadline(time.Now().Add(srv.WriteTimeout)); err != nil {
				log.Printf("icap: SetWriteDeadline error: %v", err)
			}
		}
		c, err := newConn(rw, handler)
		if err != nil {
			continue
		}
		go c.serve(srv.DebugLevel)
	}
}

// Serve accepts incoming ICAP connections on the listener l,
// creating a new service thread for each.  The service threads
// read requests and then call handler to reply to them.
func Serve(l net.Listener, handler Handler) error {
	srv := &Server{Handler: handler}
	return srv.Serve(l)
}

// ListenAndServe listens on the TCP network address addr
// and then calls Serve with handler to handle requests
// on incoming connections.
func ListenAndServe(addr string, handler Handler) error {
	server := &Server{Addr: addr, Handler: handler}
	return server.ListenAndServe()
}

func ListenAndServeDebug(addr string, handler Handler) error {
	server := &Server{Addr: addr, Handler: handler}
	server.DebugLevel = 1
	return server.ListenAndServe()
}

// ListenAndServeTLS --
func ListenAndServeTLS(addr, cert, key string, handler Handler) error {
	server := &Server{Addr: addr, Handler: handler}
	return server.ListenAndServeTLS(cert, key)
}
