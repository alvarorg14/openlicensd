package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alvarorg14/openlicensd/server/internal/api"
	"github.com/alvarorg14/openlicensd/server/internal/config"
	"github.com/alvarorg14/openlicensd/server/internal/logging"
	"github.com/alvarorg14/openlicensd/server/internal/maintenance"
	"github.com/alvarorg14/openlicensd/server/internal/ratelimit"
	"github.com/alvarorg14/openlicensd/server/internal/static"
	"github.com/alvarorg14/openlicensd/server/internal/store"
	"github.com/alvarorg14/openlicensd/server/internal/version"
)

func main() {
	bootstrapLogger := logging.NewDefault()
	slog.SetDefault(bootstrapLogger)

	cfg, err := config.Load()
	if err != nil {
		bootstrapLogger.Error("config load failed", slog.Any("err", err))
		os.Exit(1)
	}

	logger, err := logging.New(cfg.Log)
	if err != nil {
		bootstrapLogger.Error("logger init failed", slog.Any("err", err))
		os.Exit(1)
	}
	slog.SetDefault(logger)

	ctx := context.Background()
	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("store init failed", slog.Any("err", err))
		os.Exit(1)
	}
	defer st.Close()

	if err := store.BootstrapAdmin(ctx, st, cfg); err != nil {
		logger.Error("bootstrap admin failed", slog.Any("err", err))
		os.Exit(1)
	}

	bgCtx, stopBackground := context.WithCancel(context.Background())
	defer stopBackground()

	if interval := cfg.SessionCleanupInterval(); interval > 0 {
		go maintenance.NewSessionCleaner(st, interval, logger).Run(bgCtx)
		logger.Info("session cleanup enabled", slog.Duration("interval", interval))
	} else {
		logger.Info("session cleanup disabled")
	}

	srv, err := api.New(ctx, cfg, st, logger)
	if err != nil {
		logger.Error("api server init failed", slog.Any("err", err))
		os.Exit(1)
	}
	srv.StartBackground(bgCtx)
	ratelimit.LogStartup(logger, cfg.RateLimit)
	staticHandler := static.MustHandler()
	handler := srv.Router(staticHandler)

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("listening", slog.String("version", version.Version), slog.String("addr", cfg.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", slog.Any("err", err))
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	sig := <-stop
	logger.Info("shutdown signal received", slog.String("signal", sig.String()))

	stopBackground()
	logger.Info("background tasks stopped")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown failed", slog.Any("err", err))
		os.Exit(1)
	}
	logger.Info("shutdown complete")
}
