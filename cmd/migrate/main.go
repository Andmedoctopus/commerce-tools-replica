package main

import (
	"context"
	"log"
	"os"

	"commercetools-replica/internal/config"
	"commercetools-replica/internal/db"
	"commercetools-replica/internal/migrate"
)

func main() {
	cfg := config.FromEnv()
	logger := log.New(os.Stdout, "[migrate] ", log.LstdFlags|log.LUTC|log.Lshortfile)

	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DBConnString)
	if err != nil {
		logger.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	if err := migrate.Apply(ctx, pool); err != nil {
		logger.Fatalf("apply migrations: %v", err)
	}

	logger.Println("migrations applied")
}
