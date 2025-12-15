package httpserver

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"commercetools-replica/internal/domain"
)

type ctProduct struct {
	ID                  string                   `json:"id"`
	Key                 string                   `json:"key,omitempty"`
	Version             int                      `json:"version"`
	CreatedAt           time.Time                `json:"createdAt"`
	LastModifiedAt      time.Time                `json:"lastModifiedAt"`
	LastMessageSequence int                      `json:"lastMessageSequenceNumber,omitempty"`
	ProductType         *ctRef                   `json:"productType,omitempty"`
	MasterData          ctMasterData             `json:"masterData"`
	PriceMode           string                   `json:"priceMode,omitempty"`
	TaxCategory         *ctRef                   `json:"taxCategory,omitempty"`
	State               *ctRef                   `json:"state,omitempty"`
	LastVariantID       int                      `json:"lastVariantId,omitempty"`
	HasStagedChanges    bool                     `json:"hasStagedChanges"`
	Published           bool                     `json:"published"`
	Slug                map[string]string        `json:"slug,omitempty"`
	MasterVariantID     int                      `json:"masterVariantId,omitempty"`
	LastModifiedBy      interface{}              `json:"lastModifiedBy,omitempty"`
	CreatedBy           interface{}              `json:"createdBy,omitempty"`
	MetaTitle           map[string]string        `json:"metaTitle,omitempty"`
	MetaDescription     map[string]string        `json:"metaDescription,omitempty"`
	MetaKeywords        map[string]string        `json:"metaKeywords,omitempty"`
	SearchKeywords      map[string][]interface{} `json:"searchKeywords,omitempty"`
}

type ctMasterData struct {
	Current          ctProductData `json:"current"`
	Staged           ctProductData `json:"staged"`
	Published        bool          `json:"published"`
	HasStagedChanges bool          `json:"hasStagedChanges"`
}

type ctProductData struct {
	Name            map[string]string        `json:"name"`
	Description     map[string]string        `json:"description,omitempty"`
	Slug            map[string]string        `json:"slug,omitempty"`
	MetaTitle       map[string]string        `json:"metaTitle,omitempty"`
	MetaDescription map[string]string        `json:"metaDescription,omitempty"`
	MasterVariant   ctVariant                `json:"masterVariant"`
	Variants        []ctVariant              `json:"variants"`
	SearchKeywords  map[string][]interface{} `json:"searchKeywords,omitempty"`
	Attributes      []interface{}            `json:"attributes"`
	Assets          []interface{}            `json:"assets"`
	Categories      []interface{}            `json:"categories"`
	CategoryOrder   map[string]string        `json:"categoryOrderHints,omitempty"`
}

type ctVariant struct {
	ID         int           `json:"id"`
	SKU        string        `json:"sku"`
	Prices     []ctPrice     `json:"prices"`
	Images     []ctImage     `json:"images"`
	Assets     []interface{} `json:"assets"`
	Attributes []interface{} `json:"attributes"`
}

type ctPrice struct {
	ID    string       `json:"id,omitempty"`
	Value ctPriceValue `json:"value"`
}

type ctPriceValue struct {
	Type           string `json:"type"`
	CurrencyCode   string `json:"currencyCode"`
	CentAmount     int64  `json:"centAmount"`
	FractionDigits int    `json:"fractionDigits"`
}

type ctImage struct {
	URL        string        `json:"url"`
	Label      string        `json:"label,omitempty"`
	Dimensions *ctDimensions `json:"dimensions,omitempty"`
}

type ctDimensions struct {
	W int `json:"w"`
	H int `json:"h"`
}

type ctRef struct {
	TypeID string `json:"typeId,omitempty"`
	ID     string `json:"id,omitempty"`
	Key    string `json:"key,omitempty"`
}

type searchRequest struct {
	Query struct {
		Filter []filterClause `json:"filter"`
	} `json:"query"`
	Sort   []sortClause `json:"sort"`
	Limit  int          `json:"limit"`
	Offset int          `json:"offset"`
}

type filterClause struct {
	Range *rangeFilter `json:"range"`
	Exact *exactFilter `json:"exact"`
}

type rangeFilter struct {
	Field     string `json:"field"`
	FieldType string `json:"fieldType"`
	GTE       *int64 `json:"gte"`
	LTE       *int64 `json:"lte"`
}

