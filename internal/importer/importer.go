package importer

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"commercetools-replica/internal/domain"
)

type ProductWriter interface {
	Upsert(ctx context.Context, product domain.Product) (*domain.Product, error)
}

type CategoryWriter interface {
	Upsert(ctx context.Context, c domain.Category) (*domain.Category, error)
}

// CSVImporter reads commercetools-like CSV exports and inserts/updates products.
type CSVImporter struct {
	reader       *csv.Reader
	productRepo  ProductWriter
	categoryRepo CategoryWriter
	categorySeen map[string]struct{}
	projectID    string
	lastKind     string
}

func NewCSVImporter(r io.Reader, repo ProductWriter, catRepo CategoryWriter, projectID string) *CSVImporter {
	csvr := csv.NewReader(r)
	csvr.FieldsPerRecord = -1 // rows may have trailing commas
	return &CSVImporter{
		reader:       csvr,
		productRepo:  repo,
		categoryRepo: catRepo,
		categorySeen: make(map[string]struct{}),
		projectID:    projectID,
		lastKind:     "products",
	}
}

type csvRow struct {
	ID          string
	Key         string
	Name        string
	Desc        string
	SKU         string
	Cents       int64
	Currency    string
	ImageURLs   []string
	Categories  []string
	ProductType string
}

type categoryRow struct {
	Key             string
	Name            string
	Slug            string
	ParentKey       string
	OrderHint       string
	Description     string
	MetaTitle       string
	MetaDescription string
}

// Run parses CSV rows and upserts products grouped by product key.
func (i *CSVImporter) Run(ctx context.Context) (int, error) {
	headers, err := i.reader.Read()
	if err != nil {
		return 0, fmt.Errorf("read headers: %w", err)
	}

	index := headerIndex(headers)

	if isCategoryFile(index) {
		i.lastKind = "categories"
		if i.categoryRepo == nil {
			return 0, fmt.Errorf("category import requested but category repo is nil")
		}
		return i.runCategories(ctx, index)
	}
	i.lastKind = "products"

	var (
		current  *csvRow
		imported int
	)

	for {
		record, err := i.reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return imported, fmt.Errorf("read row: %w", err)
		}

		row := parseRow(record, index)
		if row == nil {
			continue
		}

		if row.Key != "" {
			if current != nil {
				if err := i.save(ctx, current); err != nil {
					return imported, err
				}
				imported++
			}
			current = row
			continue
		}

		// Continuation rows (images) belong to the current product.
		if current != nil && len(row.ImageURLs) > 0 {
			current.ImageURLs = append(current.ImageURLs, row.ImageURLs...)
		}
	}

	if current != nil {
		if err := i.save(ctx, current); err != nil {
			return imported, err
		}
		imported++
	}

	return imported, nil
}

func (i *CSVImporter) Kind() string {
	return i.lastKind
}

func (i *CSVImporter) save(ctx context.Context, row *csvRow) error {
	if row.Key == "" || row.Name == "" || row.SKU == "" || row.Cents == 0 || row.Currency == "" {
		return fmt.Errorf("invalid product row (missing required fields) for key %q", row.Key)
	}
	if row.ID != "" && len(row.ID) != 36 {
		return fmt.Errorf("invalid id for key %q: %s", row.Key, row.ID)
	}

	attrs := map[string]interface{}{}
	if len(row.ImageURLs) > 0 {
		attrs["images"] = row.ImageURLs
	}
	catKeys := pickCategoryKeys(row)
	if len(catKeys) > 0 {
		attrs["categories"] = catKeys
	}

	p := domain.Product{
		ID:          row.ID,
		ProjectID:   i.projectID,
		Key:         row.Key,
		SKU:         row.SKU,
		Name:        row.Name,
		Description: row.Desc,
		PriceCents:  row.Cents,
		Currency:    row.Currency,
		Attributes:  attrs,
	}

	_, err := i.productRepo.Upsert(ctx, p)
	if err != nil {
		return fmt.Errorf("upsert product %q: %w", row.Key, err)
	}

	if i.categoryRepo != nil {
		seen := make(map[string]struct{})
		for _, cat := range catKeys {
			cat = strings.TrimSpace(cat)
			if cat == "" {
				continue
			}
			if _, ok := seen[cat]; ok {
				continue
			}
			seen[cat] = struct{}{}
			if _, ok := i.categorySeen[cat]; ok {
				continue
			}
			if _, err := i.categoryRepo.Upsert(ctx, domain.Category{
				ProjectID: i.projectID,
				Key:       cat,
				Name:      displayNameFromKey(cat),
				Slug:      cat,
			}); err != nil {
				return fmt.Errorf("upsert category %q: %w", cat, err)
			}
			i.categorySeen[cat] = struct{}{}
		}
	}
	return nil
}

func headerIndex(headers []string) map[string]int {
	idx := make(map[string]int, len(headers))
	for i, h := range headers {
		idx[h] = i
	}
	return idx
}

