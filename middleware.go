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
	validMW := make([]MiddlewareFunc, 0, len(mw))
	for _, m := range mw {
		if m != nil {
			validMW = append(validMW, m)
		}
	}

	// no middleware, immediately return handler
	if len(validMW) == 0 {
		return handler
	}

	// single middleware, wrap and return.
	if len(validMW) == 1 {
		return validMW[0](handler)
	}

	// > 1, build chain.
	for i := len(validMW) - 1; i >= 0; i-- {
		handler = validMW[i](handler)
	}
	return handler
}
