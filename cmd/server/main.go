package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ogurasousui/codex-grpc-clean-arch/internal/adapters/repository/postgres"
	"github.com/ogurasousui/codex-grpc-clean-arch/internal/core/hello"
	"github.com/ogurasousui/codex-grpc-clean-arch/internal/core/user"
	"github.com/ogurasousui/codex-grpc-clean-arch/internal/platform/config"
	pg "github.com/ogurasousui/codex-grpc-clean-arch/internal/platform/db/postgres"
	"github.com/ogurasousui/codex-grpc-clean-arch/internal/platform/server"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "assets/local.yaml"
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	dbPool, err := pg.NewPool(ctx, cfg.Database)
	if err != nil {
		log.Fatalf("failed to initialize database pool: %v", err)
	}
	defer dbPool.Close()

	greeterSvc := hello.NewService()
	userRepo := postgres.NewUserRepository(dbPool)
	userSvc := user.NewService(userRepo, nil)
	grpcServer := server.New(cfg.Server.ListenAddr, greeterSvc, userSvc)

	log.Printf("gRPC server listening on %s", cfg.Server.ListenAddr)

	if err := grpcServer.Run(ctx); err != nil {
		log.Fatalf("server stopped with error: %v", err)
	}
}