type exactFilter struct {
	Field string `json:"field"`
	Value string `json:"value"`
}
type sortClause struct {
	Field    string `json:"field"`
	Language string `json:"language"`
	Order    string `json:"order"`
}

type searchResponse struct {
	Total   int                `json:"total"`
	Offset  int                `json:"offset"`
	Limit   int                `json:"limit"`
	Facets  []interface{}      `json:"facets"`
	Results []searchResultItem `json:"results"`
}

type searchResultItem struct {
	ID string `json:"id"`
}

type ctCategory struct {
	ID                  string            `json:"id"`
	Key                 string            `json:"key,omitempty"`
	Version             int               `json:"version"`
	CreatedAt           time.Time         `json:"createdAt"`
	LastModifiedAt      time.Time         `json:"lastModifiedAt"`
	Name                map[string]string `json:"name"`
	Slug                map[string]string `json:"slug"`
	Ancestors           []interface{}     `json:"ancestors"`
	Parent              interface{}       `json:"parent"`
	OrderHint           string            `json:"orderHint,omitempty"`
	MetaTitle           map[string]string `json:"metaTitle,omitempty"`
	MetaDescription     map[string]string `json:"metaDescription,omitempty"`
	LastMessageSequence int               `json:"lastMessageSequenceNumber,omitempty"`
}

type ctCategoryList struct {
	Limit   int          `json:"limit"`
	Offset  int          `json:"offset"`
	Count   int          `json:"count"`
	Total   int          `json:"total"`
	Results []ctCategory `json:"results"`
}

func buildCategoryList(cats []domain.Category, limit, offset int) ctCategoryList {
	if limit <= 0 {
		limit = len(cats)
	}
	if offset < 0 {
		offset = 0
	}
	end := offset + limit
	if end > len(cats) {
		end = len(cats)
	}
	sliced := []domain.Category{}
	if offset < len(cats) {
		sliced = cats[offset:end]
	}
	out := ctCategoryList{
		Limit:  limit,
		Offset: offset,
		Total:  len(cats),
		Count:  len(sliced),
	}
	for _, c := range sliced {
		out.Results = append(out.Results, toCTCategory(c))
	}
	return out
}

func parseLimitOffset(qLimit, qOffset string) (int, int) {
	limit := 0
	offset := 0
	if qLimit != "" {
		if v, err := strconv.Atoi(qLimit); err == nil && v >= 0 {
			limit = v
		}
	}
	if qOffset != "" {
		if v, err := strconv.Atoi(qOffset); err == nil && v >= 0 {
			offset = v
		}
	}
	return limit, offset
}

func toCTProduct(p domain.Product) ctProduct {
	name := map[string]string{"en": p.Name}
	desc := map[string]string{}
	if p.Description != "" {
		desc["en"] = p.Description
	}

	slug := map[string]string{}
	if p.Key != "" {
		slug["en"] = strings.ReplaceAll(strings.ToLower(p.Key), " ", "-")
	}

	images := extractImages(p.Attributes)

	variant := ctVariant{
		ID:         1,
		SKU:        p.SKU,
		Prices:     []ctPrice{{Value: ctPriceValue{Type: "centPrecision", CurrencyCode: p.Currency, CentAmount: p.PriceCents, FractionDigits: 2}}},
		Images:     images,
		Assets:     []interface{}{},
		Attributes: []interface{}{},
	}

	data := ctProductData{
		Name:            name,
		Description:     desc,
		Slug:            slug,
		MetaTitle:       map[string]string{"en": ""},
		MetaDescription: map[string]string{"en": ""},
		MasterVariant:   variant,
		Variants:        []ctVariant{},
		SearchKeywords:  map[string][]interface{}{},
		Attributes:      []interface{}{},
		Assets:          []interface{}{},
		Categories:      []interface{}{},
		CategoryOrder:   map[string]string{},
	}

	return ctProduct{
		ID:             p.ID,
		Key:            p.Key,
		Version:        1,
		CreatedAt:      p.CreatedAt,
		LastModifiedAt: p.CreatedAt,
		MasterData: ctMasterData{
			Current:          data,
			Staged:           data,
			Published:        true,
			HasStagedChanges: false,
		},
		HasStagedChanges: false,
		Published:        true,
		Slug:             slug,
		MetaTitle:        map[string]string{"en": ""},
		MetaDescription:  map[string]string{"en": ""},
		MetaKeywords:     map[string]string{},
		SearchKeywords:   map[string][]interface{}{},
		PriceMode:        "Embedded",
		LastVariantID:    1,
	}
}