func isCategoryFile(idx map[string]int) bool {
	_, hasParent := idx["parent.key"]
	_, hasSlug := idx["slug.en"]
	_, hasProductSKU := idx["variants.sku"]
	return (hasParent || hasSlug) && !hasProductSKU
}

func parseRow(record []string, index map[string]int) *csvRow {
	id := pick(record, index, "id")
	key := pick(record, index, "key")
	name := pick(record, index, "name.en")
	desc := pick(record, index, "description.en")
	sku := pick(record, index, "variants.sku")
	currency := pick(record, index, "variants.prices.value.currencyCode")
	centStr := pick(record, index, "variants.prices.value.centAmount")

	imageURL := pick(record, index, "variants.images.url")
	categories := pickCategories(record, index, "categories")
	ptype := pick(record, index, "productType.key")
	categories = normalizeCategoryKeys(categories, ptype)

	if key == "" && imageURL == "" {
		return nil
	}

	var cents int64
	if centStr != "" {
		cents, _ = strconv.ParseInt(centStr, 10, 64)
	}

	row := &csvRow{
		Key:         key,
		Name:        name,
		Desc:        desc,
		SKU:         sku,
		Cents:       cents,
		Currency:    currency,
		ID:          id,
		Categories:  categories,
		ProductType: ptype,
	}
	if imageURL != "" {
		row.ImageURLs = []string{strings.TrimSpace(imageURL)}
	}
	return row
}

func pick(record []string, index map[string]int, key string) string {
	pos, ok := index[key]
	if !ok || pos >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[pos])
}

func pickCategories(record []string, index map[string]int, key string) []string {
	val := pick(record, index, key)
	if val == "" {
		return nil
	}
	parts := strings.FieldsFunc(val, func(r rune) bool {
		return r == ',' || r == ';'
	})
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func normalizeCategoryKeys(cats []string, fallback string) []string {
	if len(cats) == 0 && fallback != "" {
		cats = []string{fallback}
	}
	seen := make(map[string]struct{})
	var out []string
	for _, c := range cats {
		n := normalizeCategoryKey(c)
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out
}

func normalizeCategoryKey(key string) string {
	key = strings.TrimSpace(key)
	key = strings.TrimSuffix(key, "-types")
	key = strings.TrimSuffix(key, "-type")
	return key
}

func displayNameFromKey(key string) string {
	parts := strings.FieldsFunc(key, func(r rune) bool {
		return r == '-' || r == '_' || r == ' '
	})
	for i, p := range parts {
		if len(p) == 0 {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
	}
	return strings.Join(parts, " ")
}

func pickCategoryKeys(row *csvRow) []string {
	if len(row.Categories) > 0 {
		return row.Categories
	}
	if row.ProductType != "" {
		return []string{row.ProductType}
	}
	return nil
}

func (i *CSVImporter) runCategories(ctx context.Context, index map[string]int) (int, error) {
	imported := 0
	for {
		record, err := i.reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return imported, fmt.Errorf("read row: %w", err)
		}
		row := parseCategoryRow(record, index)
		if row == nil {
			continue
		}
		if err := i.saveCategory(ctx, row); err != nil {
			return imported, err
		}
		imported++
	}
	return imported, nil
}

func parseCategoryRow(record []string, index map[string]int) *categoryRow {
	key := pick(record, index, "key")
	name := pick(record, index, "name.en")
	slug := pick(record, index, "slug.en")
	parent := pick(record, index, "parent.key")
	order := pick(record, index, "orderHint")
	desc := pick(record, index, "description.en")
	metaTitle := pick(record, index, "metaTitle.en")
	metaDesc := pick(record, index, "metaDescription.en")

	if key == "" {
		key = slug
	}
	if key == "" {
		return nil
	}
	if slug == "" {
		slug = key
	}
	if name == "" {
		name = displayNameFromKey(key)
	}

	return &categoryRow{
		Key:             key,
		Name:            name,
		Slug:            slug,
		ParentKey:       parent,
		OrderHint:       order,
		Description:     desc,
		MetaTitle:       metaTitle,
		MetaDescription: metaDesc,
	}
}

func (i *CSVImporter) saveCategory(ctx context.Context, row *categoryRow) error {
	key := normalizeCategoryKey(row.Key)
	if key == "" {
		return nil
	}
	if _, ok := i.categorySeen[key]; ok {
		return nil
	}
	_, err := i.categoryRepo.Upsert(ctx, domain.Category{
		ProjectID:       i.projectID,
		Key:             key,
		Name:            row.Name,
		Slug:            row.Slug,
		OrderHint:       row.OrderHint,
		ParentKey:       normalizeCategoryKey(row.ParentKey),
		Description:     row.Description,
		MetaTitle:       row.MetaTitle,
		MetaDescription: row.MetaDescription,
	})
	if err != nil {
		return fmt.Errorf("upsert category %q: %w", key, err)
	}
	i.categorySeen[key] = struct{}{}
	return nil
}
