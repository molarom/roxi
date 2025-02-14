// Copyright 2013 Julien Schmidt. All rights reserved.
// Based on the path package, Copyright 2009 The Go Authors.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

package roxi

import (
	"bytes"
	"strings"
	"testing"
)

type cleanPathTest struct {
	path, result string
	buf          []byte
}

var cleanTests = []cleanPathTest{
	// Already clean
	{"/", "/", []byte("/")},
	{"/abc", "/abc", []byte("/abc")},
	{"/a/b/c", "/a/b/c", []byte("/a/b/c")},
	{"/abc/", "/abc/", []byte("/abc/")},
	{"/a/b/c/", "/a/b/c/", []byte("/a/b/c/")},

	// missing root
	{"", "/", []byte("/")},
	{"a/", "/a/", []byte("/a/")},
	{"abc", "/abc", []byte("/abc")},
	{"abc/def", "/abc/def", []byte("/abc/def")},
	{"a/b/c", "/a/b/c", []byte("/a/b/c")},

	// Remove doubled slash
	{"//", "/", []byte("/")},
	{"/abc//", "/abc/", []byte("/abc/")},
	{"/abc/def//", "/abc/def/", []byte("/abc/def/")},
	{"/a/b/c//", "/a/b/c/", []byte("/a/b/c/")},
	{"/abc//def//ghi", "/abc/def/ghi", []byte("/abc/def/ghi")},
	{"//abc", "/abc", []byte("/abc")},
	{"///abc", "/abc", []byte("/abc")},
	{"//abc//", "/abc/", []byte("/abc/")},

	// Remove . elements
	{".", "/", []byte("/")},
	{"./", "/", []byte("/")},
	{"/abc/./def", "/abc/def", []byte("/abc/def")},
	{"/./abc/def", "/abc/def", []byte("/abc/def")},
	{"/abc/.", "/abc/", []byte("/abc/")},

	// Remove .. elements
	{"..", "/", []byte("/")},
	{"../", "/", []byte("/")},
	{"../../", "/", []byte("/")},
	{"../..", "/", []byte("/")},
	{"../../abc", "/abc", []byte("/abc")},
	{"/abc/def/ghi/../jkl", "/abc/def/jkl", []byte("/abc/def/jkl")},
	{"/abc/def/../ghi/../jkl", "/abc/jkl", []byte("/abc/jkl")},
	{"/abc/def/..", "/abc", []byte("/abc")},
	{"/abc/def/../..", "/", []byte("/")},
	{"/abc/def/../../..", "/", []byte("/")},
	{"/abc/def/../../..", "/", []byte("/")},
	{"/abc/def/../../../ghi/jkl/../../../mno", "/mno", []byte("/mno")},

	// Combinations
	{"abc/./../def", "/def", []byte("/def")},
	{"abc//./../def", "/def", []byte("/def")},
	{"abc/../../././../def", "/def", []byte("/def")},
}

func TestPathClean(t *testing.T) {
	for _, test := range cleanTests {
		if s := CleanPath(test.path); !bytes.Equal(s, test.buf) {
			t.Errorf("CleanPath(%q) = %q, want %q", test.path, s, test.result)
		}
		if s := CleanPath(test.result); !bytes.Equal(s, test.buf) {
			t.Errorf("CleanPath(%q) = %q, want %q", test.result, s, test.result)
		}
	}
}

func TestPathCleanMallocs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping malloc count in short mode")
	}

	for _, test := range cleanTests {
		test := test
		allocs := testing.AllocsPerRun(100, func() { CleanPath(test.result) })
		if allocs > 0 {
			t.Errorf("CleanPath(%q): %v allocs, want zero", string(test.result), allocs)
		}
	}
}

func BenchmarkPathClean(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, test := range cleanTests {
			CleanPath(test.path)
		}
	}
}

func genLongPaths() (testPaths []cleanPathTest) {
	for i := 1; i <= 1234; i++ {
		ss := strings.Repeat("a", i)

		correctPath := "/" + ss
		testPaths = append(testPaths, cleanPathTest{
			path:   correctPath,
			result: correctPath,
			buf:    []byte(correctPath),
		}, cleanPathTest{
			path:   ss,
			result: correctPath,
			buf:    []byte(correctPath),
		}, cleanPathTest{
			path:   "//" + ss,
			result: correctPath,
			buf:    []byte(correctPath),
		}, cleanPathTest{
			path:   "/" + ss + "/b/..",
			result: correctPath,
			buf:    []byte(correctPath),
		})
	}
	return testPaths
}

func TestPathCleanLong(t *testing.T) {
	cleanTests := genLongPaths()

	for _, test := range cleanTests {
		if s := CleanPath(test.path); !bytes.Equal(s, test.buf) {
			t.Errorf("CleanPath(%q) = %q, want %q", string(test.path), s, string(test.result))
		}
		if s := CleanPath(test.result); !bytes.Equal(s, test.buf) {
			t.Errorf("CleanPath(%q) = %q, want %q", string(test.result), s, string(test.result))
		}
	}
}

func BenchmarkPathCleanLong(b *testing.B) {
	cleanTests := genLongPaths()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, test := range cleanTests {
			CleanPath(test.path)
		}
	}
}
