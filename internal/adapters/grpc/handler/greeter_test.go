package handler

import (
	"context"
	"testing"

	greeterpb "github.com/ogurasousui/codex-grpc-clean-arch/internal/adapters/grpc/gen/greeter/v1"
)

type stubGreeter struct{}

func (stubGreeter) SayHello(ctx context.Context) (string, error) {
	return "", nil
}

func TestGreeterHandler_SayHello(t *testing.T) {
	t.Parallel()

	handler := NewGreeterHandler(stubGreeter{})

	resp, err := handler.SayHello(context.Background(), &greeterpb.SayHelloRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Message != "" {
		t.Fatalf("expected empty message, got %q", resp.Message)
	}

	if _, ok := interface{}(resp).(*greeterpb.SayHelloResponse); !ok {
		t.Fatalf("response should be greeterpb.SayHelloResponse")
	}
}
