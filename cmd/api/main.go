package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/api"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/db"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func main() {
	cfg := config.Load()
	logger, err := zap.NewProduction()
	if cfg.AppEnv == "development" {
		logger, err = zap.NewDevelopment()
	}
	if err != nil {
		panic(err)
	}
	defer logger.Sync() //nolint:errcheck
	if err := cfg.Validate(); err != nil {
		logger.Fatal("invalid service configuration", zap.Error(err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	backendStore := store.New()
	if cfg.EnableDemoData && !cfg.IsProduction() {
		backendStore = store.NewSeeded()
	}
	var entClient *ent.Client
	var closeDB func() error
	storageMode := "in_memory"
	if cfg.DatabaseURL != "" {
		client, closer, err := db.Open(cfg.DatabaseURL)
		if err != nil {
			if cfg.IsProduction() {
				logger.Fatal("database open failed", zap.Error(err))
			}
			logger.Warn("database open failed; using in-memory store", zap.Error(err))
		} else if err := db.Migrate(ctx, client); err != nil {
			if cfg.IsProduction() {
				logger.Fatal("database migration failed", zap.Error(err))
			}
			logger.Warn("database migration failed; using in-memory store", zap.Error(err))
			_ = closer()
		} else if cfg.EnableDemoData && !cfg.IsProduction() {
			if err := db.Seed(ctx, client); err != nil {
				logger.Warn("database demo seed failed; using in-memory store", zap.Error(err))
				_ = closer()
			} else {
				loaded, err := db.LoadStore(ctx, client)
				if err != nil {
					logger.Warn("database load failed; using in-memory store", zap.Error(err))
					_ = closer()
				} else {
					backendStore = loaded
					entClient = client
					closeDB = closer
					storageMode = "postgres"
				}
			}
		} else if loaded, err := db.LoadStore(ctx, client); err != nil {
			if cfg.IsProduction() {
				logger.Fatal("database load failed", zap.Error(err))
			}
			logger.Warn("database load failed; using in-memory store", zap.Error(err))
			_ = closer()
		} else {
			backendStore = loaded
			entClient = client
			closeDB = closer
			storageMode = "postgres"
		}
	}
	if closeDB != nil {
		defer func() {
			if err := closeDB(); err != nil {
				logger.Warn("database close failed", zap.Error(err))
			}
		}()
	}

	router := api.New(cfg, backendStore, logger, entClient)
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("starting gamblock-ai backend", zap.String("addr", cfg.HTTPAddr), zap.String("storage_mode", storageMode))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("server failed", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown failed", zap.Error(err))
	}
}