func extractImages(attrs map[string]interface{}) []ctImage {
	raw, ok := attrs["images"]
	if !ok {
		return nil
	}
	var urls []string
	switch v := raw.(type) {
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok {
				urls = append(urls, s)
			}
		}
	case []string:
		urls = v
	}
	var images []ctImage
	for _, u := range urls {
		if strings.TrimSpace(u) == "" {
			continue
		}
		images = append(images, ctImage{URL: u})
	}
	return images
}

func buildSearchResponse(products []domain.Product, req searchRequest) searchResponse {
	products = filterProducts(products, req)
	sortProducts(products, req)
	offset := req.Offset
	if offset < 0 {
		offset = 0
	}
	limit := req.Limit
	if limit <= 0 {
		limit = len(products)
	}

	end := offset + limit
	if end > len(products) {
		end = len(products)
	}
	sliced := []domain.Product{}
	if offset < len(products) {
		sliced = products[offset:end]
	}

	results := make([]searchResultItem, 0, len(sliced))
	for _, p := range sliced {
		results = append(results, searchResultItem{ID: p.ID})
	}

	return searchResponse{
		Total:   len(products),
		Offset:  offset,
		Limit:   limit,
		Facets:  []interface{}{},
		Results: results,
	}
}

func filterProducts(products []domain.Product, req searchRequest) []domain.Product {
	var prange *rangeFilter
	var category string
	for _, f := range req.Query.Filter {
		if f.Range != nil && f.Range.Field == "variants.prices.centAmount" {
			prange = f.Range
		}
		if f.Exact != nil && f.Exact.Field == "categories" {
			category = f.Exact.Value
		}
	}

	var filtered []domain.Product
	for _, p := range products {
		if prange != nil {
			if prange.GTE != nil && p.PriceCents < *prange.GTE {
				continue
			}
			if prange.LTE != nil && p.PriceCents > *prange.LTE {
				continue
			}
		}
		if category != "" {
			if !containsCategory(p, category) {
				continue
			}
		}
		filtered = append(filtered, p)
	}
	return filtered
}

func sortProducts(products []domain.Product, req searchRequest) {
	if len(req.Sort) == 0 {
		sort.Slice(products, func(i, j int) bool {
			return strings.ToLower(products[i].Name) < strings.ToLower(products[j].Name)
		})
		return
	}

	// Support name asc/desc and price asc/desc
	field := strings.ToLower(req.Sort[0].Field)
	order := strings.ToLower(req.Sort[0].Order)
	if order == "" {
		order = "asc"
	}

	less := func(i, j int) bool { return true }
	switch field {
	case "name":
		less = func(i, j int) bool {
			li := strings.ToLower(products[i].Name)
			lj := strings.ToLower(products[j].Name)
			if order == "desc" {
				return li > lj
			}
			return li < lj
		}
	case "variants.prices.centamount", "price", "variants.prices.value.centamount":
		less = func(i, j int) bool {
			if order == "desc" {
				return products[i].PriceCents > products[j].PriceCents
			}
			return products[i].PriceCents < products[j].PriceCents
		}
	default:
		less = func(i, j int) bool {
			return strings.ToLower(products[i].Name) < strings.ToLower(products[j].Name)
		}
	}

	sort.Slice(products, less)
}

func containsCategory(p domain.Product, categoryID string) bool {
	raw, ok := p.Attributes["categories"]
	if !ok {
		return false
	}
	switch v := raw.(type) {
	case []interface{}:
		for _, c := range v {
			if s, ok := c.(string); ok && s == categoryID {
				return true
			}
		}
	case []string:
		for _, s := range v {
			if s == categoryID {
				return true
			}
		}
	case string:
		return v == categoryID
	}
	return false
}

func toCTCategory(c domain.Category) ctCategory {
	name := c.Name
	if name == "" {
		name = c.Key
	}
	nameMap := map[string]string{"en": name}
	slugMap := map[string]string{"en": c.Slug}
	return ctCategory{
		ID:             c.ID,
		Key:            c.Key,
		Version:        1,
		CreatedAt:      c.CreatedAt,
		LastModifiedAt: c.CreatedAt,
		Name:           nameMap,
		Slug:           slugMap,
		Ancestors:      []interface{}{},
		Parent:         nil,
		OrderHint:      "",
		MetaTitle:      nameMap,
	}
}
