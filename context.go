// Copyright 2025 Brandon Epperson
// SPDX-License-Identifier: Apache-2.0

package roxi

import (
	"context"
	"net/http"
)

type ctxKey int

const (
	writerKey ctxKey = 0
)

// writerContext stores the http.ResponseWriter to pass to HandlerFuncs.
type writerContext struct {
	context.Context
	value http.ResponseWriter
}

func (c *writerContext) Value(key any) any {
	if key == writerKey {
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
