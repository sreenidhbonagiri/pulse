package metrics

import (
	"context"
	"log"
	"net/http"
	"time"
)

// StartServer exposes GET /metrics on addr. Failures are logged and ignored
// so observability cannot take down the API, worker, or scheduler.
func StartServer(ctx context.Context, addr string, m *Metrics) {
	if addr == "" || m == nil || m.Gatherer() == nil {
		return
	}

	mux := http.NewServeMux()
	mux.Handle("GET /metrics", MetricsHandler(m))

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	go func() {
		log.Printf("metrics listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("metrics server unavailable (Pulse will keep running): %v", err)
		}
	}()
}
