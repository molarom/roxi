// Copyright 2025 Brandon Epperson
// SPDX-License-Identifier: Apache-2.0

package roxi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime"
)

// Responder represents a web response.
type Responder interface {
	Response() (data []byte, contentType string, err error)
}

// StatusSetter is a responder that supports setting
// status codes other than the default 200.
type StatusSetter interface {
	Responder
	StatusCode() int
}

// NoContent is a helper responder for 204 responses.
var NoContent = emptyResponse{http.StatusNoContent}

// Default error response handlers.
var (
	// NotFound is a default 404 handler.
	NotFound = func(ctx context.Context, r *http.Request) error {
		return Respond(ctx, errorResponse{
			http.StatusNotFound,
			http.StatusText(http.StatusNotFound),
		})
	}

	// MethodNotAllowed is a default 405 handler.
	MethodNotAllowed = func(ctx context.Context, r *http.Request) error {
		return Respond(ctx, errorResponse{
			http.StatusMethodNotAllowed,
			http.StatusText(http.StatusMethodNotAllowed),
		})
	}

	// MethodNotAllowed is a default 500 handler.
	InternalServerError = func(ctx context.Context, r *http.Request) error {
		return Respond(ctx, errorResponse{
			http.StatusInternalServerError,
			http.StatusText(http.StatusInternalServerError),
		})
	}

	// DefaultPanicHandler is a default handler that executes when a panic is recovered.
	DefaultPanicHandler = func(ctx context.Context, r *http.Request, err any) {
		buf := make([]byte, 65536)
		buf = buf[:runtime.Stack(buf, false)]
		fmt.Printf("roxi: recovered panic %v: %s\n", err, buf)
		GetWriter(ctx).WriteHeader(http.StatusInternalServerError)
	}
)

// Respond sets the appropriate http headers and writes a response to
// the http.ResponseWriter in the context.
//
// It should be called as the return of a HandlerFunc.
//
// Example:
//
//	func Handler(ctx context.Context, r *http.Request) error {
//	    return Respond(ctx, NoContent)
//	}
func Respond(ctx context.Context, data Responder) error {
	w := GetWriter(ctx)

	if data == nil {
		return errors.New("respond: data is nil")
	}

	switch v := data.(type) {
	case emptyResponse:
		w.WriteHeader(v.StatusCode())
		return nil
	}

	v, ct, err := data.Response()
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", ct)
	if s, ok := data.(StatusSetter); ok {
		w.WriteHeader(s.StatusCode())
	}

	if _, err := w.Write(v); err != nil {
		return err
	}

	return nil
}

// Redirect is a helper method that wraps http.Redirect to send a Redirect response.
//
// The error returned will always be nil, as this is intended to be used as the return
// of a HandlerFunc.
//
// Example:
//
//	func Handler(ctx context.Context, r *http.Request) error {
//	    return Redirect(ctx, r, "/redirect", 301)
//	}
func Redirect(ctx context.Context, r *http.Request, url string, code int) error {
	http.Redirect(GetWriter(ctx), r, url, code)
	return nil
}

// ----------------------------------------------------------------------
// helper types

type emptyResponse struct {
	code int
}

func (r emptyResponse) Response() ([]byte, string, error) {
	return nil, "", nil
}

func (r emptyResponse) StatusCode() int {
	return r.code
}

type errorResponse struct {
	code    int
	message string
}

func (r errorResponse) Response() ([]byte, string, error) {
	return toBytes(r.message), "text/plain", nil
}

func (r errorResponse) StatusCode() int {
	return r.code
}
