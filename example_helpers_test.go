// Copyright 2025 Brandon Epperson
// SPDX-License-Identifier: Apache-2.0

package roxi_test

import (
	"context"
	"log"
	"net/http"

	"gitlab.com/romalor/roxi"
)

// HomePage implements the Responder interface.
type HomePage []byte

func (r HomePage) Response() ([]byte, string, error) {
	return r, "text/plain; charset=utf-8", nil
}

func Root(ctx context.Context, r *http.Request) error {
	return roxi.Redirect(ctx, r, "/home", 301)
}

func Home(ctx context.Context, r *http.Request) error {
	return roxi.Respond(ctx, HomePage("Welcome!"))
}

func Example_helpers() {
	mux := roxi.New()

	mux.GET("/", Root)
	mux.GET("/home", Home)

	log.Fatal(http.ListenAndServe(":8080", mux))
}
