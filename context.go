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

// writerContext stores the http.ResponseWriter to pass to HandlerFuncs.
type writerContext struct {
	context.Context
	value http.ResponseWriter
}

func (c writerContext) Value(key any) any {
	if key == writerKey {
		return c.value
	}
	return c.Context.Value(key)
}

// GetWriter returns the http.ResponseWriter from the context.
func GetWriter(ctx context.Context) http.ResponseWriter {
	if v, ok := ctx.(*writerContext); ok {
		return v.value
	}
	return getWriterFallback(ctx)
}

//go:noinline
func getWriterFallback(ctx context.Context) http.ResponseWriter {
	if v, ok := ctx.Value(writerKey).(http.ResponseWriter); ok {
		return v
	}
	return nil
}

// SetWriter allows setting a custom http.ResponseWriter in the context.
func SetWriter(ctx context.Context, w http.ResponseWriter) context.Context {
	v, ok := ctx.(*writerContext)
	if !ok {
		if ctx == nil {
			ctx = context.Background()
		}
		return writerContext{ctx, w}
	}

	v.value = w
	return v
}
