package httpserver

import (
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
