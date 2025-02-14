// Copyright 2025 Brandon Epperson
// SPDX-License-Identifier: Apache-2.0

package roxi

import (
	"context"
	"net/http"
)

type ctxKey int

const (
	writerKey ctxKey = iota
)

// setWriter adds an http.ResponseWriter to the context.
func setWriter(ctx context.Context, w http.ResponseWriter) context.Context {
	if _, ok := ctx.Value(writerKey).(http.ResponseWriter); !ok {
		return context.WithValue(ctx, writerKey, w)
	}

	return ctx
}

// GetWriter returns the http.ResponseWriter from the context.
func GetWriter(ctx context.Context) http.ResponseWriter {
	v, ok := ctx.Value(writerKey).(http.ResponseWriter)
	if !ok {
		return nil
	}
	return v
}
