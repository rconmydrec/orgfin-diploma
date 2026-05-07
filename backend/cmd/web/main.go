package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-budget/backend/internal/config"
	"github.com/go-budget/backend/internal/database"
	"github.com/go-budget/backend/internal/logger"
	"github.com/go-budget/backend/internal/server"
	"github.com/go-budget/backend/internal/workers"
)

func main() {
	cfg := config.New()
	log := logger.New(cfg.LogLevel)

	log.Info("starting go-budget server",
		"port", cfg.Port,
		"environment", cfg.Environment,
	)

	db, err := database.New(cfg.DatabaseURL, log)
	if err != nil {
		log.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Initialize async task manager.
	// Dependencies will be set by server.Start() before workerManager.Start() is called.
	workerManager, err := workers.NewManager(cfg, log, &workers.Dependencies{})
	if err != nil {
		log.Error("failed to create worker manager", "error", err)
		os.Exit(1)
	}

	srv, err := server.New(cfg, log, db, workerManager)
	if err != nil {
		log.Error("failed to create server", "error", err)
		os.Exit(1)
	}

	// Start() creates services and sets worker dependencies via workerManager.SetDependencies.
	httpServer := srv.Start()

	// Start workers after server setup so that dependencies are available for task handlers.
	if err := workerManager.Start(); err != nil {
		log.Error("failed to start worker manager", "error", err)
		os.Exit(1)
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := httpServer.ListenAndServe(); err != nil {
			log.Info("server stopped", "error", err)
		}
	}()

	log.Info("server started", "address", httpServer.Addr)

	<-quit
	log.Info("shutting down server...")

	// Stop workers BEFORE HTTP server to allow in-flight tasks to complete
	workerManager.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}

	log.Info("server exited gracefully")
}
