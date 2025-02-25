// Copyright 2025 Brandon Epperson
// SPDX-License-Identifier: Apache-2.0

package roxi

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

var emptyHandler = func(ctx context.Context, r *http.Request) error {
	return nil
}

func Test_ParseParams(t *testing.T) {
	tests := []struct {
		name    string
		wcPath  string
		path    string
		params  []string
		lastIdx int
		ok      bool
	}{
		{
			"Parse",
			"/path/:with/:param",
			"/path/sub/subsub",
			[]string{
				"with",
				"param",
			},
			15,
			true,
		},
		{
			"MismatchedSegments",
			"/user/group/:group_id",
			"/user/group",
			[]string{},
			11,
			false,
		},
		{
			"MismatchedNoLeadingSlash",
			":path",
			"foo/bar",
			[]string{},
			3,
			false,
		},
		{
			"MatchNoLeadingSlash",
			"foo/:bar",
			"foo/bar",
			[]string{"bar"},
			7,
			true,
		},
		{
			"WithTrailingSlash",
			":bar/",
			"foo/",
			[]string{"bar"},
			4,
			true,
		},
		{
			"MatchPartialMiddle",
			"/path/:bar/b",
			"/path/s/baz",
			[]string{"bar"},
			9,
			true,
		},
		{
			"WildcardShort",
			"/path/*wildcard",
			"/path/single",
			[]string{"wildcard"},
			11,
			true,
		},
		{
			"WildcardLong",
			"/path/*wildcard",
			"/path" + strings.Repeat("/path", 80),
			[]string{"wildcard"},
			404,
			true,
		},
		{
			"EmptyPath",
			":contact_id/edit",
			"",
			[]string{},
			0,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", tt.path, nil)
			lastIdx, ok := parseParams([]byte(tt.wcPath), toBytes(req.URL.Path), req)
			if ok != tt.ok {
				t.Errorf("expected: [%v]; got [%v]", tt.ok, ok)
			}

			if lastIdx != tt.lastIdx {
				t.Errorf("expected: [%d]; got [%d]", tt.lastIdx, lastIdx)
			}

			// Check path value gets set correctly.
			for _, v := range tt.params {
				pv := req.PathValue(v)
				t.Log("path value:", pv)
				if pv == "" {
					t.Errorf("expected path value [%s] to be set", v)
				}
			}
		})
	}
}

func Benchmark_ParseParams(b *testing.B) {
	tests := []struct {
		name    string
		wcPath  string
		path    string
		params  []string
		lastIdx int
		ok      bool
	}{
		{
			"Parse",
			"/path/:with/:param",
			"/path/sub/subsub",
			[]string{
				"with",
				"param",
			},
			15,
			true,
		},
		{
			"ParseMismatchedSegments",
			"/user/group/:group_id",
			"/user/group",
			[]string{},
			11,
			false,
		},
		{
			"ParseMismatchedNoLeadingSlash",
			":path",
			"foo/bar",
			[]string{},
			3,
			false,
		},
		{
			"ParseMatchNoLeadingSlash",
			"foo/:bar",
			"foo/bar",
			[]string{"bar"},
			7,
			true,
		},
		{
			"ParseWithTrailingSlash",
			":bar/",
			"foo/",
			[]string{"bar"},
			4,
			true,
		},
		{
			"ParseMatchPartialMiddle",
			"/path/:bar/b",
			"/path/s/baz",
			[]string{"bar"},
			9,
			true,
		},
		{
			"ParseWildcardShort",
			"/path/*wildcard",
			"/path/single",
			[]string{"wildcard"},
			11,
			true,
		},
		{
			"ParseWildcardLong",
			"/path/*wildcard",
			"/path" + strings.Repeat("/path", 80),
			[]string{"wildcard"},
			404,
			true,
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			req, _ := http.NewRequest("GET", tt.path, nil)
			for i := 0; i < b.N; i++ {

				lastIdx, ok := parseParams(toBytes(tt.wcPath), toBytes(req.URL.Path), req)
				if ok != tt.ok {
					b.Errorf("expected: [%v]; got [%v]", tt.ok, ok)
				}

				if lastIdx != tt.lastIdx {
					b.Errorf("expected: [%d]; got [%d]", tt.lastIdx, lastIdx)
				}

				// Check path value gets set correctly.
				for _, v := range tt.params {
					if req.PathValue(v) == "" {
						b.Errorf("expected path value [%s] to be set", v)
					}
				}
			}
		})
	}
}

func Test_CountParams(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		total int
	}{
		{
			"CountFew",
			"/a/:b/:c/:d",
			3,
		},
		{
			"CountMany",
			strings.Repeat("/:path", 128),
			128,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if count := countParams([]byte(tt.path)); count != tt.total {
				t.Errorf("expected: [%d]; got [%d]", tt.total, count)
			}
		})
	}
}

