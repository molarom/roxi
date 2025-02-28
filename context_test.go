package roxi

import (
	"context"
	"testing"
)

func Test_ContextValue(t *testing.T) {
	type testKey int

	ctx := context.WithValue(context.Background(), testKey(1), "test")

	ctx = &writerContext{ctx, writerKey, newMockResponseWriter()}

	v, ok := ctx.Value(testKey(1)).(string)
	if !ok {
		t.Error("failed to fallback to initial context")
	}

	if v != "test" {
		t.Errorf("expected: [%s]; got: [%s]", "test", v)
	}
}

func Test_ContextNilWriter(t *testing.T) {
	ctx := &writerContext{context.Background(), writerKey, nil}

	if w := GetWriter(ctx); w != nil {
		t.Errorf("unknown value returned from context: %v", w)
	}
}
