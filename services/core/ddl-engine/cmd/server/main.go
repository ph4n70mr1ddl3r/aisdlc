// Command ddl-engine serves the metadata→DDL reconciler. On boot it connects
// to the metadata DB (read entities/fields/indexes) and the tenant data DB
// (apply DDL), ensures its migrations ledger exists, then exposes /v1 routes.
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

	"github.com/jackc/pgx/v5/pgxpool"

	ddlengine "github.com/ph4n70mr1ddl3r/aisdlc/services/core/ddl-engine"
	"github.com/ph4n70mr1ddl3r/aisdlc/services/core/ddl-engine/internal/api"
	"github.com/ph4n70mr1ddl3r/aisdlc/services/core/ddl-engine/internal/config"
	"github.com/ph4n70mr1ddl3r/aisdlc/services/core/ddl-engine/internal/ddl"
	"github.com/ph4n70mr1ddl3r/aisdlc/services/core/ddl-engine/internal/meta"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("ddl-engine: config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	metaPool, err := pgxpool.New(ctx, cfg.MetaDB)
	if err != nil {
		log.Fatalf("ddl-engine: meta db connect: %v", err)
	}
	defer metaPool.Close()

	dataPool, err := pgxpool.New(ctx, cfg.DataDB)
	if err != nil {
		log.Fatalf("ddl-engine: data db connect: %v", err)
	}
	defer dataPool.Close()

	if _, err := metaPool.Exec(ctx, "SELECT 1"); err != nil {
		log.Fatalf("ddl-engine: meta db unreachable: %v", err)
	}
	if _, err := dataPool.Exec(ctx, "SELECT 1"); err != nil {
		log.Fatalf("ddl-engine: data db unreachable: %v", err)
	}
	if err := ddlengine.EnsureSchema(ctx, dataPool); err != nil {
		log.Fatalf("ddl-engine: ensure schema: %v", err)
	}
	log.Printf("ddl-engine: schema ready")

	engine := ddl.New(meta.New(metaPool), dataPool)

	if cfg.ReconcileOnBoot {
		go func() {
			log.Printf("ddl-engine: RECONCILE_ON_BOOT set — applying all entities")
			entities, err := engine.Meta().AllEntities(ctx)
			if err != nil {
				log.Printf("ddl-engine: boot reconcile: list entities: %v", err)
				return
			}
			for _, e := range entities {
				res, err := engine.Apply(ctx, e.ID)
				if err != nil {
					log.Printf("ddl-engine: boot reconcile %s: %v", e.Name, err)
					continue
				}
				log.Printf("ddl-engine: boot reconcile %s: %d statements applied", e.Name, res.Applied)
			}
		}()
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           api.NewRouter(engine),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	go func() {
		log.Printf("ddl-engine listening on %s", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("ddl-engine: listen: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("ddl-engine: shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("ddl-engine: shutdown: %v", err)
	}
}
