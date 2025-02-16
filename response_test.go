// Copyright 2025 Brandon Epperson
// SPDX-License-Identifier: Apache-2.0

package roxi

import (
	"bytes"
	"context"
	"fmt"
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

func newMockResponseWriter() *mockResponseWriter {
	return &mockResponseWriter{
		header: http.Header{},
	}
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

// mockResponder is a Responder and HTTPStatuser used in tests.
type mockResponder struct {
	data        []byte
	contentType string
	status      int
	encodeErr   error
}

func (m mockResponder) Encode() ([]byte, string, error) {
	if m.encodeErr != nil {
		return nil, "", m.encodeErr
	}
	return m.data, m.contentType, nil
}

func (m mockResponder) StatusCode() int { return m.status }

// ----------------------------------------------------------------------
// Tests

func Test_Respond(t *testing.T) {
	tests := []struct {
		name        string
		data        Responder
		statusCode  int
		contentType string
		body        []byte
		shouldErr   bool
	}{
		{
			"NilWriter",
			nil,
			0,
			"",
			nil,
			true,
		},
		{
			"EmptyResponse",
			NoContent,
			204,
			"",
			nil,
			false,
		},
		{
			"ValidResponder",
			mockResponder{[]byte("valid"), "text/plain", 200, nil},
			200,
			"text/plain",
			[]byte("valid"),
			false,
		},
		{
			"ResponderWithEncodeError",
			mockResponder{nil, "", 0, fmt.Errorf("encode err")},
			0,
			"",
			nil,
			true,
		},
	}

	for _, tt := range tests {
		// setup context.
		ctx := context.Background()
		ctx = &writerContext{ctx, writerKey, newMockResponseWriter()}

		t.Run(tt.name, func(t *testing.T) {
			if err := Respond(ctx, tt.data); err != nil && !tt.shouldErr {
				t.Errorf("unexepected error: %q", err)
			}

			w := GetWriter(ctx).(*mockResponseWriter)
			if w.statusCode != tt.statusCode {
				t.Errorf("expected statusCode: [%d]; got [%d]", tt.statusCode, w.statusCode)
			}

			ct := w.header.Get("Content-Type")
			if ct != tt.contentType {
				t.Errorf("exepected content-type: [%s]; got [%s]", tt.contentType, ct)
			}

			if !bytes.Equal(w.body, tt.body) {
				t.Errorf("expected body: [%s]; got [%s]", string(tt.body), string(w.body))
			}
		})
	}
}
