// Copyright 2025 Brandon Epperson
// SPDX-License-Identifier: Apache-2.0

package roxi

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// CORS represents the options for configuring a Handler to respond to preflight requests.
type CORS struct {
	// Allowed origins, a match sets the Access-Control-Allow-Origin header.
	Origins []string

	// Set the Vary header to "Origin".
	Vary bool

	// Headers to set in Access-Control-Expose-Headers header.
	Expose []string

	// Value for Access-Control-Max-Age header.
	MaxAge int

	// Set Access-Control-Allow-Credentials header.
	Credentials bool

	// Methods for Access-Control-Allow-Methods header.
	Methods []string

	// Methods for Access-Control-Allow-Headers.
	Headers []string
}

var defaultCORS = CORS{
	Origins: []string{"*"},
	Methods: []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
	Headers: []string{"Accept", "Content-Type", "Content-Length", "Accept-Encoding", "Origin", "Authorization"},
	MaxAge:  86400,
	Vary:    false,
}

// DefaultCORS is an OPTIONS handler with reasonable defaults set for responding to preflight requests.
//
// Values Set:
//
//	CORS{
//		Origins: []string{"*"},
//		Methods: []string{"GET", "POST", "PUT", "DELETE", "UPDATE"},
//		Headers: []string{"Accept", "Content-Type", "Content-Length", "Accept-Encoding", "Origin", "Authorization"},
//		MaxAge:  86400,
//		Vary:    false,
//	}
var DefaultCORS = defaultCORS.Handler

// Handler is OPTIONS request handler for handling preflight requests.
func (c CORS) Handler(ctx context.Context, r *http.Request) error {
	w := GetWriter(ctx)
	reqOrigin := r.Header.Get("Origin")
	for _, o := range c.Origins {
		if o == "*" || o == reqOrigin {
			w.Header().Set("Access-Control-Allow-Origin", o)
			if c.Vary {
				w.Header().Set("Vary", "Origin")
			}
			break
		}
	}

	if expose := strings.Join(c.Expose, ", "); expose != "" {
		w.Header().Set("Access-Control-Expose-Headers", expose)
	}

	if c.MaxAge != 0 {
		w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", c.MaxAge))
	}

	if c.Credentials {
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	if methods := strings.Join(c.Methods, ", "); methods != "" {
		w.Header().Set("Access-Control-Allow-Methods", methods)
	}

	if headers := strings.Join(c.Headers, ", "); headers != "" {
		w.Header().Set("Access-Control-Allow-Headers", headers)
	}

	return Respond(ctx, NoContent)
}
