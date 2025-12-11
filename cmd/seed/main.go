package main

import (
	"context"
	"log"
	"os"

	"commercetools-replica/internal/config"
	"commercetools-replica/internal/db"
	"commercetools-replica/internal/seed"
)

func main() {
	cfg := config.FromEnv()
	logger := log.New(os.Stdout, "[seed] ", log.LstdFlags|log.LUTC|log.Lshortfile)

	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DBConnString)
	if err != nil {
		logger.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	if err := seed.Apply(ctx, pool); err != nil {
		logger.Fatalf("seed apply: %v", err)
	}

	logger.Println("seed applied")
}
