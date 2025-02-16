// Copyright 2025 Brandon Epperson
// SPDX-License-Identifier: Apache-2.0

package roxi

import "net/http"

// map of http methods defined in net/http.
var httpMethods = map[string]string{
	"GET":     http.MethodGet,
	"HEAD":    http.MethodHead,
	"POST":    http.MethodPost,
	"PUT":     http.MethodPut,
	"PATCH":   http.MethodPatch,
	"DELETE":  http.MethodDelete,
	"CONNECT": http.MethodConnect,
	"OPTIONS": http.MethodOptions,
	"TRACE":   http.MethodTrace,
}
