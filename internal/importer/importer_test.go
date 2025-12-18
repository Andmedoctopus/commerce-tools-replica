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

func TestCSVImporter_RunCategoriesFile(t *testing.T) {
	csvData := `key,name.en,slug.en,parent.key,orderHint,description.en,metaTitle.en,metaDescription.en
indoor-pots,Indoor Pots,indoor-pots,,4.1,Desc indoor,Meta indoor,Meta desc indoor
,Foliage plants,foliage-plants,,3.3,Desc foliage,,
succulents,Succulents,,,,"",,Meta desc succ
,Indoor plants,indoor-plants,,3,,,
,Pots,pots,,4,,,
`
	catRepo := &stubCategoryRepo{}
	imp := NewCSVImporter(strings.NewReader(csvData), nil, catRepo, "project-123")

	count, err := imp.Run(context.Background())
	if err != nil {
		t.Fatalf("import run: %v", err)
	}
	if count != 5 {
		t.Fatalf("expected 5 categories imported, got %d", count)
	}
	if catRepo.items[0].Key != "indoor-pots" || catRepo.items[0].OrderHint != "4.1" || catRepo.items[0].ParentKey != "pots" || catRepo.items[0].Description != "Desc indoor" || catRepo.items[0].MetaTitle != "Meta indoor" || catRepo.items[0].MetaDescription != "Meta desc indoor" {
		t.Fatalf("unexpected first category %+v", catRepo.items[0])
	}
	if catRepo.items[1].Key != "foliage-plants" || catRepo.items[1].Slug != "foliage-plants" || catRepo.items[1].ParentKey != "indoor-plants" {
		t.Fatalf("expected slug fallback and inferred parent on second: %+v", catRepo.items[1])
	}
	if catRepo.items[2].Name != "Succulents" || catRepo.items[2].ParentKey != "" {
		t.Fatalf("expected title-cased root name, got %+v", catRepo.items[2])
	}
	if catRepo.items[3].Key != "indoor-plants" || catRepo.items[4].Key != "pots" {
		t.Fatalf("expected root categories to be imported, got %v and %v", catRepo.items[3].Key, catRepo.items[4].Key)
	}
}

func TestDetectKind(t *testing.T) {
	productCSV := `id,key,name.en,variants.sku
prod-1,prod-1,Prod One,SKU-1`
	categoryCSV := `key,name.en,slug.en,parent.key
indoor-pots,Indoor Pots,indoor-pots,`

	kind, err := DetectKind(strings.NewReader(productCSV))
	if err != nil {
		t.Fatalf("detect product kind: %v", err)
	}
	if kind != KindProducts {
		t.Fatalf("expected product kind, got %s", kind)
	}

	kind, err = DetectKind(strings.NewReader(categoryCSV))
	if err != nil {
		t.Fatalf("detect category kind: %v", err)
	}
	if kind != KindCategories {
		t.Fatalf("expected category kind, got %s", kind)
	}
}
