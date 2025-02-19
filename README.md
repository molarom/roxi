# Roxi [![Coverage](https://gitlab.com/romalor/roxi/badges/main/coverage.svg?style=flat-square)]

Roxi is a lightweight, zero alloc http multiplexer or router.

## Quick Start

### Minimal

```go
package main

import (
	"context"
	"log"
	"net/http"

	"gitlab.com/romalor/roxi"
)

// HomePage implements the Responder interface
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

func main() {
	mux := roxi.NewWithDefaults()

	mux.GET("/", Root)
	mux.GET("/home", Home)

	log.Fatal(http.ListenAndServe(":8080", mux))
}
```

### No helper methods

```go
package main

import (
	"context"
	"log"
	"net/http"

	"gitlab.com/romalor/roxi"
)

func Root(ctx context.Context, r *http.Request) error {
	http.Redirect(GetWriter(ctx), r, "/home", 301)
    return nil
}

func Home(ctx context.Context, r *http.Request) error {
    // Error handling is optional here since we're writing directly to the writer, 
    // but the mux will still log the error to help with further troubleshooting.
    if _, err := fmt.Fprintf(GetWriter(ctx), "Welcome!"); err != nil {
        return err
    }
    return nil
}

func main() {
	mux := roxi.NewWithDefaults()

	mux.GET("/", Root)
	mux.GET("/home", Home)

	log.Fatal(http.ListenAndServe(":8080", mux))
}
```
