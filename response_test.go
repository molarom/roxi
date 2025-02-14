// Copyright 2025 Brandon Epperson
// SPDX-License-Identifier: Apache-2.0

package roxi

import (
	"context"
	"net/http"
	"testing"
)

// ----------------------------------------------------------------------
// Test Data

// mockResponseWriter is a writer used in tests.
type mockResponseWriter struct {
	body       []byte
	statusCode int
	header     http.Header
}

func (w *mockResponseWriter) Header() http.Header {
	return w.header
}

func (w *mockResponseWriter) Write(b []byte) (int, error) {
	w.body = b
	return len(w.body), nil
}

func (w *mockResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

// ----------------------------------------------------------------------
// Tests

func Test_Respond(t *testing.T) {
	// setup writer
	mockWriter := &mockResponseWriter{
		header: http.Header{},
	}

	// setup context
	ctx := context.Background()
	ctx = setWriter(ctx, mockWriter)

	tests := []struct {
		name       string
		data       Responder
		statusCode int
		body       []byte
		shouldErr  bool
	}{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		})
	}
}
