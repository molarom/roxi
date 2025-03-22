// Copyright 2025 Brandon Epperson
// SPDX-License-Identifier: Apache-2.0

// Package roxi is a lightweight http multiplexer.
//
// This package borrows inspiration from Julien Schmidt's httprouter and Daniel Imfeld's
// httptreemuxin that it uses the same format for path variables and makes
// use of a PATRICA tree, but the tree and variable handling implementation differs from both.
package roxi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	stdpath "path"
	"regexp"
	"strings"
	"sync"
)

// pool for writerContext.
var ctxPool = sync.Pool{
	New: func() any {
		return new(writerContext)
	},
}

// HandlerFunc represents a function to handle HTTP requests.
//
// The http.ResponseWriter can be retrieved from the context with:
//
//	roxi.GetWriter(ctx)
type HandlerFunc func(ctx context.Context, r *http.Request) error

// ServeHTTP implements the http.Handler interface.
//
// If a HandlerFunc is invoked with ServeHTTP and returns an error,
// http.Error will be called in the manner below:
//
//	http.Error(w, err.Error(), http.StatusInternalServerError)
//
// If this behavior is undesired, the error must be handled and set to nil
// prior to the function's return.
func (f HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Setup context.
	ctx := ctxPool.Get().(*writerContext)
	ctx.Context = r.Context()
	ctx.value = w
	defer ctxPool.Put(ctx)

	if err := f(ctx, r); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Mux represents an http.Handler for registering HandlerFuncs to handle
// HTTP requests.
type Mux struct {
	trees map[string]*node
	mw    []MiddlewareFunc

	// Routing
	routeCaseInsensitive bool

	// Redirects
	redirectTrailingSlash bool
	redirectCleanPath     bool

	// OPTIONS hander
	optionsHandler http.Handler

	// Error handlers
	methodNotAllowed http.Handler
	notFound         http.Handler
	errHandler       http.Handler

	// Panics
	panicHandler PanicHandler
}

// New returns a new initialized Mux.
//
// No options are configured other than the default error handlers and panic handler.
func New(opts ...func(*Mux)) *Mux {
	m := &Mux{
		trees:            make(map[string]*node),
		methodNotAllowed: HandlerFunc(MethodNotAllowed),
		notFound:         HandlerFunc(NotFound),
		errHandler:       HandlerFunc(InternalServerError),
		panicHandler:     DefaultPanicHandler,
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
//	New(append([]func(*Mux){
//			WithSetAllowHeader(),
//			WithRedirectCleanPath(),
//			WithRedirectTrailingSlash(),
//		}, opts...)...)
func NewWithDefaults(opts ...func(*Mux)) *Mux {
	return New(
		append([]func(*Mux){
			WithRedirectCleanPath(),
			WithRedirectTrailingSlash(),
		}, opts...)...,
	)
}

// ----------------------------------------------------------------------
// Mux options

// WithMiddleware registers global middleware for the mux to execute.
//
// Middleware registered directly with the mux will execute prior to
// any middleware registered with HandlerFuncs.
func WithMiddleware(mw ...MiddlewareFunc) func(*Mux) {
	return func(m *Mux) {
		m.mw = mw
	}
}

// WithPanicHandler enables panic recovery in the mux and registers a PanicHandler
// that executes if a panic occurs during the lifecycle of the mux.
//
// When adding a panic handler, the mux will also log the error message
// from the panic.
//
// To disable the panic handler, provide a nil value to the handler parameter.
func WithPanicHandler(handler PanicHandler) func(*Mux) {
	return func(m *Mux) {
		m.panicHandler = handler
	}
}

// WithOptionsHandler sets a handler for the mux to handle OPTIONS requests.
func WithOptionsHandler(handler http.Handler) func(*Mux) {
	return func(m *Mux) {
		m.optionsHandler = handler
	}
}

// WithRedirectCaseInsensitive enables case insensitive routing.
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

// WithNotFoundHandler replaces the default 404 response handler.
func WithNotFoundHandler(handler http.Handler) func(*Mux) {
	return func(m *Mux) {
		m.notFound = handler
	}
}

// WithErrorHandler replaces the default 500 response handler.
//
// A nil handler will be ignored.
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
	ctx := ctxPool.Get().(*writerContext)
	ctx.Context = r.Context()
	ctx.value = w
	defer ctxPool.Put(ctx)

	if m.panicHandler != nil {
		defer func() {
			if rec := recover(); rec != nil {
				m.panicHandler(ctx, r, rec)
			}
		}()
	}

	path := toBytes(r.URL.Path)

	if root := m.trees[r.Method]; root != nil {
		// search for handler
		if handler, found := root.search(path, r); found {
			if err := handler(ctx, r); err != nil {
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
			if m.redirectCleanPath {
				path = CleanPath(r.URL.Path)
			}

			if m.redirectTrailingSlash {
				if len(path) > 1 && path[len(path)-1] == '/' {
					path = path[:len(path)-1]
				}
			}

			if m.routeCaseInsensitive {
				path = bytes.ToLower(path)
			}

			if redirect {
				// found a match, redirect to correct path.
				if _, found := root.search(path, r); found {
					r.URL.Path = toString(path)
					_ = Redirect(ctx, r, r.URL.String(), code)
					return
				}
			}
		}
	}

	if m.optionsHandler != nil {
		if allow := m.allowed(path, r); allow != "" {
			w.Header().Set("Allow", allow)
			m.optionsHandler.ServeHTTP(w, r)
			return
		}
	}

	if m.methodNotAllowed != nil {
		if allow := m.allowed(path, r); allow != "" {
			w.Header().Set("Allow", allow)
			m.methodNotAllowed.ServeHTTP(w, r)
			return
		}
	}

	// not found case.
	m.notFound.ServeHTTP(w, r)
}

func (m *Mux) allowed(path []byte, r *http.Request) string {
	allowed := make([]string, 0, 9)
	if m.optionsHandler != nil {
		allowed = append(allowed, http.MethodOptions)
	}
	for method, t := range m.trees {
		if method == http.MethodOptions || method == r.Method {
			continue
		}
		if _, ok := t.search(path, r); ok {
			allowed = append(allowed, method)
		}
	}
	if len(allowed) != 0 {
		return strings.Join(allowed, ",")
	}
	return ""
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
		root.insert(toBytes(strings.ToLower(path)), handlerFunc)
	} else {
		root.insert(toBytes(path), handlerFunc)
	}
}

// ----------------------------------------------------------------------
// File Server methods

// FileServer wraps http.FileServer to allow the mux's error handlers to be
// called over the internal http.FileServer ones.
//
// The path must end in a wildcard with the name '*file'.
func (m *Mux) FileServer(path string, fs http.FileSystem, mw ...MiddlewareFunc) {
	// check path
	if err := checkFSPath(path); err != nil {
		panic(err)
	}

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
	}, mw...)
}

