package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
	mediaRoot := envOrDefault("MEDIA_ROOT", "media")
	mediaBaseURL := envOrDefault("MEDIA_BASE_URL", "media")
	archiveDir := archiveDirForInput(inputPath)

	if info.IsDir() {
		runner := importerRunner{
			projectKey:   projectKey,
			projectID:    proj.ID,
			productRepo:  productRepo,
			categoryRepo: categoryRepo,
			mediaRoot:    mediaRoot,
			mediaBaseURL: mediaBaseURL,
			archiveDir:   archiveDir,
		}
		if err := runner.importDirectory(ctx, inputPath); err != nil {
			log.Fatalf("import directory: %v", err)
		}
		return
	}

	runner := importerRunner{
		projectKey:   projectKey,
		projectID:    proj.ID,
		productRepo:  productRepo,
		categoryRepo: categoryRepo,
		mediaRoot:    mediaRoot,
		mediaBaseURL: mediaBaseURL,
		archiveDir:   archiveDir,
	}
	count, kind, dur, err := runner.importFile(ctx, inputPath)
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

type importerRunner struct {
	projectKey   string
	projectID    string
	productRepo  importer.ProductWriter
	categoryRepo importer.CategoryWriter
	mediaRoot    string
	mediaBaseURL string
	archiveDir   string
}

func (r importerRunner) importDirectory(ctx context.Context, dir string) error {
	if err := r.restoreArchive(ctx); err != nil {
		return err
	}
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
		count, kind, dur, err := r.importFileCore(ctx, path)
		if err != nil {
			return fmt.Errorf("import %s: %w", path, err)
		}
		total += count
		fmt.Printf("Imported %d %s from %s in %s\n", count, kind, path, dur.Truncate(time.Millisecond))
	}

	fmt.Printf("Imported %d records from %d files into project %s in %s\n", total, len(ordered), r.projectKey, time.Since(start).Truncate(time.Millisecond))
	if err := r.archiveMedia(); err != nil {
		return err
	}
	return nil
}

func (r importerRunner) importFile(ctx context.Context, path string) (int, string, time.Duration, error) {
	if err := r.restoreArchive(ctx); err != nil {
		return 0, "", 0, err
	}
	count, kind, dur, err := r.importFileCore(ctx, path)
	if err != nil {
		return count, kind, dur, err
	}
	if err := r.archiveMedia(); err != nil {
		return count, kind, dur, err
	}
	return count, kind, dur, nil
}

func (r importerRunner) importFileCore(ctx context.Context, path string) (int, string, time.Duration, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, "", 0, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	imp := importer.NewCSVImporter(f, r.productRepo, r.categoryRepo, r.projectID, r.projectKey, importer.WithMedia(r.mediaRoot, r.mediaBaseURL))

	start := time.Now()
	count, err := imp.Run(ctx)
	if err != nil {
		return count, imp.Kind(), 0, err
	}

	return count, imp.Kind(), time.Since(start), nil
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func (r importerRunner) restoreArchive(ctx context.Context) error {
	if r.mediaRoot == "" || r.projectKey == "" || r.archiveDir == "" {
		return nil
	}
	archivePath := r.archivePath()
	info, err := os.Stat(archivePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("stat archive: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("archive path is a directory: %s", archivePath)
	}

	projectDir := filepath.Join(r.mediaRoot, r.projectKey)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		return fmt.Errorf("create media dir: %w", err)
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("read gzip: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("read archive: %w", err)
		}
		if hdr.Name == "" {
			continue
		}
		rel := filepath.Clean(hdr.Name)
		if strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
			return fmt.Errorf("invalid archive entry: %s", hdr.Name)
		}
		dest := filepath.Join(projectDir, rel)
		if hdr.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(dest, 0o755); err != nil {
				return fmt.Errorf("create dir: %w", err)
			}
			continue
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		if _, err := os.Stat(dest); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("stat file: %w", err)
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return fmt.Errorf("create parent: %w", err)
		}
		out, err := os.Create(dest)
		if err != nil {
			return fmt.Errorf("create file: %w", err)
		}
		if _, err := io.Copy(out, tr); err != nil {
			_ = out.Close()
			return fmt.Errorf("write file: %w", err)
		}
		if err := out.Close(); err != nil {
			return fmt.Errorf("close file: %w", err)
		}
	}
	return nil
}

func (r importerRunner) archiveMedia() error {
	if r.mediaRoot == "" || r.projectKey == "" || r.archiveDir == "" {
		return nil
	}
	projectDir := filepath.Join(r.mediaRoot, r.projectKey)
	if _, err := os.Stat(projectDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("stat media dir: %w", err)
	}
	if err := os.MkdirAll(r.archiveDir, 0o755); err != nil {
		return fmt.Errorf("create archive dir: %w", err)
	}

	files, err := collectMediaFiles(projectDir)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}

	tmp, err := os.CreateTemp(r.archiveDir, "media-*.tar.gz")
	if err != nil {
		return fmt.Errorf("create archive: %w", err)
	}
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
	}()

	gzw := gzip.NewWriter(tmp)
	tw := tar.NewWriter(gzw)

	for _, filePath := range files {
		rel, err := filepath.Rel(projectDir, filePath)
		if err != nil {
			_ = tw.Close()
			_ = gzw.Close()
			return fmt.Errorf("rel path: %w", err)
		}
		info, err := os.Stat(filePath)
		if err != nil {
			_ = tw.Close()
			_ = gzw.Close()
			return fmt.Errorf("stat file: %w", err)
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			_ = tw.Close()
			_ = gzw.Close()
			return fmt.Errorf("header: %w", err)
		}
		hdr.Name = filepath.ToSlash(rel)
		hdr.ModTime = time.Unix(0, 0)
		if err := tw.WriteHeader(hdr); err != nil {
			_ = tw.Close()
			_ = gzw.Close()
			return fmt.Errorf("write header: %w", err)
		}
		f, err := os.Open(filePath)
		if err != nil {
			_ = tw.Close()
			_ = gzw.Close()
			return fmt.Errorf("open file: %w", err)
		}
		if _, err := io.Copy(tw, f); err != nil {
			_ = f.Close()
			_ = tw.Close()
			_ = gzw.Close()
			return fmt.Errorf("write file: %w", err)
		}
		if err := f.Close(); err != nil {
			_ = tw.Close()
			_ = gzw.Close()
			return fmt.Errorf("close file: %w", err)
		}
	}

	if err := tw.Close(); err != nil {
		_ = gzw.Close()
		return fmt.Errorf("close tar: %w", err)
	}
	if err := gzw.Close(); err != nil {
		return fmt.Errorf("close gzip: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close archive: %w", err)
	}

	finalPath := r.archivePath()
	if err := os.Rename(tmp.Name(), finalPath); err != nil {
		return fmt.Errorf("finalize archive: %w", err)
	}
	return nil
}

func (r importerRunner) archivePath() string {
	return filepath.Join(r.archiveDir, "media.tar.gz")
}

func collectMediaFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk media dir: %w", err)
	}
	sort.Strings(files)
	return files, nil
}

func archiveDirForInput(inputPath string) string {
	if inputPath == "" {
		return ""
	}
	info, err := os.Stat(inputPath)
	if err == nil && info.IsDir() {
		return inputPath
	}
	return filepath.Dir(inputPath)
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
