// Copyright 2025 Brandon Epperson
// SPDX-License-Identifier: Apache-2.0

package roxi_test

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime/debug"

	"gitlab.com/romalor/roxi"
)

// ----------------------------------------------------------------------
// Mux

func Example_wildcards() {
	// HandlerFunc for route registration.
	h := func(ctx context.Context, r *http.Request) error {
		roxi.GetWriter(ctx).WriteHeader(204)
		return nil
	}

	// Create the mux.
	mux := roxi.New()

	// Register wildcard on the root.
	mux.GET("/*catchall", h)

	// Static tokens will not conflict.
	mux.GET("/foo/bar", h)

	// Neither will path variables, so long as they
	// are on different path segments.
	mux.GET("/foo/:baz", h)

	// Different methods will not conflict.
	mux.POST("/*catchall", h)

	// Another example of path variables not
	// conflicting with static tokens.
	mux.POST("/foo/:bar/:baz/:qux", h)
	mux.POST("/foo/quux/corge/grault", h)

	// Print registered routes.
	for method, routes := range mux.Routes() {
		for _, route := range routes {
			fmt.Printf("%s %s\n", method, route)
		}
	}
	// Unordered output:
	// GET /*catchall
	// GET /foo/:baz
	// GET /foo/bar
	// POST /*catchall
	// POST /foo/:bar/:baz/:qux
	// POST /foo/quux/corge/grault
}

// ----------------------------------------------------------------------
// Mux Options

func ExampleWithOptionsHandler() {
	optHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	}

	// Create new mux with options handler.
	mux := roxi.New(
		roxi.WithOptionsHandler(http.HandlerFunc(optHandler)),
	)

	log.Fatal(http.ListenAndServe(":8080", mux))
}

func ExampleWithCaseInsensitiveRouting() {
	mux := roxi.New(roxi.WithCaseInsensitiveRouting())

	mux.GET("/foo", func(ctx context.Context, r *http.Request) error {
		roxi.GetWriter(ctx).WriteHeader(204)
		return nil
	})

	r, _ := http.NewRequest("GET", "/FOO", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, r)

	fmt.Println(w.Result().Header.Get("Location"), w.Result().StatusCode)
	// Output: /foo 301
}

func ExampleWithRedirectTrailingSlash() {
	mux := roxi.New(roxi.WithRedirectTrailingSlash())

	mux.GET("/foo", func(ctx context.Context, r *http.Request) error {
		roxi.GetWriter(ctx).WriteHeader(204)
		return nil
	})

	r, _ := http.NewRequest("GET", "/foo/", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, r)

	fmt.Println(w.Result().Header.Get("Location"), w.Result().StatusCode)
	// Output: /foo 301
}

func ExampleWithRedirectCleanPath() {
	mux := roxi.New(roxi.WithRedirectCleanPath())

	mux.GET("/foo", func(ctx context.Context, r *http.Request) error {
		roxi.GetWriter(ctx).WriteHeader(204)
		return nil
	})

	r, _ := http.NewRequest("GET", "/..//..///foo", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, r)

	fmt.Println(w.Result().Header.Get("Location"), w.Result().StatusCode)
	// Output: /foo 301
}

func ExampleWithPanicHandler() {
	// Panic handler that returns the stack in the response
	ph := func(ctx context.Context, r *http.Request, err interface{}) {
		w := roxi.GetWriter(ctx)
		fmt.Println(w, err, string(debug.Stack()))
		w.WriteHeader(http.StatusInternalServerError)
	}

	mux := roxi.New(roxi.WithPanicHandler(ph))

	mux.GET("/panic", func(ctx context.Context, r *http.Request) error {
		panic("at the disco")
	})

	log.Fatal(http.ListenAndServe(":8080", mux))
}

func ExampleWithPanicHandler_disabled() {
	mux := roxi.New(roxi.WithPanicHandler(nil))

	mux.GET("/panic", func(ctx context.Context, r *http.Request) error {
		panic("at the disco")
	})

	log.Fatal(http.ListenAndServe(":8080", mux))
}

// ----------------------------------------------------------------------
// File Server

func ExampleMux_FileServer() {
	mux := roxi.New()

	// Serve /tmp under /files
	mux.FileServer("/files/*file", http.Dir(os.TempDir()))

	log.Fatal(http.ListenAndServe(":8080", mux))
}
