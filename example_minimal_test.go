package roxi_test

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"gitlab.com/romalor/roxi"
)

func Index(ctx context.Context, r *http.Request) error {
	http.Redirect(roxi.GetWriter(ctx), r, "/home", 301)
	return nil
}

func Welcome(ctx context.Context, r *http.Request) error {
	// Error handling is optional here since we're writing directly to the writer,
	// but the mux will still log the error to help with further troubleshooting if
	// one is returned.
	if _, err := fmt.Fprintf(roxi.GetWriter(ctx), "Welcome!"); err != nil {
		return err
	}
	return nil
}

func Example() {
	mux := roxi.NewWithDefaults()

	mux.GET("/", Index)
	mux.GET("/home", Welcome)

	log.Fatal(http.ListenAndServe(":8080", mux))
}
