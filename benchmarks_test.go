package roxi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

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
	":path",
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
// Alloc Tests

func Test_MuxRoutingAllocs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping alloc tests in short mode.")
	}

	routes := generateRoutes("/v1", verbs)

	mux := New()
	for _, r := range routes {
		mux.GET(r, func(ctx context.Context, r *http.Request) error { return nil })
	}

	w := newMockResponseWriter()
	req, _ := http.NewRequest("GET", "/", nil)
	u := req.URL
	q := req.URL.RawQuery

	for _, r := range routes {
		req.Method = "GET"
		req.RequestURI = r
		u.Path = r
		u.RawQuery = q

		allocs := testing.AllocsPerRun(100, func() { mux.ServeHTTP(w, req) })
		if allocs > 0 {
			t.Errorf("mux.ServeHTTP(): expected zero allocs; got [%v]", allocs)
		}
	}
}

func Benchmark_Mux(b *testing.B) {
	muxes := []struct {
		name   string
		mux    http.Handler
		method string
		path   string
	}{
		// {
		// 	"Single",
		// 	buildMux(singleRoute{}),
		// 	http.MethodGet,
		// 	"/path",
		// },
		// {
		// 	"Many",
		// 	buildMux(manyRoutes{}),
		// 	http.MethodGet,
		// 	"/v1/path/path",
		// },
		// {
		// 	"SingleWithMiddleware",
		// 	buildMux(singleRoute{}, WithMiddleware(mw("1"))),
		// 	http.MethodGet,
		// 	"/path",
		// },
		// {
		// 	"ManyWithMiddleware",
		// 	buildMux(manyRoutes{}, WithMiddleware(mw("1"))),
		// 	http.MethodGet,
		// 	"/v1/path/path",
		// },
		// {
		// 	"SingleWithManyMiddleware",
		// 	buildMux(singleRoute{}, WithMiddleware(mw("1"), mw("2"), mw("3"), mw("4"))),
		// 	http.MethodGet,
		// 	"/path",
		// },
		// {
		// 	"ManyWithManyMiddleware",
		// 	buildMux(manyRoutes{}, WithMiddleware(mw("1"), mw("2"), mw("3"), mw("4"))),
		// 	http.MethodGet,
		// 	"/v1/path/path",
		// },
		// {
		// 	"Params",
		// 	buildMux(paramsRoute{}),
		// 	http.MethodGet,
		// 	"/path/banana/banana/banana/terracotta/pie",
		// },
		// {
		// 	"NotFound",
		// 	buildMux(singleRoute{}),
		// 	http.MethodGet,
		// 	"/foo/bar",
		// },
		{
			"MethodNotAllowed",
			buildMux(singleRoute{}),
			http.MethodPost,
			"/path",
		},
		{
			"MethodNotAllowedAllMethods",
			buildMux(allbutPOST{}),
			http.MethodPost,
			"/path",
		},
		//{
		//	"OptionsAllMethods",
		//	buildMux(allbutPOST{}, WithOptionsHandler(DefaultCORS)),
		//	http.MethodPost,
		//	"/path",
		//},
		// {
		// 	"RespondBytes",
		// 	buildMux(bRespRoute{}),
		// 	http.MethodGet,
		// 	"/path",
		// },
		// {
		// 	"RespondJSON",
		// 	buildMux(jsonRespRoute{}),
		// 	http.MethodGet,
		// 	"/path",
		// },
	}

	for _, tt := range muxes {
		r, _ := http.NewRequest(tt.method, tt.path, nil)
		w := newMockResponseWriter()
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				tt.mux.ServeHTTP(w, r)
			}
		})
	}
}

// func Benchmark_MuxParallel(b *testing.B) {
// 	muxes := []struct {
// 		name string
// 		mux  http.Handler
// 	}{
// 		{
// 			"Many",
// 			buildMux(manyRoutes{}),
// 		},
// 		{
// 			"ManyWithManyMiddleware",
// 			buildMux(manyRoutes{}, WithMiddleware(mw("1"), mw("2"), mw("3"), mw("4"))),
// 		},
// 	}
//
// 	for _, tt := range muxes {
// 		b.RunParallel(func(p *testing.PB) {
// 			for p.Next() {
// 				r, _ := http.NewRequest(http.MethodGet, "/v1/path/path", nil)
// 				w := httptest.NewRecorder()
//
// 				tt.mux.ServeHTTP(w, r)
// 			}
// 		})
// 	}
// }

// ----------------------------------------------------------------------
// Responders

type bytesResp []byte

func (r bytesResp) Response() ([]byte, string, error) {
	return r, "text/html; charset=utf-8", nil
}

type jsonResp struct {
	Status string `json:"status"`
}

func (r jsonResp) Response() ([]byte, string, error) {
	data, err := json.Marshal(r)
	return data, "application/json; charset=utf-8", err
}

// ----------------------------------------------------------------------
// Routes

type singleRoute struct{}

func (r singleRoute) Add(mux *Mux) {
	mux.GET("/path", func(ctx context.Context, r *http.Request) error { return nil })
}

type paramsRoute struct{}

func (r paramsRoute) Add(mux *Mux) {
	mux.GET("/path/:foo/:bar/:baz/:qux/:quux", func(ctx context.Context, r *http.Request) error { return nil })
}

type bRespRoute struct{}

func (r bRespRoute) Add(mux *Mux) {
	mux.GET("/path", func(ctx context.Context, r *http.Request) error {
		return Respond(ctx, bytesResp("ok"))
	})
}

type jsonRespRoute struct{}

func (r jsonRespRoute) Add(mux *Mux) {
	mux.GET("/path", func(ctx context.Context, r *http.Request) error {
		return Respond(ctx, jsonResp{"ok"})
	})
}

type manyRoutes struct{}

func (r manyRoutes) Add(mux *Mux) {
	routes := generateRoutes("/v1", verbs)
	for _, r := range routes {
		mux.GET(r, func(ctx context.Context, r *http.Request) error { return nil })
	}
}

type allbutPOST struct{}

func (r allbutPOST) Add(mux *Mux) {
	for v := range httpMethods {
		if v != "POST" {
			mux.Handle(v, "/path", func(ctx context.Context, r *http.Request) error { return nil })
		}
	}
}

// ----------------------------------------------------------------------
// Middleware

func mw(value string) MiddlewareFunc {
	return func(handle HandlerFunc) HandlerFunc {
		return func(ctx context.Context, r *http.Request) error {
			fmt.Fprintf(GetWriter(ctx), "%s", value)
			return nil
		}
	}
}

// ----------------------------------------------------------------------
// Mux builders

type routeAdder interface {
	Add(mux *Mux)
}

func buildMux(routes routeAdder, opts ...func(*Mux)) *Mux {
	mux := New(opts...)

	routes.Add(mux)
	return mux
}
