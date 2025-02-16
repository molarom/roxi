// Package roxi is a lightweight http multiplexer.
//
// This package borrows inspiration from Julien Schmidt's httprouter and Daniel Imfeld's
// httptreemux. It still makes use of a PATRICA tree, but the implementation differs from both.
//
// The aim was to have a mux that mets the following requirements:
//
// 1. A path segment may be variable in one route and a static token in another.
// 2. Path values can be retrieved with r.PathValue(<var>)
// 3. HandlerFunc's accept a context.Context parameter and can return errors.
// 4. Keep mux configuration simple.
// 5. Be as performant and memory efficent as possible.
//
// There are some additional methods included in this package that may optionally be used
// to improve developer experience, such as Decode and Respond for handling request
// and response data respectively. These components were inspired by Bill Kennedy's Service project.
//
// Minimal Example:
//
// package main
//
// import (
//
//	"context"
//	"log"
//	"net/http"
//
//	"gitlab.com/romalor/roxi"
//
// )
//
// type HomePage []byte
//
//	func (r HomePage) Encode() ([]byte, string, error) {
//		return r, "text/plain; charset=utf-8", nil
//	}
//
//	func Root(ctx context.Context, r *http.Request) error {
//		return roxi.Redirect(ctx, r, "/home", 301)
//	}
//
//	func Home(ctx context.Context, r *http.Request) error {
//		return roxi.Respond(ctx, HomePage("Welcome!"))
//	}
//
//	func main() {
//		mux := roxi.New()
//
//		mux.GET("/", Root)
//		mux.GET("/home", Home)
//
//		log.Fatal(http.ListenAndServe(":8080", mux))
//	}
package roxi

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
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
func (f HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Setup context.
	ctx := r.Context()
	ctx = &writerContext{ctx, writerKey, w}

	// TODO: evaluate what best behavior would be here.
	_ = f(ctx, r)
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

	// pool for writerContext
	ctxPool sync.Pool
}

// New returns a new initalized Mux. No options are configured other than the default error handlers.
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
//	 New(
//			WithSetAllowHeader(),
//			WithRedirectCleanPath(),
//			WithRedirectTrailingSlash(),
//			WithPanicHandler(DefaultPanicHandler),
//			WithOptionsHandler(HandlerFunc(DefaultCORS)),
//		)
func NewWithDefaults() *Mux {
	return New(
		WithSetAllowHeader(),
		WithRedirectCleanPath(),
		WithRedirectTrailingSlash(),
		WithPanicHandler(DefaultPanicHandler),
		WithOptionsHandler(HandlerFunc(DefaultCORS)),
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
				m.log("recovered panic: %v", rec)
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
			m.log("executing handler: %s", err)
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
				_ = Redirect(ctx, r, r.URL.String(), code)
				return
			}
		}
	}

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
	root.insert([]byte(path), handlerFunc)
}

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
func (m *Mux) PrintRoutes() {
	for k, v := range m.trees {
		fmt.Printf("[%s]\n", k)
		v.print(1)
	}
}
