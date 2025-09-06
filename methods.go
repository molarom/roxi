// Copyright 2025 Brandon Epperson
// SPDX-License-Identifier: Apache-2.0

package roxi

import (
	"net/http"
	"strings"
)

// method flags for caching allowed methods for routes.
type methodFlag uint16

const (
	GET methodFlag = 1 << iota
	HEAD
	POST
	PUT
	PATCH
	DELETE
	CONNECT
	OPTIONS
	TRACE
)

var httpMethods = map[string]methodFlag{
	http.MethodGet:     GET,
	http.MethodHead:    HEAD,
	http.MethodPost:    POST,
	http.MethodPut:     PUT,
	http.MethodPatch:   PATCH,
	http.MethodDelete:  DELETE,
	http.MethodConnect: CONNECT,
	http.MethodOptions: OPTIONS,
	http.MethodTrace:   TRACE,
}

// String implements the fmt.Stringer interface.
func (m methodFlag) String() string {
	switch m {
	case GET:
		return "GET"
	case HEAD:
		return "HEAD"
	case POST:
		return "POST"
	case PUT:
		return "PUT"
	case PATCH:
		return "PATCH"
	case DELETE:
		return "DELETE"
	case CONNECT:
		return "CONNECT"
	case OPTIONS:
		return "OPTIONS"
	case TRACE:
		return "TRACE"
	}

	methods := make([]string, 0, 9)
	for method := GET; method <= TRACE; method <<= 1 {
		if m&method != 0 {
			methods = append(methods, method.String())
		}
	}
	return strings.Join(methods, ", ")
}
