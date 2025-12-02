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

// Default error response handlers.
var (
	// NotFound is a default 404 handler.
	NotFound = func(ctx context.Context, r *http.Request) error {
		return respond(ctx, &errorResponse{
			http.StatusNotFound,
			http.StatusText(http.StatusNotFound),
		})
	}

	// MethodNotAllowed is a default 405 handler.
	MethodNotAllowed = func(ctx context.Context, r *http.Request) error {
		return respond(ctx, &errorResponse{
			http.StatusMethodNotAllowed,
			http.StatusText(http.StatusMethodNotAllowed),
		})
	}

	// MethodNotAllowed is a default 500 handler.
	InternalServerError = func(ctx context.Context, r *http.Request) error {
		return respond(ctx, &errorResponse{
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

func respond(ctx context.Context, data *errorResponse) error {
	w := GetWriter(ctx)

	if data == nil {
		return errors.New("respond: data is nil")
	}

	v, ct, err := data.Response()
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", ct)
	w.WriteHeader(data.StatusCode())

	if _, err := w.Write(v); err != nil {
		return err
	}

	return nil
}

// ----------------------------------------------------------------------
// helper types

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
