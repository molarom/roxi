// Copyright 2025 Brandon Epperson
// SPDX-License-Identifier: Apache-2.0

// Package roxi is a lightweight http multiplexer.
//
// This package borrows inspiration from Julien Schmidt's httprouter and Daniel Imfeld's
// httptreemux. It still makes use of a PATRICA tree, but the tree implementation differs from both.
//
// The aim was to have a mux that mets the following requirements:
//
// 1. A path segment may be variable in one route and a static token in another.
// 2. Path values can be retrieved with r.PathValue(<var>)
// 3. HandlerFunc's accept a context.Context parameter and can return errors.
// 4. Provide a simple mux-wide configuration.
// 5. Be as performant and memory efficent as possible.
// 6. Integrate well with net/http as well as any extension packages.
//
// There are some additional methods included in this package that may optionally be used
// to improve developer experience, such as Decode and Respond for handling request
// and response data respectively. These components were inspired by Bill Kennedy's Service project.
package roxi

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	stdpath "path"
	"regexp"
	"strings"
	"sync"
)

// HandlerFunc represents a function to handle HTTP requests.
//
// The http.ResponseWriter can be retrieved from the context with:
//
//	GetWriter(ctx)
type HandlerFunc func(ctx context.Context, r *http.Request) error

// ServeHTTP implements the http.Handler interface.
//
// If a HandlerFunc is invoked with ServeHTTP and returns an error,
// http.Error is called to return the error in the response text and set the
// response code to 500.
//
// If this behavior is undesired, the error must be handled and set to nil
// prior to it's return.
func (f HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Setup context.
	ctx := r.Context()
	ctx = &writerContext{ctx, writerKey, w}

	if err := f(ctx, r); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Logger represents a function used by the Mux to log internal events.
type Logger func(msg string, args ...any)

// Mux represents an http.Handler for registering HandlerFuncs to handle
// HTTP requests.
type Mux struct {
	log   Logger
	trees map[string]*node
	mw    []MiddlewareFunc

	// Routing
	setAllowHeader       bool
	routeCaseInsensitive bool

	// Redirects
	redirectTrailingSlash bool
	redirectCleanPath     bool

	// Error handlers
	methodNotAllowed http.Handler
	notFound         http.Handler
	errHandler       http.Handler

	// Panics
	panicHandler PanicHandler

	// pool for writerContext
	ctxPool sync.Pool
}

// New returns a new initalized Mux.
//
// No options are configured other than the default error handlers.
// This includes the omission of the default PanicHandler.
func New(opts ...func(*Mux)) *Mux {
	m := &Mux{
		log:              log.Printf,
		trees:            make(map[string]*node),
		methodNotAllowed: HandlerFunc(MethodNotAllowed),
		notFound:         HandlerFunc(NotFound),
		errHandler:       HandlerFunc(InternalServerError),
		ctxPool: sync.Pool{
			New: func() any {
				return new(writerContext)
			},
		},
	}

	for _, o := range opts {
		o(m)
	}
	return m
}

// NewWithDefaults is a helper method to return a mux with default options enabled.
//
// It is equivalent to calling:
//
// New(
// 		WithSetAllowHeader(),
// 		WithRedirectCleanPath(),
// 		WithRedirectTrailingSlash(),
// 		WithPanicHandler(DefaultPanicHandler),
//      opts...,
// )

func NewWithDefaults(opts ...func(*Mux)) *Mux {
	return New(
		append([]func(*Mux){
			WithSetAllowHeader(),
			WithRedirectCleanPath(),
			WithRedirectTrailingSlash(),
			WithPanicHandler(DefaultPanicHandler),
		}, opts...)...,
	)
}

// ----------------------------------------------------------------------
// Mux options

// WithLogger replaces the mux's internal logger. By default, it calls log.Printf.
func WithLogger(log Logger) func(*Mux) {
	return func(m *Mux) {
		m.log = log
	}
}

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
//	m.Handler("OPTIONS", "/*path", handler)
func WithOptionsHandler(handler http.Handler) func(*Mux) {
	return func(m *Mux) {
		m.Handler("OPTIONS", "/*path", handler)
	}
}

// WithSetAllowHeader enables the mux to set the Allow header on 405 responses.
func WithSetAllowHeader() func(*Mux) {
	return func(m *Mux) {
		m.setAllowHeader = true
	}
}

// WithRedirectCaseInsensitive enables case insensitive routing.
//
// Example:
//
// mux.GET("/FOO", handlerFunc) Matches: ("/FOO", "/foo", "/fOO", etc.)
func WithCaseInsensitiveRouting() func(*Mux) {
	return func(m *Mux) {
		m.routeCaseInsensitive = true
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

// WithMethodNotAllowedHandler replaces the default 405 response handler.
func WithMethodNotAllowedHandler(handler http.Handler) func(*Mux) {
	return func(m *Mux) {
		m.methodNotAllowed = handler
	}
}

// WithNotFoundResponse replaces the default 404 response handler.
func WithNotFoundHandler(handler http.Handler) func(*Mux) {
	return func(m *Mux) {
		m.notFound = handler
	}
}

// WithErrorResponse replaces the default 500 response handler.
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
	ctx := m.ctxPool.Get().(*writerContext)
	ctx.Context = r.Context()
	ctx.key = writerKey
	ctx.value = w
	defer m.ctxPool.Put(ctx)

	if m.panicHandler != nil {
		defer func() {
			if rec := recover(); rec != nil {
				m.log("recovered", "panic", rec)
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

	// search for handler
	if handler, found := root.search(path, r); found {
		if err := handler(ctx, r); err != nil {
			m.log("executing handler", "error", err)
			m.errHandler.ServeHTTP(w, r)
		}
		return
	}

	// don't redirect if proxy connection or root path are requested.
	if r.Method != http.MethodConnect && !bytes.Equal(path, []byte{'/'}) {
		// following the same redirect behavior as httprouter
		code := http.StatusMovedPermanently
		if r.Method != http.MethodGet {
			code = http.StatusPermanentRedirect
		}

		// check if any redirect behavior is enabled.
		redirect := (m.redirectCleanPath || m.redirectTrailingSlash || m.routeCaseInsensitive)

		// step through each enabled path scrubbing option
		if m.routeCaseInsensitive {
			path = toBytes(strings.ToLower(r.URL.Path))
		}

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
				m.log("redirecting to", "path", r.URL.Path)
				_ = Redirect(ctx, r, r.URL.String(), code)
				return
			}
		}
	}

	// not found case.
	m.log("no matching route registered for", "path", r.URL.Path)
	m.notFound.ServeHTTP(w, r)
}

// Handler registers an http.Handler to handle requests at the given
// method and path.
func (m *Mux) Handler(method, path string, handler http.Handler, mw ...MiddlewareFunc) {
	m.Handle(method, path, func(ctx context.Context, r *http.Request) error {
		handler.ServeHTTP(GetWriter(ctx), r)
		return nil
	}, mw...)
}

// HandlerFunc registers an http.HandlerFunc to handle requests at the given
// method and path.
func (m *Mux) HandlerFunc(method, path string, handler http.HandlerFunc, mw ...MiddlewareFunc) {
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

	if len(path) == 0 {
		panic("cannot register empty path")
	}

	if len(path) > 0 && path[0] != '/' {
		panic("path '" + path + "' does not begin with '/'")
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

	if m.routeCaseInsensitive {
		root.insert([]byte(strings.ToLower(path)), handlerFunc)
	} else {
		root.insert([]byte(path), handlerFunc)
	}
}

// ----------------------------------------------------------------------
// File Server methods

// FileServer wraps http.FileServer to allow the mux's error handlers to be
// called over the internal http.FileServer ones.
//
// The path must end in a wildcard with the name '*file'.
func (m *Mux) FileServer(path string, fs http.FileSystem) {
	// check path
	checkFSPath(path)

	fsrv := http.FileServer(fs)
	m.GET(path, func(ctx context.Context, r *http.Request) error {
		f := stdpath.Clean(r.PathValue("file"))
		if _, err := fs.Open(f); err != nil {
			if !os.IsNotExist(err) {
				return err
			}
			m.notFound.ServeHTTP(GetWriter(ctx), r)
		}

		r.URL.Path = f
		fsrv.ServeHTTP(GetWriter(ctx), r)
		return nil
	})
}

// FileServerRE serves files from the specified http.Dir but restricts
// file lookups to require matching the specified regular expression.
//
// The path must end in a wildcard with the name '*file'.
func (m *Mux) FileServerRE(path, regex string, fs http.FileSystem) {
	// check path
	checkFSPath(path)

	re := regexp.MustCompile(regex)

	fsrv := http.FileServer(fs)
	m.GET(path, func(ctx context.Context, r *http.Request) error {
		f := stdpath.Clean(r.PathValue("file"))
		if !re.MatchString(f) {
			m.log("regexp did not match: %s", f)
			m.notFound.ServeHTTP(GetWriter(ctx), r)
			return nil
		}

		if _, err := fs.Open(f); err != nil {
			if !os.IsNotExist(err) {
				return err
			}
			m.notFound.ServeHTTP(GetWriter(ctx), r)
			return nil
		}

		r.URL.Path = f
		fsrv.ServeHTTP(GetWriter(ctx), r)
		return nil
	})
}

func checkFSPath(path string) {
	if len(path) == 0 {
		panic("cannot register empty path")
	}

	if len(path) > 0 && path[0] != '/' {
		panic("path '" + path + "' does not begin with '/'")
	}

	if path[len(path)-6:] != "/*file" {
		panic("file server path must end in '/*file'")
	}
}

// ----------------------------------------------------------------------
// Helper methods

// GET is a helper method for m.Handle("GET", path, handlerFunc, mw...)
func (m *Mux) GET(path string, handlerFunc HandlerFunc, mw ...MiddlewareFunc) {
	m.Handle("GET", path, handlerFunc, mw...)
}

// HEAD is a helper method for m.Handle("HEAD", path, handlerFunc, mw...)
func (m *Mux) HEAD(path string, handlerFunc HandlerFunc, mw ...MiddlewareFunc) {
	m.Handle("HEAD", path, handlerFunc, mw...)
}

// POST is a helper method for m.Handle("POST", path, handlerFunc, mw...)
func (m *Mux) POST(path string, handlerFunc HandlerFunc, mw ...MiddlewareFunc) {
	m.Handle("POST", path, handlerFunc, mw...)
}

// PUT is a helper method for m.Handle("PUT", path, handlerFunc, mw...)
func (m *Mux) PUT(path string, handlerFunc HandlerFunc, mw ...MiddlewareFunc) {
	m.Handle("PUT", path, handlerFunc, mw...)
}

// PATCH is a helper method for m.Handle("PATCH", path, handlerFunc, mw...)
func (m *Mux) PATCH(path string, handlerFunc HandlerFunc, mw ...MiddlewareFunc) {
	m.Handle("PATCH", path, handlerFunc, mw...)
}

// DELETE is a helper method for m.Handle("DELETE", path, handlerFunc, mw...)
func (m *Mux) DELETE(path string, handlerFunc HandlerFunc, mw ...MiddlewareFunc) {
	m.Handle("DELETE", path, handlerFunc, mw...)
}

// OPTIONS is a helper method for m.Handle("OPTIONS", path, handlerFunc, mw...)
func (m *Mux) OPTIONS(path string, handlerFunc HandlerFunc, mw ...MiddlewareFunc) {
	m.Handle("OPTIONS", path, handlerFunc, mw...)
}

// PrintTree prints the contents of the routing tree.
//
// The root node is always skipped when performing lookups,
// so seeing:
//
//	  []: nil
//		   ["/"]: <...>
//
// is expected behavior when printing the Tree.
func (m *Mux) PrintTree() {
	for k, v := range m.trees {
		fmt.Printf("[%s]\n", k)
		v.print(1)
	}
}
