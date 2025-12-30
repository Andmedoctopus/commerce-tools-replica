package httpserver

import (
	"strconv"
	"strings"
	"time"

	"commercetools-replica/internal/domain"
)

type ctCart struct {
	Type                            string                    `json:"type"`
	ID                              string                    `json:"id"`
	Version                         int                       `json:"version"`
	VersionModifiedAt               time.Time                 `json:"versionModifiedAt"`
	LastMessageSequenceNumber       int                       `json:"lastMessageSequenceNumber,omitempty"`
	CreatedAt                       time.Time                 `json:"createdAt"`
	LastModifiedAt                  time.Time                 `json:"lastModifiedAt"`
	LastModifiedBy                  *ctActor                  `json:"lastModifiedBy,omitempty"`
	CreatedBy                       *ctActor                  `json:"createdBy,omitempty"`
	CustomerID                      string                    `json:"customerId,omitempty"`
	LineItems                       []ctLineItem              `json:"lineItems"`
	CartState                       string                    `json:"cartState"`
	TotalPrice                      ctPriceValue              `json:"totalPrice"`
	ShippingMode                    string                    `json:"shippingMode"`
	Shipping                        []interface{}             `json:"shipping"`
	CustomLineItems                 []interface{}             `json:"customLineItems"`
	DiscountCodes                   []interface{}             `json:"discountCodes"`
	DirectDiscounts                 []interface{}             `json:"directDiscounts"`
	InventoryMode                   string                    `json:"inventoryMode"`
	PriceRoundingMode               string                    `json:"priceRoundingMode"`
	TaxMode                         string                    `json:"taxMode"`
	TaxRoundingMode                 string                    `json:"taxRoundingMode"`
	TaxCalculationMode              string                    `json:"taxCalculationMode"`
	DeleteDaysAfterLastModification int                       `json:"deleteDaysAfterLastModification"`
	RefusedGifts                    []interface{}             `json:"refusedGifts"`
	Origin                          string                    `json:"origin"`
	ItemShippingAddresses           []interface{}             `json:"itemShippingAddresses"`
	DiscountTypeCombination         ctDiscountTypeCombination `json:"discountTypeCombination"`
	TotalLineItemQuantity           int                       `json:"totalLineItemQuantity,omitempty"`
}

type ctActor struct {
	ClientID         string `json:"clientId,omitempty"`
	IsPlatformClient bool   `json:"isPlatformClient"`
	Customer         *ctRef `json:"customer,omitempty"`
}

type ctDiscountTypeCombination struct {
	Type string `json:"type"`
}

type ctLineItem struct {
	ID                         string            `json:"id"`
	ProductID                  string            `json:"productId"`
	ProductKey                 string            `json:"productKey,omitempty"`
	ProductType                *ctProductType    `json:"productType,omitempty"`
	ProductSlug                map[string]string `json:"productSlug,omitempty"`
	Name                       map[string]string `json:"name"`
	Variant                    ctVariant         `json:"variant"`
	Price                      ctPrice           `json:"price"`
	Quantity                   int               `json:"quantity"`
	DiscountedPricePerQuantity []interface{}     `json:"discountedPricePerQuantity"`
	PerMethodTaxRate           []interface{}     `json:"perMethodTaxRate"`
	AddedAt                    time.Time         `json:"addedAt"`
	LastModifiedAt             time.Time         `json:"lastModifiedAt"`
	State                      []interface{}     `json:"state"`
	PriceMode                  string            `json:"priceMode"`
	LineItemMode               string            `json:"lineItemMode"`
	PriceRoundingMode          string            `json:"priceRoundingMode"`
	TotalPrice                 ctPriceValue      `json:"totalPrice"`
	TaxedPricePortions         []interface{}     `json:"taxedPricePortions"`
}

type ctProductType struct {
	TypeID  string `json:"typeId,omitempty"`
	ID      string `json:"id,omitempty"`
	Version int    `json:"version,omitempty"`
}

type cartLineSnapshot struct {
	ProductKey  string
	ProductName string
	SKU         string
	ProductSlug string
	Currency    string
	PriceCents  int64
	Images      []string
}

