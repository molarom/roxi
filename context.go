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
	key   ctxKey
	value http.ResponseWriter
}

func (c *writerContext) Value(key any) any {
	if key == c.key {
		return c.value
	}
	return c.Context.Value(key)
}

// GetWriter returns the http.ResponseWriter from the context.
func GetWriter(ctx context.Context) http.ResponseWriter {
	v, ok := ctx.Value(writerKey).(http.ResponseWriter)
	if !ok {
		return nil
	}
	return v
}
