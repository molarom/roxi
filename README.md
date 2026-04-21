# Roxi ![Coverage](https://gitlab.com/romalor/roxi/badges/main/coverage.svg?style=flat-square) [![Docs](https://godoc.org/gitlab.com/romalor/roxi?status.svg)](https://pkg.go.dev/gitlab.com/romalor/roxi)

Roxi is a lightweight http multiplexer or router.

This package borrows inspiration from [Julien Schmidt's httprouter]() and [Daniel Imfeld's httptreemux](https://github.com/dimfeld/httptreemux), in that it uses the same format for path variables and makes use of a PATRICA tree, but the tree and variable handling implementation differs from both.

The aim was to have a mux that meets the following requirements:

1.  A path segment may be variable in one route and a static token in another.
2.  Path values can be retrieved with r.PathValue(<var>)
3.  HandlerFunc's accept a context.Context parameter and return errors.
4.  Provide a simple mux-wide configuration.
5.  Be as performant and memory efficent as possible.
6.  Integrate well with net/http.

## Quick Start

### Install

```bash
go get gitlab.com/romalor/roxi/v2
```

### Example

For more examples and in-depth information, check out the [Documentation](https://pkg.go.dev/gitlab.com/romalor/roxi).

```go
package main

import (
	"context"
	"fmt"
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

## Routing Rules

The routing rules are the same as httptreemux, with the exception of not allowing routes to escape the reserved `:` and `*` variable identifiers.

Path variables in the format `:variable`, will only match a single path segment:

```
Route:
  /foo/:bar

Matches:
  /foo/1
  /foo/baz

Does not match:
  /foo/bar/baz
  /foo/1/bar
```

Wildcards in the format `*wildcard` will match any route suffix following the previous path segment, to include the slash:

```
Route:
  /foo/bar/*wildcard

Matches:
  /foo/bar/
  /foo/bar/baz
  /foo/bar/waldo/fred

Does not match:
  /foo/
```

### Routing Priority

To capture the priority in a TL;DR statement: "Most specific wins, so long as it matches entirely."

When searching for a matching route, the mux does so in the following order:

1. Static routes.
2. Path variables.
3. Wildcards.

For a clearer breakdown of routing decisions:

```
Routes:
- /foo/xzyzz/baz
- /foo/:bar/baz
- /foo/:bar/*wildcard

Non-Matching Requests:
- /foo/xzyzz/ba does not match /foo/xzyzz/baz, missing 'z'.
- /foo/bar/ba does not match /foo/:bar/baz, missing 'z'.
- /foo/bar does not match any pattern, missing final path segment.

Matching Requests:
- /foo/xzyzz/baz matches /foo/xzyzz/baz
- /foo/quux/baz matches /foo/:bar/baz
- /foo/quux/quo matches /foo/:bar/*wildcard
- /foo/bar/ matches /foo/bar/*wildcard
```

### Accessing Variables

Accessing variables is done in the same manner as `net/http`. Simply `r.PathValue("foo")` for any variables wildcard that is registered in your route.

For a more complete represenation:

```go
mux.GET("/foo/:bar", func(ctx context.Context, r *http.Request) error {
    v := r.PathValue("bar")
}
```

A full route registration example can be found within the package documentation.
