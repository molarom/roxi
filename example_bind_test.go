// Copyright 2025 Brandon Epperson
// SPDX-License-Identifier: Apache-2.0

package roxi_test

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"gitlab.com/romalor/roxi"
)

// RequestBody implements the Binder and Validator interfaces.
type RequestBody []byte

func (r *RequestBody) Bind(data []byte) error {
	*r = data
	return nil
}

func (r RequestBody) Validate() error {
	if r != nil {
		return nil
	}
	return fmt.Errorf("requestbody cannot be nil")
}

func BindHandler(ctx context.Context, r *http.Request) error {
	b := &RequestBody{}
	if err := roxi.Bind(r, b); err != nil {
		return err
	}
	return roxi.Respond(ctx, roxi.NoContent)
}

func Example_binder() {
	mux := roxi.New()

	mux.GET("/", BindHandler)

	log.Fatal(http.ListenAndServe(":8080", mux))
}
