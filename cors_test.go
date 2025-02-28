package roxi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func Test_DefaultCORS(t *testing.T) {
	r, _ := http.NewRequest("OPTIONS", "/", nil)
	w := httptest.NewRecorder()

	DefaultCORS.ServeHTTP(w, r)

	h := w.Header().Get("Access-Control-Allow-Origin")
	exp := "*"
	if h != exp {
		t.Errorf("expected origin header: [%q]; got [%q]", exp, h)
	}

	h = w.Header().Get("Access-Control-Allow-Methods")
	exp = strings.Join(defaultCORS.Methods, ", ")
	if h != exp {
		t.Errorf("expected methods header: [%q]; got [%q]", exp, h)
	}

	h = w.Header().Get("Access-Control-Allow-Headers")
	exp = strings.Join(defaultCORS.Headers, ", ")
	if h != exp {
		t.Errorf("expected headers header: [%q]; got [%q]", exp, h)
	}

	h = w.Header().Get("Access-Control-Max-Age")
	exp = "86400"
	if h != exp {
		t.Errorf("expected max-age header: [%q]; got [%q]", exp, h)
	}
}