func Test_Tree(t *testing.T) {
	tree := &node{}

	// preload with some routes.
	tree.insert([]byte("/"), emptyHandler)
	tree.insert([]byte("/home/:sub/:path"), emptyHandler)
	tree.insert([]byte("/lib/books/:book"), emptyHandler)
	tree.insert([]byte("/users/add/:user_id"), emptyHandler)
	tree.insert([]byte("/:path"), emptyHandler)
	tree.insert([]byte("/users/:id/app"), emptyHandler)
	tree.insert([]byte("/users/:id/settings"), emptyHandler)
	tree.insert([]byte("/users/:id/show"), emptyHandler)

	// Inserts

	insertTests := []struct {
		name      string
		path      string
		shouldErr bool
	}{
		{"Exact1", "/home/:sub/:path", true},
		{"Exact2", "/lib/books/:book", true},
		{"Exact3", "/users/add/:user_id", true},
		{"Exact4", "/:path", true},
		{"MatchingParam1", "/home/user/:path", false},
		{"MatchingParam2", "/home/print/:path", false},
		{"MatchingParam3", "/:path/todo", false},
		{"MatchingParam4", "/:path/", false},
		{"MismatchedParam1", "/home/:sub/:user_id", true},
		{"MismatchedParam2", "/lib/books/:wrong", true},
		{"MismatchedParam3", "/users/add/:group_id", true},
		{"MismatchedParam4", "/:root", true},
		{"NewPathNoParam", "/new/path", false},
		{"NewPathWithParam", "/new/path/:param", false},
		{"NewPathWithWildCard", "/new/wc/*wildcard", false},
		{"BadParamName1", "/bad/:pa:ram", true},
		{"BadParamName2", "/bad/:ram:", true},
		{"BadParamName3", "/bad/:", true},
		{"BadParamName4", "/bad/:asdf*", true},
		{"BadWildCard1", "/*path/bar", true},
		{"BadWildCard2", "/*", true},
		{"BadWildCard3", "/path/*ff*", true},
	}

	for _, tt := range insertTests {
		t.Run(fmt.Sprintf("Insert-%s", tt.name), func(t *testing.T) {
			defer func() {
				r := recover()
				if r != nil && !tt.shouldErr {
					t.Errorf("unexpected panic:\n%v\n", r)
				}

				if r == nil && tt.shouldErr {
					t.Errorf("no panic when one was expected.")
				}
			}()

			tree.insert([]byte(tt.path), emptyHandler)
		})
	}

	// Searches

	searchTests := []struct {
		name   string
		path   string
		params []string
		found  bool
	}{
		{
			"MatchHomeWildcard",
			"/home/catch/wildcard",
			[]string{
				"sub",
				"path",
			},
			true,
		},
		{
			"MissingHomeWildcardFinalPath",
			"/home/catch",
			[]string{},
			false,
		},
		{
			"MatchPathParamHomeUser",
			"/home/user/value",
			[]string{
				"path",
			},
			true,
		},
		{
			"MatchParamIdStaticRemainder1",
			"/users/1/settings",
			[]string{
				"id",
			},
			true,
		},
		{
			"MatchParamIdStaticRemainder2",
			"/users/1/app",
			[]string{
				"id",
			},
			true,
		},
		{
			"MatchRootParam",
			"/path1",
			[]string{
				"path",
			},
			true,
		},
		{
			"UnregisteredPath",
			"/foo/bar",
			[]string{},
			false,
		},
		{
			"MatchWildcard",
			"/new/wc/path",
			[]string{
				"wildcard",
			},
			true,
		},
	}

	for _, tt := range searchTests {
		t.Run(fmt.Sprintf("Search-%s", tt.name), func(t *testing.T) {
			req := &http.Request{}
			if _, ok := tree.search([]byte(tt.path), req); ok != tt.found {
				t.Errorf("expected: [%v]; got: [%v]", tt.found, ok)
			}

			// Check path value gets set correctly.
			for _, v := range tt.params {
				if pv := req.PathValue(v); pv == "" {
					t.Errorf("expected path value [%s] to be set", v)
				}
			}
		})
	}

	// ----------------------------------------------------------------------
	// Edge cases

	singleRouteTests := []struct {
		name   string
		wcPath string
		path   string
		ok     bool
	}{
		{
			"ParamsShortPath",
			"/foo/:bar",
			"/foo/bar",
			true,
		},
		{
			"ParamsLongPath",
			"/foo/:bar/:baz/:qux/:quux/:corge/:grault/:garply/:waldo/:fred/:plugh",
			"/foo/:bar/:baz/:qux/:quux/:corge/:grault/:garply/:waldo/:fred/:plugh",
			true,
		},
	}

	for _, tt := range singleRouteTests {
		t.Run(fmt.Sprintf("SingleRoute-%s", tt.name), func(t *testing.T) {
			tree := &node{}
			tree.insert([]byte(tt.wcPath), emptyHandler)
			if _, ok := tree.search([]byte(tt.path), &http.Request{}); ok != tt.ok {
				t.Errorf("expected: [%v]; got [%v]", tt.ok, ok)
				tree.print(0)
			}
		})
	}

	tree.print(0)
}
