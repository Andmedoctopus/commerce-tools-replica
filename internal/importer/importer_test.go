package importer

import (
	"context"
	"strings"
	"testing"

	"commercetools-replica/internal/domain"
)

type stubProductRepo struct {
	items []domain.Product
}

func (s *stubProductRepo) Upsert(_ context.Context, p domain.Product) (*domain.Product, error) {
	s.items = append(s.items, p)
	return &p, nil
}

func TestCSVImporter_Run(t *testing.T) {
	csvData := `key,name.en,description.en,variants.sku,variants.prices.value.centAmount,variants.prices.value.currencyCode,variants.images.url
prod-1,Prod One,Desc one,SKU-1,100,EUR,https://example.com/img1.jpg
,,,,,,https://example.com/img2.jpg
prod-2,Prod Two,Desc two,SKU-2,200,USD,`

	repo := &stubProductRepo{}
	imp := NewCSVImporter(strings.NewReader(csvData), repo, "project-123")

	count, err := imp.Run(context.Background())
	if err != nil {
		t.Fatalf("import run: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 products imported, got %d", count)
	}

	if len(repo.items) != 2 {
		t.Fatalf("expected 2 products saved, got %d", len(repo.items))
	}

	if len(repo.items[0].Attributes["images"].([]string)) != 2 {
		t.Fatalf("expected 2 images on first product")
	}
	if repo.items[0].Key != "prod-1" || repo.items[0].SKU != "SKU-1" || repo.items[0].PriceCents != 100 || repo.items[0].Currency != "EUR" {
		t.Fatalf("unexpected product data: %+v", repo.items[0])
	}
}
