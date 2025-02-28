// Copyright 2025 Brandon Epperson
// SPDX-License-Identifier: Apache-2.0

package roxi

import (
	"context"
	"net/http"
)

// PanicHandler represents a function to recover from panics that may
// occur during the lifecycle of the mux.
type PanicHandler func(ctx context.Context, r *http.Request, err interface{})
