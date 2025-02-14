// Copyright 2025 Brandon Epperson
// SPDX-License-Identifier: Apache-2.0

// Package roxi represents an http.Handler and associated framework.
//
// The mux was inspired by Julien Schmidt's httprouter and Daniel Imfeld's
// httptreemux. The aim was to as close as possible to performance of httprouter,
// but maintain much of the flexibility of routing patterns in httptreemux.
//
// The routing rules are the same as httptreemux, with the exception of
// supporting routes that escape the param and wildcard characters.
//
// TODO: add routing examples.
//
// This mux also makes use of the routing wildcard feature introduced in
// go 1.22 over a custom Param implementation, allowing param and wildcard
// variables to be retrieved with `r.PathValue('<var_name>')`.
//
// The framework components were inspired by Bill Kennedy's Service framework.
package roxi

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
)

// HandlerFunc represents a function to handle HTTP requests.
//
// The http.ResponseWriter can be retrieved from the context with:
//
//	GetWriter(ctx)
type HandlerFunc func(ctx context.Context, r *http.Request) error

// ServeHTTP implements the http.Handler interface.
func (f HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Setup context.
	ctx := r.Context()
	ctx = setWriter(ctx, w)
	f(ctx, r)
}

// Logger represents a function used by the Mux to log internal events.
type Logger func(msg string, args ...any)

// Mux represents an http.Handler for registering HandlerFuncs to handle
// HTTP requests.
type Mux struct {
	log            Logger
	trees          map[string]*node
	mw             []MiddlewareFunc
	panicHandler   PanicHandler
	setAllowHeader bool

	// Redirects
	redirectTrailingSlash bool
	redirectCleanPath     bool

	// Error handlers
	methodNotAllowed http.Handler
	notFound         http.Handler
	errHandler       http.Handler
}

// New returns a new Mux.
func New(log Logger, opts ...func(*Mux)) *Mux {
	m := &Mux{
		log:              log,
		trees:            make(map[string]*node),
		methodNotAllowed: HandlerFunc(MethodNotAllowed),
		notFound:         HandlerFunc(NotFound),
		errHandler:       HandlerFunc(InternalServerError),
	}

	for _, o := range opts {
		o(m)
	}
	return m
}

// ----------------------------------------------------------------------
// Mux options

// WithMiddleware registers global middleware for the mux to execute.
func WithMiddleware(mw ...MiddlewareFunc) func(*Mux) {
	return func(m *Mux) {
		m.mw = mw
	}
}

// WithPanicHandler registers a PanicHandler to recover from panics.
//
// When adding a panic handler, the mux will also log the error message
// from the panic.
func WithPanicHandler(handler PanicHandler) func(*Mux) {
	return func(m *Mux) {
		m.panicHandler = handler
	}
}

// WithOptionsHandler is a helper method to add a default OPTIONS handler to the Mux.
//
// It is equivalent to calling:
//
//	m.Handler("OPTIONS", "/", handler)
func WithOptionsHandler(handler http.Handler) func(*Mux) {
	return func(m *Mux) {
		m.Handler("OPTIONS", "/", handler)
	}
}

// WithSetAllowHeader enables the mux to set the Allow header on 405 responses.
func WithSetAllowHeader() func(*Mux) {
	return func(m *Mux) {
		m.setAllowHeader = true
	}
}

// WithRedirectTrailingSlash enables redirection of unmatched request paths
// that contain a trailing '/' character.
func WithRedirectTrailingSlash() func(*Mux) {
	return func(m *Mux) {
		m.redirectTrailingSlash = true
	}
}

// WithRedirectCleanPath enables the cleaning of the request path for
// redirection of unmatched request paths.
func WithRedirectCleanPath() func(*Mux) {
	return func(m *Mux) {
		m.redirectCleanPath = true
	}
}

// WithNotAllowedResponse sets a Handler to be executed for 405 responses.
func WithMethodNotAllowedHandler(handler http.Handler) func(*Mux) {
	return func(m *Mux) {
		m.methodNotAllowed = handler
	}
}

