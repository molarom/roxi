// Copyright 2025 Brandon Epperson
// SPDX-License-Identifier: Apache-2.0

package roxi

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
)

// PanicHandler represents a function to recover from panics that may
// occur during the lifecycle of the mux.
type PanicHandler func(ctx context.Context, r *http.Request, err interface{})

// DefaultPanicHandler is an optional handler that executes when a panic is recovered.
var DefaultPanicHandler = func(ctx context.Context, r *http.Request, err interface{}) {
	buf := make([]byte, 65536)
	buf = buf[:runtime.Stack(buf, false)]
	fmt.Printf("roxi: recovered panic %v: %s\n", err, buf)
	GetWriter(ctx).WriteHeader(http.StatusInternalServerError)
}
