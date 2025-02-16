// Copyright 2025 Brandon Epperson
// SPDX-License-Identifier: Apache-2.0

package roxi

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

// ----------------------------------------------------------------------
// Tests

func Test_Mux(t *testing.T) {
	mux := New()

	id := -1
	mux.GET("/user/:userid", func(ctx context.Context, r *http.Request) error {
		if v := r.PathValue("userid"); v != "" {
			id, _ = strconv.Atoi(v)
			return nil
		}
		return fmt.Errorf("failed to get path value")
	})

	w := newMockResponseWriter()
	r, _ := http.NewRequest("GET", "/user/12", nil)

	mux.ServeHTTP(w, r)

	if id != 12 {
		t.Errorf("request failed, user id: %d", id)
	}
}

func Test_HandleFunc(t *testing.T) {
	mux := New()

	id := -1
	mux.HandlerFunc("GET", "/user/:userid", func(w http.ResponseWriter, r *http.Request) {
		if v := r.PathValue("userid"); v != "" {
			id, _ = strconv.Atoi(v)
		}
	})

	w := newMockResponseWriter()
	r, _ := http.NewRequest("GET", "/user/12", nil)

	mux.ServeHTTP(w, r)

	if id != 12 {
		t.Errorf("request failed, user id: %d", id)
	}
}

func Test_Subrouting(t *testing.T) {
	v1 := New()

	v1.GET("/accounts", func(ctx context.Context, r *http.Request) error {
		return Respond(ctx, NoContent)
	})

	mux := New()
	mux.Handler("GET", "/v1/*path", http.StripPrefix("/v1", v1))

	r, _ := http.NewRequest("GET", "/v1/accounts", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, r)

	if w.Result().StatusCode != 204 {
		t.Errorf("failed request to: %s", "/v1/accounts")
	}
}

func Test_MuxMethods(t *testing.T) {
	mux := New()

	for k := range httpMethods {
		mux.Handle(k, "/"+k, func(ctx context.Context, r *http.Request) error {
			return Respond(ctx, NoContent)
		})

		w := httptest.NewRecorder()
		r, _ := http.NewRequest(k, "/"+k, nil)
		mux.ServeHTTP(w, r)

		if w.Result().StatusCode != 204 {
			t.Errorf("request failed with %s %s", k, "/"+k)
		}
	}
}

func Test_NotFound(t *testing.T) {
	mux := New()

	mux.GET("/unused", func(ctx context.Context, r *http.Request) error {
		return InternalServerError(ctx, r)
	})

	r, _ := http.NewRequest("GET", "/unused/", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, r)
	if w.Result().StatusCode != 404 {
		t.Error("failed to fallback to 404 handler")
	}
}

func Test_PanicHandler(t *testing.T) {
	mux := New(WithPanicHandler(DefaultPanicHandler))

	mux.GET("/panic", func(ctx context.Context, r *http.Request) error {
		panic("at the disco")
	})

	r, _ := http.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, r)

	if w.Result().StatusCode != 500 {
		t.Errorf("panic handler did not execute; got status code [%d]", w.Result().StatusCode)
	}
}

func Test_RedirectTrailingSlash(t *testing.T) {
	mux := New(WithRedirectTrailingSlash())

	mux.GET("/redirect", func(ctx context.Context, r *http.Request) error {
		return Respond(ctx, NoContent)
	})

	r, _ := http.NewRequest("GET", "/redirect/", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, r)
	if w.Result().StatusCode != 301 {
		t.Errorf("failed redirect; got status code [%d]", w.Result().StatusCode)
	}
}

func Test_OptionsHandler(t *testing.T) {
	mux := New(WithOptionsHandler(HandlerFunc(DefaultCORS)))

	mux.GET("/unused", func(ctx context.Context, r *http.Request) error {
		return InternalServerError(ctx, r)
	})

	r, _ := http.NewRequest("OPTIONS", "/unused", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, r)
	if w.Result().StatusCode != 204 {
		t.Error("failed to fallback to options handler")
	}
}

func Test_SetAllowHeader(t *testing.T) {
	mux := New(WithSetAllowHeader())

	mux.GET("/unused", func(ctx context.Context, r *http.Request) error {
		return InternalServerError(ctx, r)
	})

	r, _ := http.NewRequest("POST", "/unused", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, r)
	if res := w.Result().Header.Get("Allow"); res != "GET" {
		t.Errorf("expected [%s]; got [%s]", "GET", res)
	}
}

func Test_SetAllowHeaderWithOptions(t *testing.T) {
	mux := New(
		WithOptionsHandler(HandlerFunc(DefaultCORS)),
		WithSetAllowHeader(),
	)

	mux.GET("/unused", func(ctx context.Context, r *http.Request) error {
		return InternalServerError(ctx, r)
	})

	r, _ := http.NewRequest("POST", "/unused", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, r)

	res := w.Result().Header.Get("Allow")

	// ordering isn't guaranteed.
	if !(res == "OPTIONS,GET" || res == "GET,OPTIONS") {
		t.Errorf("expected [%s]; got [%s]", "GET,OPTIONS", res)
	}
}

func Test_Middleware(t *testing.T) {
	middleware := false
	mw := func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, r *http.Request) error {
			middleware = true
			return next(ctx, r)
		}
	}

	mux := New(WithMiddleware(mw))
	mux.GET("/middleware", func(ctx context.Context, r *http.Request) error {
		return Respond(ctx, NoContent)
	})

	r, _ := http.NewRequest("GET", "/middleware", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, r)
	if !middleware {
		t.Error("middleware failed to execute")
	}
}

func Test_RedirectCleanPath(t *testing.T) {
	mux := New(WithRedirectCleanPath())

	mux.GET("/redirect", func(ctx context.Context, r *http.Request) error {
		return Respond(ctx, NoContent)
	})

	r, _ := http.NewRequest("GET", "/../../redirect", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, r)
	if w.Result().StatusCode != 301 {
		t.Errorf("failed redirect; got status code [%d]", w.Result().StatusCode)
	}
}

// ----------------------------------------------------------------------
// Benchmark Data

var verbs = []string{
	"accounts",
	"users",
	"settings",
	"static",
	"auth",
	"oauth",
	"view",
	"views",
	"list",
	"stats",
	"statistics",
	"metrics",
	"home",
	"help",
	"contact",
	"address",
}

// generateRoutes generates a list of static routes from the verbs provided.
func generateRoutes(prefix string, verbs []string) []string {
	routes := make([]string, 0, len(verbs)*len(verbs))

	for _, v := range verbs {
		for _, s := range verbs {
			routes = append(routes, fmt.Sprintf("%s/%s/%s", prefix, v, s))
		}
	}

	return routes
}

// ----------------------------------------------------------------------
// Benchmarks

func Benchmark_Load(b *testing.B) {
	routes := generateRoutes("/v1", verbs)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		mux := NewWithDefaults()
		for _, r := range routes {
			mux.GET(r, func(ctx context.Context, r *http.Request) error { return nil })
		}
	}
}

func Benchmark_Routing(b *testing.B) {
	routes := generateRoutes("/v1", verbs)

	mux := NewWithDefaults()
	for _, r := range routes {
		mux.GET(r, func(ctx context.Context, r *http.Request) error { return nil })
	}

	w := newMockResponseWriter()
	req, _ := http.NewRequest("GET", "/", nil)
	u := req.URL
	q := req.URL.RawQuery

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, r := range routes {
			req.Method = "GET"
			req.RequestURI = r
			u.Path = r
			u.RawQuery = q

			mux.ServeHTTP(w, req)
		}
	}
}
