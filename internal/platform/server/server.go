package server

import (
	"context"
	"errors"
	"fmt"
	"net"

	greeterpb "github.com/ogurasousui/codex-grpc-clean-arch/internal/adapters/grpc/gen/greeter/v1"
	"github.com/ogurasousui/codex-grpc-clean-arch/internal/adapters/grpc/handler"
	"github.com/ogurasousui/codex-grpc-clean-arch/internal/core/hello"
	"google.golang.org/grpc"
)

// Server は gRPC サーバーのライフサイクルを管理します。
type Server struct {
	listenAddr string
	grpcServer *grpc.Server
}

// New は指定されたアドレスで待ち受ける gRPC サーバーを構築します。
func New(listenAddr string, greeter hello.Greeter, opts ...grpc.ServerOption) *Server {
	srv := grpc.NewServer(opts...)
	greeterHandler := handler.NewGreeterHandler(greeter)
	greeterpb.RegisterGreeterServiceServer(srv, greeterHandler)

	return &Server{
		listenAddr: listenAddr,
		grpcServer: srv,
	}
}

// Run はサーバーを起動し、コンテキストがキャンセルされると GracefulStop します。
func (s *Server) Run(ctx context.Context) error {
	lis, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.listenAddr, err)
	}

	go func() {
		<-ctx.Done()
		s.grpcServer.GracefulStop()
	}()

	if err := s.grpcServer.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		return fmt.Errorf("serve gRPC: %w", err)
	}

	return nil
}

// GracefulStop はサーバーを安全に停止します。
func (s *Server) GracefulStop() {
	s.grpcServer.GracefulStop()
}
