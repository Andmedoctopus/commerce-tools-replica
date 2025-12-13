package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"commercetools-replica/internal/config"
	"commercetools-replica/internal/db"
	"commercetools-replica/internal/domain"
	"commercetools-replica/internal/importer"
	"commercetools-replica/internal/repository/product"
	"commercetools-replica/internal/repository/project"
)

func main() {
	var (
		filePath   string
		projectKey string
	)
	flag.StringVar(&filePath, "file", "", "Path to commercetools product CSV export")
	flag.StringVar(&projectKey, "project", "", "Project key to import into")
	flag.Parse()

	if filePath == "" || projectKey == "" {
		flag.Usage()
		os.Exit(2)
	}

	cfg := config.FromEnv()
	ctx := context.Background()

	pool, err := db.Connect(ctx, cfg.DBConnString)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	projRepo := project.NewPostgres(pool)
	proj, err := projRepo.GetByKey(ctx, projectKey)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			proj, err = ensureProject(ctx, projRepo, projectKey)
		}
		if err != nil {
			log.Fatalf("ensure project %q: %v", projectKey, err)
		}
	}

	f, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("open file: %v", err)
	}
	defer f.Close()

	imp := importer.NewCSVImporter(f, product.NewPostgres(pool), proj.ID)

	start := time.Now()
	count, err := imp.Run(ctx)
	if err != nil {
		log.Fatalf("import failed: %v", err)
	}

	fmt.Printf("Imported %d products into project %s in %s\n", count, projectKey, time.Since(start).Truncate(time.Millisecond))
}

func ensureProject(ctx context.Context, repo project.Repository, key string) (*domain.Project, error) {
	p := &domain.Project{
		Key:  key,
		Name: key,
	}
	created, err := repo.Create(ctx, p)
	if err != nil {
		return nil, err
	}
	return created, nil
}
