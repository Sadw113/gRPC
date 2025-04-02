package main

import (
	"context"
	sso "gRPC/gRPC/genGo"
	"gRPC/internal/config"
	"gRPC/internal/repo"
	"gRPC/internal/service"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	customLogger "gRPC/pkg/logger"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Printf(".env file doesn't exist or can't read .env")
	}

	var cfg config.AppConfig
	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatal(errors.Wrap(err, "failed to load configuration"))
	}

	logger, err := customLogger.NewLogger(cfg.LogLevel)
	if err != nil {
		log.Fatal(errors.Wrap(err, "error initializing logger"))
	}

	repository, err := repo.NewRepository(context.Background(), cfg.PostgreSQL)
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to initialize repository"))
	}

	serviceInstance := service.NewService(repository, logger, cfg.SecretKeys)

	lis, err := net.Listen("tcp", cfg.GRPC.ListenAddress)

	if err != nil {
		log.Fatal(errors.Wrap(err, "failed listening tcp port"))
	}

	gPRCServer := grpc.NewServer()

	sso.RegisterAuthServiceServer(gPRCServer, serviceInstance)

	go func() {
		logger.Infof("Starting server on %s", cfg.GRPC.ListenAddress)
		if err := gPRCServer.Serve(lis); err != nil {
			log.Fatal(errors.Wrap(err, "failed to start server"))
		}

	}()

	// Ожидание системных сигналов для корректного завершения работы
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	<-signalChan

	logger.Info("Shutting down gracefully...")
}
