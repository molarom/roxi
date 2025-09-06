// Copyright 2025 Brandon Epperson
// SPDX-License-Identifier: Apache-2.0

package roxi

// MiddlewareFunc represents HandlerFuncs that are chained in execution.
type MiddlewareFunc func(handle HandlerFunc) HandlerFunc

// MiddlewareStack represents a group of MiddlewareFunc to execute in sequence.
//
// Execution order will be in the order of arguments provided, for example:
//
//	MiddlewareStack(handler,
//	                LoggingMW, // 1
//	                ErrorsMW)  // 2
func MiddlewareStack(handler HandlerFunc, mw ...MiddlewareFunc) HandlerFunc {
	for i := len(mw) - 1; i >= 0; i-- {
		mwFn := mw[i]
		if mwFn != nil {
			handler = mwFn(handler)
		}
	}
	return handler
}
