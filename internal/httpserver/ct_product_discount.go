package httpserver

import "time"

type ctProductDiscount struct {
	ID             string                 `json:"id"`
	Key            string                 `json:"key,omitempty"`
	Name           map[string]string      `json:"name"`
	Description    map[string]string      `json:"description,omitempty"`
	Value          ctProductDiscountValue `json:"value"`
	Predicate      string                 `json:"predicate"`
	IsActive       bool                   `json:"isActive"`
	SortOrder      string                 `json:"sortOrder"`
	Version        int                    `json:"version"`
	CreatedAt      time.Time              `json:"createdAt"`
	LastModifiedAt time.Time              `json:"lastModifiedAt"`
}

type ctProductDiscountValue struct {
	Type      string `json:"type"`
	Permyriad int    `json:"permyriad"`
}

type ctProductDiscountList struct {
	Limit   int                 `json:"limit"`
	Offset  int                 `json:"offset"`
	Count   int                 `json:"count"`
	Total   int                 `json:"total"`
	Results []ctProductDiscount `json:"results"`
}

func buildProductDiscountList(limit, offset int) ctProductDiscountList {
	createdAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	discounts := []ctProductDiscount{
		{
			ID:          "disc-1",
			Key:         "spring-15",
			Name:        map[string]string{"en": "Spring 15%"},
			Description: map[string]string{"en": "15% off selected categories"},
			Value: ctProductDiscountValue{
				Type:      "relative",
				Permyriad: 1500,
			},
			Predicate:      `product.categories.id containsAny ("cactus", "cat-uuid-2")`,
			IsActive:       true,
			SortOrder:      "0.2",
			Version:        1,
			CreatedAt:      createdAt,
			LastModifiedAt: createdAt,
		},
	}

	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	end := offset + limit
	if end > len(discounts) {
		end = len(discounts)
	}
	sliced := []ctProductDiscount{}
	if offset < len(discounts) {
		sliced = discounts[offset:end]
	}

	return ctProductDiscountList{
		Limit:   limit,
		Offset:  offset,
		Count:   len(sliced),
		Total:   len(discounts),
		Results: sliced,
	}
}
