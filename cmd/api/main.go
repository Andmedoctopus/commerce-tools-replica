package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"commercetools-replica/internal/config"
	"commercetools-replica/internal/httpserver"
)

func main() {
	cfg := config.FromEnv()
	logger := log.New(os.Stdout, "[api] ", log.LstdFlags|log.LUTC|log.Lshortfile)

	srv := httpserver.New(cfg.HTTPAddr, logger)

	serverErr := make(chan error, 1)
	go func() {
		logger.Printf("starting http server on %s", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-stopCh:
		logger.Printf("received signal %s, shutting down", sig)
	case err := <-serverErr:
		logger.Printf("server error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Printf("graceful shutdown failed: %v", err)
	} else {
		logger.Printf("server stopped")
	}

	// TODO: wire DB connection and dependency injection when repositories/services are added.
}
