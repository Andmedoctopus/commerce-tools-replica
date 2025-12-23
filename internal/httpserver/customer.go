package httpserver

import (
	"encoding/base64"
	"time"

	"commercetools-replica/internal/domain"
)

type signupRequest struct {
	Email                  string           `json:"email"`
	Password               string           `json:"password"`
	FirstName              string           `json:"firstName"`
	LastName               string           `json:"lastName"`
	DateOfBirth            string           `json:"dateOfBirth"`
	Addresses              []addressRequest `json:"addresses"`
	DefaultShippingAddress *int             `json:"defaultShippingAddress"`
	DefaultBillingAddress  *int             `json:"defaultBillingAddress"`
}

type addressRequest struct {
	FirstName  string `json:"firstName"`
	LastName   string `json:"lastName"`
	Country    string `json:"country"`
	StreetName string `json:"streetName"`
	PostalCode string `json:"postalCode"`
	City       string `json:"city"`
	Email      string `json:"email"`
	Department string `json:"department"`
}

type tokenRequest struct {
	GrantType string `form:"grant_type" binding:"required"`
	Username  string `form:"username" binding:"required"`
	Password  string `form:"password" binding:"required"`
	Scope     string `form:"scope" binding:"required"`
}

type customerResponse struct {
	Customer ctCustomer `json:"customer"`
}

type ctCustomer struct {
	ID                        string        `json:"id"`
	Version                   int           `json:"version"`
	VersionModifiedAt         time.Time     `json:"versionModifiedAt"`
	LastMessageSequenceNumber int           `json:"lastMessageSequenceNumber"`
	CreatedAt                 time.Time     `json:"createdAt"`
	LastModifiedAt            time.Time     `json:"lastModifiedAt"`
	LastModifiedBy            auditInfo     `json:"lastModifiedBy"`
	CreatedBy                 auditInfo     `json:"createdBy"`
	Email                     string        `json:"email"`
	FirstName                 string        `json:"firstName,omitempty"`
	LastName                  string        `json:"lastName,omitempty"`
	DateOfBirth               string        `json:"dateOfBirth,omitempty"`
	Password                  string        `json:"password,omitempty"`
	Addresses                 []ctAddress   `json:"addresses"`
	DefaultShippingAddressID  string        `json:"defaultShippingAddressId,omitempty"`
	DefaultBillingAddressID   string        `json:"defaultBillingAddressId,omitempty"`
	ShippingAddressIDs        []string      `json:"shippingAddressIds"`
	BillingAddressIDs         []string      `json:"billingAddressIds"`
	IsEmailVerified           bool          `json:"isEmailVerified"`
	CustomerGroupAssignments  []interface{} `json:"customerGroupAssignments"`
	Stores                    []interface{} `json:"stores"`
	AuthenticationMode        string        `json:"authenticationMode"`
}

type auditInfo struct {
	ClientID         string `json:"clientId"`
	IsPlatformClient bool   `json:"isPlatformClient"`
}

type ctAddress struct {
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

var auditDefaults = auditInfo{
	ClientID:         "G-q8-RwsnGEU-laJdMCAWR6Z",
	IsPlatformClient: false,
}

func toCTCustomer(c domain.Customer) ctCustomer {
	created := c.CreatedAt
	if created.IsZero() {
		created = time.Now().UTC()
	}
	addresses := make([]ctAddress, 0, len(c.Addresses))
	for _, a := range c.Addresses {
		addresses = append(addresses, ctAddress{
			ID:         a.ID,
			FirstName:  a.FirstName,
			LastName:   a.LastName,
			Country:    a.Country,
			StreetName: a.StreetName,
			PostalCode: a.PostalCode,
			City:       a.City,
			Email:      a.Email,
			Department: a.Department,
		})
	}

	shipping := c.ShippingAddressIDs
	if shipping == nil {
		shipping = []string{}
	}
	billing := c.BillingAddressIDs
	if billing == nil {
		billing = []string{}
	}

	return ctCustomer{
		ID:                        c.ID,
		Version:                   1,
		VersionModifiedAt:         created,
		LastMessageSequenceNumber: 1,
		CreatedAt:                 created,
		LastModifiedAt:            created,
		LastModifiedBy:            auditDefaults,
		CreatedBy:                 auditDefaults,
		Email:                     c.Email,
		FirstName:                 c.FirstName,
		LastName:                  c.LastName,
		DateOfBirth:               c.DateOfBirth,
		Password:                  maskPassword(c.PasswordHash),
		Addresses:                 addresses,
		DefaultShippingAddressID:  c.DefaultShippingAddressID,
		DefaultBillingAddressID:   c.DefaultBillingAddressID,
		ShippingAddressIDs:        shipping,
		BillingAddressIDs:         billing,
		IsEmailVerified:           false,
		CustomerGroupAssignments:  []interface{}{},
		Stores:                    []interface{}{},
		AuthenticationMode:        "Password",
	}
}

func maskPassword(hash string) string {
	if hash == "" {
		return ""
	}
	enc := base64.StdEncoding.EncodeToString([]byte(hash))
	if len(enc) >= 4 {
		return "****" + enc[len(enc)-4:]
	}
	return "****"
}
