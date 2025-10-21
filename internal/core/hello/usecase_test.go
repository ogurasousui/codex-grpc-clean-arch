package hello

import (
	"context"
	"testing"
)

func TestServiceSayHello(t *testing.T) {
	t.Parallel()

	svc := NewService()
	msg, err := svc.SayHello(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if msg != "" {
		t.Fatalf("expected empty message, got %q", msg)
	}
}
