package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ogurasousui/codex-grpc-clean-arch/internal/core/hello"
	"github.com/ogurasousui/codex-grpc-clean-arch/internal/platform/server"
)

const defaultListenAddr = ":50051"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	greeterSvc := hello.NewService()
	grpcServer := server.New(defaultListenAddr, greeterSvc)

	log.Printf("gRPC server listening on %s", defaultListenAddr)

	if err := grpcServer.Run(ctx); err != nil {
		log.Fatalf("server stopped with error: %v", err)
	}
}
