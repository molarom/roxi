// Copyright 2025 Brandon Epperson
// SPDX-License-Identifier: Apache-2.0

package roxi_test

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime/debug"
	"time"

	"gitlab.com/romalor/roxi"
)

// ----------------------------------------------------------------------
// Mux

func Example_wildcards() {
	// HandlerFunc for route registration.
	h := func(ctx context.Context, r *http.Request) error {
		return roxi.Respond(ctx, roxi.NoContent)
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
	mux.PrintRoutes()
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

func ExampleWithLogger() {
	// Use slog.Info instead of log.Print
	mux := roxi.New(
		roxi.WithLogger(slog.Info),
	)

	log.Fatal(http.ListenAndServe(":8080", mux))
}

func ExampleWithMiddleware() {
	// Simple logging middleware
	mw := func(next roxi.HandlerFunc) roxi.HandlerFunc {
		return func(ctx context.Context, r *http.Request) error {
			before := time.Now()
			log.Print("time before:", before)

			if err := next(ctx, r); err != nil {
				return err
			}

			after := time.Since(before)
			log.Print("time after:", after)

			return nil
		}
	}

	// Create mux and register middleware
	mux := roxi.New(
		roxi.WithMiddleware(mw),
	)

	// Simple HandlerFunc
	mux.GET("/", func(ctx context.Context, r *http.Request) error {
		return roxi.Respond(ctx, roxi.NoContent)
	})

	log.Fatal(http.ListenAndServe(":8080", mux))
}

func ExampleWithOptionsHandler() {
	// Setup CORS
	cors := roxi.CORS{
		Origins: []string{"your.domain"},
		Headers: []string{"Authorization"},
		Methods: []string{"GET"},
	}

	// Create new mux with options handler.
	mux := roxi.New(
		roxi.WithOptionsHandler(cors.HandlerFunc()),
	)

	log.Fatal(http.ListenAndServe(":8080", mux))
}

func ExampleWithCaseInsensitiveRouting() {
	mux := roxi.New(roxi.WithCaseInsensitiveRouting())

	mux.GET("/foo", func(ctx context.Context, r *http.Request) error {
		return roxi.Respond(ctx, roxi.NoContent)
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
		return roxi.Respond(ctx, roxi.NoContent)
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
		return roxi.Respond(ctx, roxi.NoContent)
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
		fmt.Println(w, err, debug.Stack())
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

func ExampleMux_FileServerRE() {
	mux := roxi.New()

	// Serve all files in /tmp ending in '.html' or '.js' under /files
	mux.FileServerRE("/files/*file", `.*\.(html|js)$`, http.Dir(os.TempDir()))

	log.Fatal(http.ListenAndServe(":8080", mux))
}
