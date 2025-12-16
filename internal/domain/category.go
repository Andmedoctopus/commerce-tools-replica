package domain

import "time"

type Category struct {
	ID              string    `json:"id"`
	ProjectID       string    `json:"-"`
	Key             string    `json:"key"`
	Name            string    `json:"name"`
	Slug            string    `json:"slug,omitempty"`
	OrderHint       string    `json:"orderHint,omitempty"`
	ParentKey       string    `json:"parentKey,omitempty"`
	Description     string    `json:"description,omitempty"`
	MetaTitle       string    `json:"metaTitle,omitempty"`
	MetaDescription string    `json:"metaDescription,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
}
