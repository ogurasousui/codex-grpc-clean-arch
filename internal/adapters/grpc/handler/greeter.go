package handler

import (
	"context"

	greeterpb "github.com/ogurasousui/codex-grpc-clean-arch/internal/adapters/grpc/gen/greeter/v1"
	"github.com/ogurasousui/codex-grpc-clean-arch/internal/core/hello"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// GreeterHandler は gRPC 層からユースケースを呼び出すアダプタです。
type GreeterHandler struct {
	greeter hello.Greeter
	greeterpb.UnimplementedGreeterServiceServer
}

// NewGreeterHandler は GreeterHandler を生成します。
func NewGreeterHandler(g hello.Greeter) *GreeterHandler {
	return &GreeterHandler{greeter: g}
}

// SayHello はユースケースを呼び出し、空文字列メッセージを含むレスポンスを返します。
func (h *GreeterHandler) SayHello(ctx context.Context, _ *emptypb.Empty) (*greeterpb.SimpleResponse, error) {
	message, err := h.greeter.SayHello(ctx)
	if err != nil {
		return nil, err
	}
	return &greeterpb.SimpleResponse{Message: message}, nil
}