// WithNotFoundResponse sets a Handler to executed for 404 responses.
func WithNotFoundHandler(handler http.Handler) func(*Mux) {
	return func(m *Mux) {
		m.notFound = handler
	}
}

// WithErrorResponse sets a Handler to be executed for 500 responses.
func WithErrorHandler(handler http.Handler) func(*Mux) {
	return func(m *Mux) {
		m.errHandler = handler
	}
}

// ----------------------------------------------------------------------
// Methods

// ServeHTTP implements the http.Handler interface.
func (m *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Setup context.
	ctx := r.Context()
	ctx = setWriter(ctx, w)

	if m.panicHandler != nil {
		defer func() {
			if rec := recover(); rec != nil {
				m.log("recovered panic", "panic", rec)
				m.panicHandler(ctx, r, rec)
			}
		}()
	}

	path := toBytes(r.URL.Path)

	root, found := m.trees[r.Method]
	if !found {
		if m.setAllowHeader {
			allowed := make([]string, 0, 9)
			for method, t := range m.trees {
				if _, ok := t.search(path, r); !ok {
					continue
				}
				allowed = append(allowed, method)
			}

			allow := strings.Join(allowed, ",")
			if allow != "" {
				found = true
				w.Header().Set("Allow", allow)
			}
		} else {
			for _, t := range m.trees {
				if _, ok := t.search(path, r); !ok {
					continue
				}

				// break iteration early if we're not using the match
				found = true
				break
			}
		}

		if found {
			m.methodNotAllowed.ServeHTTP(w, r)
			return
		}
	}

	if handler, found := root.search(path, r); found {
		if err := handler(ctx, r); err != nil {
			m.log("error executing handler", "error", err)
			m.errHandler.ServeHTTP(w, r)
		}
		return
	}

	if r.Method != http.MethodConnect && !bytes.Equal(path, []byte{'/'}) {
		// following the same redirect behavior as httprouter
		code := http.StatusMovedPermanently
		if r.Method != http.MethodGet {
			code = http.StatusPermanentRedirect
		}

		// check if any redirect behavior is enabled.
		redirect := (m.redirectCleanPath || m.redirectTrailingSlash)

		// step through each enabled path scrubbing option
		if m.redirectCleanPath {
			path = CleanPath(r.URL.Path)
		}

		if m.redirectTrailingSlash {
			if len(path) > 1 && path[len(path)-1] == '/' {
				path = path[:len(path)-1]
			}
		}

		if redirect {
			// found a match, redirect to correct path.
			if _, found := root.search(path, r); found {
				r.URL.Path = toString(path)
				Redirect(ctx, r, r.URL.String(), code)
				return
			}
		}
	}

	m.methodNotAllowed.ServeHTTP(w, r)
}

// Handler registers an http.Handler to handle requests at the given
// method and path.
func (m *Mux) Handler(method, path string, handler http.Handler, mw ...MiddlewareFunc) {
	m.Handle(method, path, func(ctx context.Context, r *http.Request) error {
		handler.ServeHTTP(GetWriter(ctx), r)
		return nil
	}, mw...)
}

// Handle registers a HandlerFunc to handle requests at the given
// method and path.
//
// Handle only allows standard HTTP methods provided by net/http.
func (m *Mux) Handle(method, path string, handlerFunc HandlerFunc, mw ...MiddlewareFunc) {
	if method == "" {
		panic("method cannot be empty")
	}

	if _, ok := httpMethods[method]; !ok {
		panic("method '" + method + "' is not a valid http method")
	}

	if handlerFunc == nil {
		panic("handlerfunc cannot be nil")
	}

	root := m.trees[method]
	if root == nil {
		root = &node{}

		m.trees[method] = root
	}

	handlerFunc = MiddlewareStack(handlerFunc, mw...)
	handlerFunc = MiddlewareStack(handlerFunc, m.mw...)
	root.insert([]byte(path), handlerFunc)
}

// PrintRoutes prints all of the registered routes on the Mux.
func (m *Mux) PrintRoutes() {
	for k, v := range m.trees {
		fmt.Printf("[%s]\n", k)
		v.print(1)
	}
}
