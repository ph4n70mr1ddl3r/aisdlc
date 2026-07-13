// Command metadata-api serves the dictionary CRUD (Layers 0–3) over HTTP.
// It runs embedded migrations on boot, then exposes /v1/{resource}.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	metadataapi "github.com/ph4n70mr1ddl3r/aisdlc/services/core/metadata-api"
	"github.com/ph4n70mr1ddl3r/aisdlc/services/core/metadata-api/internal/api"
	"github.com/ph4n70mr1ddl3r/aisdlc/services/core/metadata-api/internal/config"
	"github.com/ph4n70mr1ddl3r/aisdlc/services/core/metadata-api/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("metadata-api: config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("metadata-api: db connect: %v", err)
	}
	defer pool.Close()

	if _, err := pool.Exec(ctx, "SELECT 1"); err != nil {
		log.Fatalf("metadata-api: db unreachable: %v", err)
	}
	if err := metadataapi.Migrate(ctx, pool); err != nil {
		log.Fatalf("metadata-api: migrate: %v", err)
	}
	log.Printf("metadata-api: migrations applied")

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           api.NewRouter(store.New(pool)),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
	}

	go func() {
		log.Printf("metadata-api listening on %s", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("metadata-api: listen: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("metadata-api: shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("metadata-api: shutdown: %v", err)
	}
}
