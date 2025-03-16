// Copyright 2025 Brandon Epperson
// SPDX-License-Identifier: Apache-2.0

package roxi

import (
	"context"
	"net/http"
	"strconv"
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
	Origins:     []string{"*"},
	Methods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
	Expose:      []string{"Content-Encoding"},
	Headers:     []string{"Accept", "Content-Type", "Content-Length", "Accept-Encoding", "Origin", "Authorization"},
	Credentials: true,
	MaxAge:      86400,
	Vary:        true,
}

// DefaultCORS is an optional OPTIONS handler with reasonable defaults set for responding to preflight requests.
//
// Values Set:
//
//	CORS{
//	    Origins:     []string{"*"},
//	    Methods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
//	    Expose:      []string{"Content-Encoding"},
//	    Headers:     []string{"Accept", "Content-Type", "Content-Length", "Accept-Encoding", "Origin", "Authorization"},
//	    Credentials: true,
//	    MaxAge:      86400,
//	    Vary:        true,
//	}
var DefaultCORS = defaultCORS.HandlerFunc()

// Handler returns a request handler for preflight requests.
func (c CORS) HandlerFunc() HandlerFunc {
	// create the joined strings on initialization
	expose := strings.Join(c.Expose, ", ")
	methods := strings.Join(c.Methods, ", ")
	headers := strings.Join(c.Headers, ", ")

	return func(ctx context.Context, r *http.Request) error {
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

		if c.MaxAge != 0 {
			w.Header().Set("Access-Control-Max-Age", strconv.Itoa(c.MaxAge))
		}

		if c.Credentials {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if expose != "" {
			w.Header().Set("Access-Control-Expose-Headers", expose)
		}

		if methods != "" {
			w.Header().Set("Access-Control-Allow-Methods", methods)
		}

		if headers != "" {
			w.Header().Set("Access-Control-Allow-Headers", headers)
		}

		return Respond(ctx, NoContent)
	}
}
