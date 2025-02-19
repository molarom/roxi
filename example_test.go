// Copyright 2025 Brandon Epperson
// SPDX-License-Identifier: Apache-2.0

package roxi_test

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"

	"gitlab.com/romalor/roxi"
)

// ----------------------------------------------------------------------
// Mux Options

func ExampleWithLogger() {
	// Use slog.Info instead of log.Printf
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

func ExampleWithOptionsHandler_gorillaCORS() {
	// Empty HandlerFunc to pass to Gorilla handler.
	h := roxi.HandlerFunc(func(context.Context, *http.Request) error {
		return nil
	})

	mux := roxi.New(
		roxi.WithOptionsHandler(handlers.CORS()(h)),
	)

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
