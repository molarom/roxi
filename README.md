# Roxi ![Coverage](https://gitlab.com/romalor/roxi/badges/main/coverage.svg?style=flat-square) [![Docs](https://godoc.org/gitlab.com/romalor/roxi?status.svg)](https://pkg.go.dev/gitlab.com/romalor/roxi)

Roxi is a lightweight http multiplexer or router.

This package borrows inspiration from [Julien Schmidt's httprouter]() and [Daniel Imfeld's httptreemux](https://github.com/dimfeld/httptreemux), in that it uses the same format for path variables and makes use of a PATRICA tree, but the tree and variable handling implementation differs from both.

The aim was to have a mux that meets the following requirements:
 1. A path segment may be variable in one route and a static token in another.
 2. Path values can be retrieved with r.PathValue(<var>)
 3. HandlerFunc's accept a context.Context parameter and return errors.
 4. Provide a simple mux-wide configuration.
 5. Be as performant and memory efficent as possible.
 6. Integrate well with net/http.

There are some additional methods included in this package that may optionally be used to improve developer experience, such as Bind and Respond for handling request
and response data respectively. These components were inspired by [Bill Kennedy's Service project](https://github.com/ardanlabs/service).

## Quick Start

### Install

```bash
go get gitlab.com/romalor/roxi@v1
```

### Example

For more examples and in-depth information, check out the [Documentation](https://pkg.go.dev/gitlab.com/romalor/roxi).

```go
package main

import (
	"context"
	"log"
	"net/http"

	"gitlab.com/romalor/roxi"
)

func Root(ctx context.Context, r *http.Request) error {
	http.Redirect(roxi.GetWriter(ctx), r, "/home", 301)
    return nil
}

func Home(ctx context.Context, r *http.Request) error {
	// Error handling is optional here since we're writing directly to the writer,
	// but the mux will still log the error to help with further troubleshooting if
	// one is returned.
    if _, err := fmt.Fprintf(roxi.GetWriter(ctx), "Welcome!"); err != nil {
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
