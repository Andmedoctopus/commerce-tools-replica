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

type stubCategoryRepo struct {
	items []domain.Category
}

func (s *stubProductRepo) Upsert(_ context.Context, p domain.Product) (*domain.Product, error) {
	s.items = append(s.items, p)
	return &p, nil
}

func (s *stubCategoryRepo) Upsert(_ context.Context, c domain.Category) (*domain.Category, error) {
	s.items = append(s.items, c)
	return &c, nil
}

func TestCSVImporter_Run(t *testing.T) {
	csvData := `id,key,name.en,description.en,variants.sku,variants.prices.value.centAmount,variants.prices.value.currencyCode,productType.key,categories,variants.images.url
00000000-0000-0000-0000-000000000001,prod-1,Prod One,Desc one,SKU-1,100,EUR,pots,cat-1;cat-2,https://example.com/img1.jpg
,,,,,,,,,https://example.com/img2.jpg
00000000-0000-0000-0000-000000000002,prod-2,Prod Two,Desc two,SKU-2,200,USD,succulents,,`

	repo := &stubProductRepo{}
	catRepo := &stubCategoryRepo{}
	imp := NewCSVImporter(strings.NewReader(csvData), repo, catRepo, "project-123")

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
	if repo.items[0].ID != "00000000-0000-0000-0000-000000000001" {
		t.Fatalf("expected id to be preserved, got %s", repo.items[0].ID)
	}
	if cats, ok := repo.items[0].Attributes["categories"].([]string); !ok || len(cats) != 2 {
		t.Fatalf("expected categories on first product, got %+v", repo.items[0].Attributes["categories"])
	}
	if len(catRepo.items) != 3 { // cat-1, cat-2, productType fallback (succulents)
		t.Fatalf("expected 3 category upserts, got %d", len(catRepo.items))
	}
}
