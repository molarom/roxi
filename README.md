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

## Routing Rules

The routing rules are the same as httptreemux, with the exception of allowing routes to escape the reserved `:` and `*` variable identifiers.

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

Futher explaination:
1. Static routes.
2. Path variables.
3. Wildcards.

Meaning:
```
Routes:
- /foo/xzyzz/baz
- /foo/:bar/baz
- /foo/:bar/*wildcard

Requests:
- /foo/xzyzz/baz only matches /foo/xzyzz/baz
- /foo/quux/baz only matches /foo/:bar/baz
- /foo/quux/quo only matches /foo/:bar/*wildcard
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

## Middleware

Roxi supports middleware out of the box by allowing a variadic argument that accepts MiddlewareFuncs on each method 
that registers a Handler or HandlerFunc on the mux. 

The mux can also hold global middleware that will execute for every handler when passed to the `WithMiddleware()` function upon creation.

This package does not provide any middleware for you, but writing or integrating middleware provided by other frameworks, such
as gorilla is fairly straightforward.

### Examples

Roxi specific examples can be found within the package documentation.

Gorilla (Registered as HTTP Handler):
```go
package main

import (
	"context"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"gitlab.com/romalor/roxi"
)

func main() {
	mux := roxi.New()

    // Empty HandlerFunc for example.
	h := roxi.HandlerFunc(func(ctx context.Context, r *http.Request) error { 
        return nil 
    })

    // Register on the mux with the gorilla.LoggingHandler middleware.
	mux.Handler("GET", "/logged", handlers.LoggingHandler(os.Stdout, h))

}
```

Gorilla (Registered As MiddlewareFunc):
```go
package example

import (
	"context"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"gitlab.com/romalor/roxi"
)

// Create roxi.MiddlewareFunc to wrap Gorilla LoggingHandler.
func RoxiGorillaLogger(next roxi.HandlerFunc) roxi.HandlerFunc {
	return func(ctx context.Context, r *http.Request) error {
		handlers.LoggingHandler(os.Stdout, next)
		return nil
	}
}

func main() {
	mux := roxi.New()

	// empty handlerFunc for example.
	h := func(ctx context.Context, r *http.Request) error {
		return nil
	}

	// Register at the handler.
	mux.GET("/logged", h, RoxiGorillaLogger)
	// OR
	// Register for an entire mux:
	_ = roxi.New(roxi.WithMiddleware(RoxiGorillaLogger))
}
```

## Performance 

### Benchmarks

Ran with `make bench`. Benchmarks not related to the Mux were omitted. Other http routers were added by manually updating the mux in the `Benchmark_Load` and `Benchmark_Routing` tests. This will be tidied up eventually and currently serves as a (very) rough comparison.

**NOTE:** Only httptreemux supports the same variable patterns created by the `generateRoutes` function.

TL;DR:
Router | Static Routes | Variables
--- | --- | ---
httprouter | 1 | N/A
roxi | 2 | 1
httptreemux | 3 | 2
net/http | 4 | N/A


#### Hardware
```
goos: darwin
goarch: arm64
pkg: gitlab.com/romalor/roxi
cpu: Apple M4 Pro
```

#### Roxi (Static Only)

```
Benchmark_Load-12                  39248             30375 ns/op           47600 B/op       1166 allocs/op
Benchmark_Routing-12              136495              8536 ns/op               0 B/op          0 allocs/op
```

#### Roxi (Variables)

```
Benchmark_Load-12                  35451             33798 ns/op           52160 B/op       1288 allocs/op
Benchmark_Routing-12              118816              9971 ns/op               0 B/op          0 allocs/op
```

#### httptreemux (Static Only)

```
Benchmark_Load-12                  23160             51959 ns/op          122881 B/op       1453 allocs/op
Benchmark_Routing-12              129406              9103 ns/op               0 B/op          0 allocs/op
```

#### httptreemux (Variables)

```
Benchmark_Load-12                  20521             57921 ns/op          137537 B/op       1626 allocs/op
Benchmark_Routing-12               92210             12808 ns/op           11648 B/op        100 allocs/op
```

#### httprouter 

```
Benchmark_Load-12                  47868             25041 ns/op           37416 B/op        959 allocs/op
Benchmark_Routing-12              238778              4973 ns/op               0 B/op          0 allocs/op
```

#### net/http

```
Benchmark_Load-12                   5448            216450 ns/op          240090 B/op       3163 allocs/op
Benchmark_Routing-12               35581             33443 ns/op               0 B/op          0 allocs/op
```
