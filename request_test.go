package roxi_test

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"

	"gitlab.com/romalor/roxi"
)

type mockBinder []byte

func (b *mockBinder) Bind(data []byte) error {
	*b = data
	return nil
}

func (b mockBinder) Validate() error {
	if b != nil {
		return nil
	}
	return fmt.Errorf("mockbinder cannot be nil")
}

func Test_Bind(t *testing.T) {
	buf := bytes.NewBufferString("test")
	r, _ := http.NewRequest("GET", "/", buf)

	b := &mockBinder{}
	if err := roxi.Bind(r, b); err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if !bytes.Equal(*b, []byte("test")) {
		t.Errorf("expected: [%s]; got [%s]", "test", string(*b))
	}
}

type errReader struct{}

func (e *errReader) Read([]byte) (int, error) {
	return 0, fmt.Errorf("read error")
}

func Test_BindReaderError(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", &errReader{})

	b := &mockBinder{}
	if err := roxi.Bind(r, b); err == nil {
		t.Errorf("failed to err on empty request body")
	}
}

type errBinder struct{}

func (e *errBinder) Bind([]byte) error {
	return fmt.Errorf("bind error")
}

func Test_BindError(t *testing.T) {
	buf := bytes.NewBufferString("")
	r, _ := http.NewRequest("GET", "/", buf)

	b := &errBinder{}
	if err := roxi.Bind(r, b); err == nil {
		t.Errorf("failed to catch bind err")
	}
}

type errValidator struct{}

func (e *errValidator) Bind([]byte) error {
	return nil
}

func (e *errValidator) Validate() error {
	return fmt.Errorf("validate error")
}

func Test_ValidateError(t *testing.T) {
	buf := bytes.NewBufferString("")
	r, _ := http.NewRequest("GET", "/", buf)

	b := &errValidator{}
	if err := roxi.Bind(r, b); err == nil {
		t.Errorf("failed to catch validate err")
	}
}
