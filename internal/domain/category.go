package domain

import "time"

type Category struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"-"`
	Key       string    `json:"key"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}
