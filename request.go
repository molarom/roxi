// Copyright 2025 Brandon Epperson
// SPDX-License-Identifier: Apache-2.0

package roxi

import (
	"fmt"
	"io"
	"net/http"
)

// Binder represents a type that an http.Request body can be bound to.
type Binder interface {
	Bind(data []byte) error
}

// Validator is an optional interface a Binder can implement to
// run additional validation for a bound type.
type Validator interface {
	Validate() error
}

// Bind attempts to bind an http.Request body to a Request.
func Bind(r *http.Request, v Binder) error {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("bind: failed to read payload: %w", err)
	}

	if err := v.Bind(data); err != nil {
		return fmt.Errorf("bind: %w", err)
	}

	if v, ok := v.(Validator); ok {
		if err := v.Validate(); err != nil {
			return err
		}
	}

	return nil
}
