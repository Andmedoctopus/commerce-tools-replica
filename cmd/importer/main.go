package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"commercetools-replica/internal/config"
	"commercetools-replica/internal/db"
	"commercetools-replica/internal/domain"
	"commercetools-replica/internal/importer"
	"commercetools-replica/internal/migrate"
	category "commercetools-replica/internal/repository/category"
	"commercetools-replica/internal/repository/product"
	"commercetools-replica/internal/repository/project"
)

func main() {
	var (
		inputPath  string
		projectKey string
	)
	flag.StringVar(&inputPath, "path", "", "Path to CSV file or directory containing commercetools exports (defaults to imports/<project>)")
	flag.StringVar(&inputPath, "file", "", "Path to commercetools CSV export (deprecated, same as -path)")
	flag.StringVar(&projectKey, "project", "", "Project key to import into")
	flag.Parse()

	if projectKey == "" {
		flag.Usage()
		os.Exit(2)
	}
	if inputPath == "" {
		inputPath = filepath.Join("imports", projectKey)
	}

	cfg := config.FromEnv()
	ctx := context.Background()

	logger := log.New(os.Stdout, "[importer] ", log.LstdFlags|log.LUTC|log.Lshortfile)

	pool, err := db.Connect(ctx, cfg.DBConnString)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	if err := migrate.Apply(ctx, pool); err != nil {
		log.Fatalf("apply migrations: %v", err)
	}

	projRepo := project.NewPostgres(pool, logger)
	proj, err := projRepo.GetByKey(ctx, projectKey)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			proj, err = ensureProject(ctx, projRepo, projectKey)
		}
		if err != nil {
			log.Fatalf("ensure project %q: %v", projectKey, err)
		}
	}

	info, err := os.Stat(inputPath)
	if err != nil {
		log.Fatalf("stat path %q: %v", inputPath, err)
	}

	productRepo := product.NewPostgres(pool, logger)
	categoryRepo := category.NewPostgres(pool)

	if info.IsDir() {
		if err := importDirectory(ctx, inputPath, projectKey, proj.ID, productRepo, categoryRepo); err != nil {
			log.Fatalf("import directory: %v", err)
		}
		return
	}

	count, kind, dur, err := importFile(ctx, inputPath, proj.ID, productRepo, categoryRepo)
	if err != nil {
		log.Fatalf("import failed: %v", err)
	}

	fmt.Printf("Imported %d %s into project %s in %s\n", count, kind, projectKey, dur.Truncate(time.Millisecond))
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

func importDirectory(ctx context.Context, dir, projectKey, projectID string, productRepo importer.ProductWriter, categoryRepo importer.CategoryWriter) error {
	catFiles, productFiles, err := discoverFiles(dir)
	if err != nil {
		return err
	}

	ordered := append(catFiles, productFiles...) // categories first to establish hierarchy
	if len(ordered) == 0 {
		return fmt.Errorf("no CSV exports found in %s", dir)
	}

	start := time.Now()
	total := 0
	for _, path := range ordered {
		count, kind, dur, err := importFile(ctx, path, projectID, productRepo, categoryRepo)
		if err != nil {
			return fmt.Errorf("import %s: %w", path, err)
		}
		total += count
		fmt.Printf("Imported %d %s from %s in %s\n", count, kind, path, dur.Truncate(time.Millisecond))
	}

	fmt.Printf("Imported %d records from %d files into project %s in %s\n", total, len(ordered), projectKey, time.Since(start).Truncate(time.Millisecond))
	return nil
}

func importFile(ctx context.Context, path, projectID string, productRepo importer.ProductWriter, categoryRepo importer.CategoryWriter) (int, string, time.Duration, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, "", 0, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	imp := importer.NewCSVImporter(f, productRepo, categoryRepo, projectID)

	start := time.Now()
	count, err := imp.Run(ctx)
	if err != nil {
		return count, imp.Kind(), 0, err
	}

	return count, imp.Kind(), time.Since(start), nil
}

func discoverFiles(dir string) (categories []string, products []string, err error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("read dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".csv" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		kind, err := detectFileKind(path)
		if err != nil {
			return nil, nil, err
		}
		switch kind {
		case importer.KindCategories:
			categories = append(categories, path)
		default:
			products = append(products, path)
		}
	}

	sort.Strings(categories)
	sort.Strings(products)
	return categories, products, nil
}

func detectFileKind(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	kind, err := importer.DetectKind(f)
	if err != nil {
		return "", fmt.Errorf("detect kind for %s: %w", path, err)
	}
	return kind, nil
}
