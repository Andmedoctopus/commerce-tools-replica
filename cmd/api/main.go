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
	"commercetools-replica/internal/db"
	"commercetools-replica/internal/httpserver"
	cartrepo "commercetools-replica/internal/repository/cart"
	categoryrepo "commercetools-replica/internal/repository/category"
	customerrepo "commercetools-replica/internal/repository/customer"
	productrepo "commercetools-replica/internal/repository/product"
	projectrepo "commercetools-replica/internal/repository/project"
	anonymoussvc "commercetools-replica/internal/service/anonymous"
	cartsvc "commercetools-replica/internal/service/cart"
	categorysvc "commercetools-replica/internal/service/category"
	customersvc "commercetools-replica/internal/service/customer"
	productsvc "commercetools-replica/internal/service/product"
)

func main() {
	cfg := config.FromEnv()
	logger := log.New(os.Stdout, "[api] ", log.LstdFlags|log.LUTC|log.Lshortfile)

	ctx := context.Background()
	dbpool, err := db.Connect(ctx, cfg.DBConnString)
	if err != nil {
		logger.Fatalf("connect to db: %v", err)
	}
	defer dbpool.Close()

	projectRepo := projectrepo.NewPostgres(dbpool, logger)
	productRepo := productrepo.NewPostgres(dbpool, logger)
	productService := productsvc.New(productRepo)
	cartRepo := cartrepo.NewPostgres(dbpool)
	cartService := cartsvc.New(cartRepo, productRepo)
	categoryRepo := categoryrepo.NewPostgres(dbpool)
	categoryService := categorysvc.New(categoryRepo)
	customerRepo := customerrepo.NewPostgres(dbpool, logger)
	customerService := customersvc.New(customerRepo)
	anonymousService := anonymoussvc.New()

	srv, err := httpserver.New(cfg.HTTPAddr, logger, dbpool, httpserver.Deps{
		ProjectRepo:  projectRepo,
		ProductSvc:   productService,
		CartSvc:      cartService,
		CategorySvc:  categoryService,
		CustomerSvc:  customerService,
		AnonymousSvc: anonymousService,
	})
	if err != nil {
		logger.Fatalf("init server: %v", err)
	}

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
