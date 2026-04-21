package roxi_test

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"gitlab.com/romalor/roxi/v2"
)

func Index(ctx context.Context, r *http.Request) error {
	http.Redirect(roxi.GetWriter(ctx), r, "/home", http.StatusMovedPermanently)
	return nil
}

func Welcome(ctx context.Context, r *http.Request) error {
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
