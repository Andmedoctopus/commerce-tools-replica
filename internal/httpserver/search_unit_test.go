package httpserver

import (
	"testing"

	"commercetools-replica/internal/domain"
)

func TestBuildSearchResponse_FiltersByPriceAndCategory(t *testing.T) {
	products := []domain.Product{
		{ID: "cheap", Name: "Cheap", PriceCents: 50, Attributes: map[string]interface{}{"categories": []string{"cat-key"}}},
		{ID: "costly", Name: "Costly", PriceCents: 500, Attributes: map[string]interface{}{"categories": []string{"other"}}},
	}
	categories := []domain.Category{{ID: "cat-id", Key: "cat-key", Name: "Cat"}}
	req := searchRequest{}
	req.Query.Filter = []filterClause{
		{Range: &rangeFilter{Field: "variants.prices.centAmount", GTE: int64Ptr(10), LTE: int64Ptr(100)}},
		{Exact: &exactFilter{Field: "categories", Value: "cat-id"}},
	}
	req.Limit = 10
	resp := buildSearchResponse(products, categories, req)
	if resp.Total != 1 || len(resp.Results) != 1 || resp.Results[0].ID != "cheap" {
		t.Fatalf("unexpected search response %+v", resp)
	}
}

func TestSortProducts_DefaultsToNameAsc(t *testing.T) {
	products := []domain.Product{
		{Name: "Zeta"},
		{Name: "Alpha"},
	}
	req := searchRequest{}
	sortProducts(products, req)
	if products[0].Name != "Alpha" {
		t.Fatalf("expected Alpha first, got %+v", products)
	}
}

func int64Ptr(v int64) *int64 {
	return &v
}
