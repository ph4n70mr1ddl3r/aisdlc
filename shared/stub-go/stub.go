// Package stub provides a shared M0 health-and-info HTTP server that all
// service stubs delegate to instead of duplicating the same 60-line main.go.
// Replaced by the real implementation in each service's milestone (ROADMAP.md).
package stub

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Run starts a stub HTTP server on the given port with /healthz and / endpoints.
// Blocks until SIGINT/SIGTERM, then performs a graceful 5-second shutdown.
func Run(port string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
			log.Printf("stub: /healthz encode: %v", err)
		}
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		svc := os.Getenv("OTEL_SERVICE_NAME")
		if err := json.NewEncoder(w).Encode(map[string]string{"service": svc}); err != nil {
			log.Printf("stub: / encode: %v", err)
		}
	})

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("stub listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("stub: listen: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("stub: shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("stub: shutdown: %v", err)
	}
}
