package httpserver

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"commercetools-replica/internal/domain"
	"log"
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
	Ancestors           []ctRef           `json:"ancestors"`
	Parent              *ctRef            `json:"parent"`
	OrderHint           string            `json:"orderHint,omitempty"`
	MetaTitle           map[string]string `json:"metaTitle,omitempty"`
	MetaDescription     map[string]string `json:"metaDescription,omitempty"`
	Description         map[string]string `json:"description,omitempty"`
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
	keyToCat := make(map[string]domain.Category, len(cats))
	keyToID := make(map[string]string, len(cats))
	for _, c := range cats {
		keyToCat[c.Key] = c
		keyToID[c.Key] = c.ID
	}
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
		parentID := keyToID[c.ParentKey]
		ancestors := buildAncestors(c, keyToCat, keyToID)
		out.Results = append(out.Results, toCTCategory(c, parentID, ancestors))
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

func buildAncestors(c domain.Category, keyToCat map[string]domain.Category, keyToID map[string]string) []ctRef {
	var ancestors []ctRef
	seen := make(map[string]struct{})
	current := c.ParentKey
	for current != "" {
		if _, ok := seen[current]; ok {
			break
		}
		seen[current] = struct{}{}
		id := keyToID[current]
		if id == "" {
			break
		}
		ancestors = append([]ctRef{{TypeID: "category", ID: id}}, ancestors...)
		next := keyToCat[current].ParentKey
		current = next
	}
	return ancestors
}

func toCTCategory(c domain.Category, parentID string, ancestors []ctRef) ctCategory {
	name := c.Name
	if name == "" {
		name = c.Key
	}
	nameMap := map[string]string{"en": name}
	slugVal := c.Slug
	if slugVal == "" {
		slugVal = c.Key
	}
	slugMap := map[string]string{"en": slugVal}
	var parentRef *ctRef
	if parentID != "" {
		parentRef = &ctRef{TypeID: "category", ID: parentID}
	}
	metaTitle := nameMap
	metaDesc := map[string]string{}
	if c.MetaDescription != "" {
		metaDesc["en"] = c.MetaDescription
	}
	descMap := map[string]string{}
	if c.Description != "" {
		descMap["en"] = c.Description
	}
	return ctCategory{
		ID:              c.ID,
		Key:             c.Key,
		Version:         1,
		CreatedAt:       c.CreatedAt,
		LastModifiedAt:  c.CreatedAt,
		Name:            nameMap,
		Slug:            slugMap,
		Ancestors:       ancestors,
		Parent:          parentRef,
		OrderHint:       c.OrderHint,
		MetaTitle:       metaTitle,
		MetaDescription: metaDesc,
		Description:     descMap,
	}
}

func toCTProduct(logger *log.Logger, p domain.Product, fileURLHost string) ctProduct {
	name := map[string]string{"en": p.Name}
	desc := map[string]string{}
	if p.Description != "" {
		desc["en"] = p.Description
	}

	slug := map[string]string{}
	if p.Key != "" {
		slug["en"] = strings.ReplaceAll(strings.ToLower(p.Key), " ", "-")
	}

	images := extractImages(logger, p.Attributes, fileURLHost)

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

func extractImages(logger *log.Logger, attrs map[string]interface{}, fileURLHost string) []ctImage {
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
		var url = fmt.Sprintf("%s%s", fileURLHost, u)
		logger.Printf(">>> URL %s ... %s", url, fileURLHost)
		images = append(images, ctImage{URL: url})
	}
	return images
}

func buildSearchResponse(products []domain.Product, categories []domain.Category, req searchRequest) searchResponse {
	idToKey, keyToID := categoryMaps(categories)
	products = filterProducts(products, req, idToKey, keyToID)
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

func categoryMaps(categories []domain.Category) (map[string]string, map[string]string) {
	if len(categories) == 0 {
		return nil, nil
	}
	idToKey := make(map[string]string, len(categories))
	keyToID := make(map[string]string, len(categories))
	for _, c := range categories {
		if c.ID != "" && c.Key != "" {
			idToKey[c.ID] = c.Key
			keyToID[c.Key] = c.ID
		}
	}
	return idToKey, keyToID
}

func filterProducts(products []domain.Product, req searchRequest, categoryIDToKey map[string]string, categoryKeyToID map[string]string) []domain.Product {
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

	var categoryTargets []string
	if category != "" {
		categoryTargets = append(categoryTargets, category)
		if categoryIDToKey != nil {
			if k := categoryIDToKey[category]; k != "" {
				categoryTargets = append(categoryTargets, k)
			}
		}
		if categoryKeyToID != nil {
			if id := categoryKeyToID[category]; id != "" {
				categoryTargets = append(categoryTargets, id)
			}
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
		if len(categoryTargets) > 0 {
			if !containsAnyCategory(p, categoryTargets) {
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

func containsAnyCategory(p domain.Product, candidates []string) bool {
	raw, ok := p.Attributes["categories"]
	if !ok {
		return false
	}
	switch v := raw.(type) {
	case []interface{}:
		for _, c := range v {
			s, ok := c.(string)
			if !ok {
				continue
			}
			for _, candidate := range candidates {
				if s == candidate {
					return true
				}
			}
		}
	case []string:
		for _, s := range v {
			for _, candidate := range candidates {
				if s == candidate {
					return true
				}
			}
		}
	case string:
		for _, candidate := range candidates {
			if v == candidate {
				return true
			}
		}
	}
	return false
}