func toCTCart(cart domain.Cart, customer *domain.Customer) ctCart {
	state := strings.TrimSpace(cart.State)
	if state == "" {
		state = "Active"
	} else if strings.EqualFold(state, "active") {
		state = "Active"
	} else if strings.EqualFold(state, "deleted") {
		state = "Deleted"
	}

	customerID := ""
	if customer != nil {
		customerID = customer.ID
	} else if cart.CustomerID != nil {
		customerID = *cart.CustomerID
	}

	lineItems := make([]ctLineItem, 0, len(cart.Lines))
	totalQty := 0
	for _, line := range cart.Lines {
		snap := parseLineSnapshot(line.Snapshot)
		name := snap.ProductName
		if name == "" {
			name = snap.ProductKey
		}
		if name == "" {
			name = line.ProductID
		}
		slug := snap.ProductSlug
		if slug == "" {
			slug = snap.ProductKey
		}
		price := line.UnitPriceCents
		if snap.PriceCents > 0 {
			price = snap.PriceCents
		}
		currency := snap.Currency
		if currency == "" {
			currency = cart.Currency
		}

		images := imagesFromURLs(snap.Images)
		variant := ctVariant{
			ID:         1,
			SKU:        snap.SKU,
			Prices:     []ctPrice{{Value: ctPriceValue{Type: "centPrecision", CurrencyCode: currency, CentAmount: price, FractionDigits: 2}}},
			Images:     images,
			Assets:     []interface{}{},
			Attributes: []interface{}{},
		}

		var productSlug map[string]string
		if slug != "" {
			productSlug = map[string]string{"en": slug}
		}

		lineItems = append(lineItems, ctLineItem{
			ID:                         line.ID,
			ProductID:                  line.ProductID,
			ProductKey:                 snap.ProductKey,
			ProductSlug:                productSlug,
			Name:                       map[string]string{"en": name},
			Variant:                    variant,
			Price:                      ctPrice{Value: ctPriceValue{Type: "centPrecision", CurrencyCode: currency, CentAmount: price, FractionDigits: 2}},
			Quantity:                   line.Quantity,
			DiscountedPricePerQuantity: []interface{}{},
			PerMethodTaxRate:           []interface{}{},
			AddedAt:                    line.CreatedAt,
			LastModifiedAt:             line.CreatedAt,
			State:                      []interface{}{},
			PriceMode:                  "Platform",
			LineItemMode:               "Standard",
			PriceRoundingMode:          "HalfEven",
			TotalPrice: ctPriceValue{
				Type:           "centPrecision",
				CurrencyCode:   currency,
				CentAmount:     line.TotalCents,
				FractionDigits: 2,
			},
			TaxedPricePortions: []interface{}{},
		})
		totalQty += line.Quantity
	}

	actor := buildActor(customerID)
	totalPrice := ctPriceValue{
		Type:           "centPrecision",
		CurrencyCode:   cart.Currency,
		CentAmount:     cart.TotalCents,
		FractionDigits: 2,
	}

	out := ctCart{
		Type:                            "Cart",
		ID:                              cart.ID,
		Version:                         1,
		VersionModifiedAt:               cart.CreatedAt,
		LastMessageSequenceNumber:       1,
		CreatedAt:                       cart.CreatedAt,
		LastModifiedAt:                  cart.CreatedAt,
		LastModifiedBy:                  actor,
		CreatedBy:                       actor,
		CustomerID:                      customerID,
		LineItems:                       lineItems,
		CartState:                       state,
		TotalPrice:                      totalPrice,
		ShippingMode:                    "Single",
		Shipping:                        []interface{}{},
		CustomLineItems:                 []interface{}{},
		DiscountCodes:                   []interface{}{},
		DirectDiscounts:                 []interface{}{},
		InventoryMode:                   "None",
		PriceRoundingMode:               "HalfEven",
		TaxMode:                         "Platform",
		TaxRoundingMode:                 "HalfEven",
		TaxCalculationMode:              "LineItemLevel",
		DeleteDaysAfterLastModification: 90,
		RefusedGifts:                    []interface{}{},
		Origin:                          "Customer",
		ItemShippingAddresses:           []interface{}{},
		DiscountTypeCombination:         ctDiscountTypeCombination{Type: "Stacking"},
	}
	if totalQty > 0 {
		out.TotalLineItemQuantity = totalQty
	}
	return out
}

func buildActor(customerID string) *ctActor {
	if customerID == "" {
		return nil
	}
	return &ctActor{
		ClientID:         auditDefaults.ClientID,
		IsPlatformClient: false,
		Customer:         &ctRef{TypeID: "customer", ID: customerID},
	}
}

func parseLineSnapshot(raw map[string]interface{}) cartLineSnapshot {
	var out cartLineSnapshot
	if raw == nil {
		return out
	}
	if v, ok := raw["productKey"].(string); ok {
		out.ProductKey = v
	}
	if v, ok := raw["productName"].(string); ok {
		out.ProductName = v
	}
	if v, ok := raw["sku"].(string); ok {
		out.SKU = v
	}
	if v, ok := raw["productSlug"].(string); ok {
		out.ProductSlug = v
	}
	switch v := raw["priceCents"].(type) {
	case int64:
		out.PriceCents = v
	case int32:
		out.PriceCents = int64(v)
	case int:
		out.PriceCents = int64(v)
	case float64:
		out.PriceCents = int64(v)
	case string:
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			out.PriceCents = parsed
		}
	}
	if v, ok := raw["currency"].(string); ok {
		out.Currency = v
	}
	out.Images = parseImageList(raw["images"])
	return out
}

func parseImageList(raw interface{}) []string {
	if raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case []string:
		return v
	case []interface{}:
		var out []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func imagesFromURLs(urls []string) []ctImage {
	if len(urls) == 0 {
		return nil
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
