package domain

import "time"

type Product struct {
	ID          string                 `json:"id"`
	ProjectID   string                 `json:"-"`
	Key         string                 `json:"key"`
	SKU         string                 `json:"sku"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	PriceCents  int64                  `json:"priceCents"`
	Currency    string                 `json:"currency"`
	Attributes  map[string]interface{} `json:"attributes,omitempty"`
	CreatedAt   time.Time              `json:"createdAt"`
}
