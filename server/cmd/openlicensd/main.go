package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/openlicensd/openlicensd/server/internal/api"
	"github.com/openlicensd/openlicensd/server/internal/config"
	"github.com/openlicensd/openlicensd/server/internal/maintenance"
	"github.com/openlicensd/openlicensd/server/internal/ratelimit"
	"github.com/openlicensd/openlicensd/server/internal/static"
	"github.com/openlicensd/openlicensd/server/internal/store"
	"github.com/openlicensd/openlicensd/server/internal/version"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer st.Close()

	if err := store.BootstrapAdmin(ctx, st, cfg); err != nil {
		log.Fatalf("bootstrap: %v", err)
	}

	bgCtx, stopBackground := context.WithCancel(context.Background())
	defer stopBackground()

	if interval := cfg.SessionCleanupInterval(); interval > 0 {
		go maintenance.NewSessionCleaner(st, interval).Run(bgCtx)
		log.Printf("session cleanup every %s", interval)
	} else {
		log.Printf("session cleanup disabled")
	}

	srv := api.New(ctx, cfg, st)
	srv.StartBackground(bgCtx)
	ratelimit.LogStartup(cfg.RateLimit)
	staticHandler := static.MustHandler()
	handler := srv.Router(staticHandler)

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("openlicensd %s listening on %s", version.Version, cfg.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	stopBackground()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}
}
