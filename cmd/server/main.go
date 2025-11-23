package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zalhui/pull-request-service/internal/config"
	"github.com/zalhui/pull-request-service/internal/db"
	"github.com/zalhui/pull-request-service/internal/handler"
	"github.com/zalhui/pull-request-service/internal/logger"
	"github.com/zalhui/pull-request-service/internal/repository"
	"github.com/zalhui/pull-request-service/internal/service"
)

const (
	defaultTimeout         = 10 * time.Second
	defaultShutdownTimeout = 5 * time.Second
)

func main() {
	cfg := config.Load()

	zapLogger, err := logger.New(cfg.LogLevel)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer zapLogger.Sync()

	log := zapLogger.Sugar()
	log.Infow("Starting PR Reviewer Service",
		"port", cfg.ServerPort,
		"log_level", cfg.LogLevel)

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	dbPool, err := db.NewConnection(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	log.Infow("Database connection established",
		"host", cfg.DBHost,
		"port", cfg.DBPort,
		"database", cfg.DBName)

	repo := repository.NewPostgresRepository(log, dbPool)
	services := service.NewService(repo, log)
	handlers := handler.NewHandler(services, log)

	router := handlers.SetupRouter()

	server := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: router,
	}

	go func() {
		log.Infof("Server starting on :%s", cfg.ServerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	ctx, shutdownCancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
	defer shutdownCancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Errorf("Server forced to shutdown: %v", err)
	} else {
		log.Info("Server exited gracefully")
	}
}
