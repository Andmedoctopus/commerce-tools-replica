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

// CSVImporter reads commercetools-like CSV exports and inserts/updates products.
type CSVImporter struct {
	reader      *csv.Reader
	productRepo ProductWriter
	projectID   string
}

func NewCSVImporter(r io.Reader, repo ProductWriter, projectID string) *CSVImporter {
	csvr := csv.NewReader(r)
	csvr.FieldsPerRecord = -1 // rows may have trailing commas
	return &CSVImporter{
		reader:      csvr,
		productRepo: repo,
		projectID:   projectID,
	}
}

type csvRow struct {
	ID        string
	Key       string
	Name      string
	Desc      string
	SKU       string
	Cents     int64
	Currency  string
	ImageURLs []string
}

// Run parses CSV rows and upserts products grouped by product key.
func (i *CSVImporter) Run(ctx context.Context) (int, error) {
	headers, err := i.reader.Read()
	if err != nil {
		return 0, fmt.Errorf("read headers: %w", err)
	}
	index := headerIndex(headers)

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
	return nil
}

func headerIndex(headers []string) map[string]int {
	idx := make(map[string]int, len(headers))
	for i, h := range headers {
		idx[h] = i
	}
	return idx
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

	if key == "" && imageURL == "" {
		return nil
	}

	var cents int64
	if centStr != "" {
		cents, _ = strconv.ParseInt(centStr, 10, 64)
	}

	row := &csvRow{
		Key:      key,
		Name:     name,
		Desc:     desc,
		SKU:      sku,
		Cents:    cents,
		Currency: currency,
		ID:       id,
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
