package domain

import "time"

type Cart struct {
	ID          string     `json:"id"`
	ProjectID   string     `json:"-"`
	CustomerID  *string    `json:"customerId,omitempty"`
	AnonymousID *string    `json:"-"`
	Currency    string     `json:"currency"`
	TotalCents  int64      `json:"totalCents"`
	State       string     `json:"state"`
	CreatedAt   time.Time  `json:"createdAt"`
	Lines       []CartLine `json:"lineItems,omitempty"`
}

type CartLine struct {
	ID             string                 `json:"id"`
	CartID         string                 `json:"cartId"`
	ProductID      string                 `json:"productId"`
	Quantity       int                    `json:"quantity"`
	UnitPriceCents int64                  `json:"unitPriceCents"`
	TotalCents     int64                  `json:"totalCents"`
	Snapshot       map[string]interface{} `json:"snapshot,omitempty"`
	CreatedAt      time.Time              `json:"createdAt"`
}