// FileServerRE serves files from the specified http.Dir but restricts
// file lookups to require matching the specified regular expression.
//
// The path must end in a wildcard with the name '*file'.
func (m *Mux) FileServerRE(path, regex string, fs http.FileSystem, mw ...MiddlewareFunc) {
	// check path
	if err := checkFSPath(path); err != nil {
		panic(err)
	}

	re := regexp.MustCompile(regex)

	fsrv := http.FileServer(fs)
	m.GET(path, func(ctx context.Context, r *http.Request) error {
		f := stdpath.Clean(r.PathValue("file"))
		if !re.MatchString(f) {
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
	}, mw...)
}

func checkFSPath(path string) error {
	if len(path) == 0 {
		return errors.New("cannot register empty path")
	}

	if len(path) > 0 && path[0] != '/' {
		return errors.New("path '" + path + "' does not begin with '/'")
	}

	if len(path) < 6 || path[len(path)-6:] != "/*file" {
		return errors.New("file server path must end in '/*file'")
	}

	return nil
}

// ----------------------------------------------------------------------
// Helper methods

// GET is a helper method for m.Handle("GET", path, handlerFunc, mw...)
func (m *Mux) GET(path string, handlerFunc HandlerFunc, mw ...MiddlewareFunc) {
	m.Handle(http.MethodGet, path, handlerFunc, mw...)
}

// HEAD is a helper method for m.Handle("HEAD", path, handlerFunc, mw...)
func (m *Mux) HEAD(path string, handlerFunc HandlerFunc, mw ...MiddlewareFunc) {
	m.Handle(http.MethodHead, path, handlerFunc, mw...)
}

// POST is a helper method for m.Handle("POST", path, handlerFunc, mw...)
func (m *Mux) POST(path string, handlerFunc HandlerFunc, mw ...MiddlewareFunc) {
	m.Handle(http.MethodPost, path, handlerFunc, mw...)
}

// PUT is a helper method for m.Handle("PUT", path, handlerFunc, mw...)
func (m *Mux) PUT(path string, handlerFunc HandlerFunc, mw ...MiddlewareFunc) {
	m.Handle(http.MethodPut, path, handlerFunc, mw...)
}

// PATCH is a helper method for m.Handle("PATCH", path, handlerFunc, mw...)
func (m *Mux) PATCH(path string, handlerFunc HandlerFunc, mw ...MiddlewareFunc) {
	m.Handle(http.MethodPatch, path, handlerFunc, mw...)
}

// DELETE is a helper method for m.Handle("DELETE", path, handlerFunc, mw...)
func (m *Mux) DELETE(path string, handlerFunc HandlerFunc, mw ...MiddlewareFunc) {
	m.Handle(http.MethodDelete, path, handlerFunc, mw...)
}

// OPTIONS is a helper method for m.Handle("OPTIONS", path, handlerFunc, mw...)
func (m *Mux) OPTIONS(path string, handlerFunc HandlerFunc, mw ...MiddlewareFunc) {
	m.Handle(http.MethodOptions, path, handlerFunc, mw...)
}

// ----------------------------------------------------------------------
// Debugging methods

// PrintRoutes prints all of the routes registered in the Mux.
// Ordering of Methods is not guaranteed.
//
// Routes are printed in the format:
//
//	<Method> <path>
func (m *Mux) PrintRoutes() {
	for k, v := range m.trees {
		v.printLeaves(toBytes(k + " "))
	}
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
