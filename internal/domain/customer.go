package domain

import "time"

// CustomerAddress stores address fields returned to clients.
type CustomerAddress struct {
	ID         string `json:"id"`
	FirstName  string `json:"firstName,omitempty"`
	LastName   string `json:"lastName,omitempty"`
	Country    string `json:"country,omitempty"`
	StreetName string `json:"streetName,omitempty"`
	PostalCode string `json:"postalCode,omitempty"`
	City       string `json:"city,omitempty"`
	Email      string `json:"email,omitempty"`
	Department string `json:"department,omitempty"`
}

// Customer represents a registered user tied to a project.
type Customer struct {
	ID                       string            `json:"id"`
	ProjectID                string            `json:"projectId"`
	Email                    string            `json:"email"`
	PasswordHash             string            `json:"-"`
	FirstName                string            `json:"firstName,omitempty"`
	LastName                 string            `json:"lastName,omitempty"`
	DateOfBirth              string            `json:"dateOfBirth,omitempty"`
	Addresses                []CustomerAddress `json:"addresses,omitempty"`
	DefaultShippingAddressID string            `json:"defaultShippingAddressId,omitempty"`
	DefaultBillingAddressID  string            `json:"defaultBillingAddressId,omitempty"`
	ShippingAddressIDs       []string          `json:"shippingAddressIds,omitempty"`
	BillingAddressIDs        []string          `json:"billingAddressIds,omitempty"`
	CreatedAt                time.Time         `json:"createdAt"`
}
