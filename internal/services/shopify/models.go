package shopify

import (
	"time"
)

// Product represents a Shopify product
type Product struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	BodyHTML    string    `json:"body_html"`
	Vendor      string    `json:"vendor"`
	ProductType string    `json:"product_type"`
	Handle      string    `json:"handle"`
	Status      string    `json:"status"`
	Tags        string    `json:"tags"`
	Variants    []Variant `json:"variants"`
	Images      []Image   `json:"images"`
	Options     []Option  `json:"options"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	PublishedAt *time.Time `json:"published_at"`
}

// Variant represents a product variant
type Variant struct {
	ID                   int64     `json:"id"`
	ProductID            int64     `json:"product_id"`
	Title                string    `json:"title"`
	Price                string    `json:"price"`
	Sku                  string    `json:"sku"`
	Position             int       `json:"position"`
	InventoryPolicy      string    `json:"inventory_policy"`
	CompareAtPrice       *string   `json:"compare_at_price"`
	FulfillmentService   string    `json:"fulfillment_service"`
	InventoryManagement  string    `json:"inventory_management"`
	Option1              *string   `json:"option1"`
	Option2              *string   `json:"option2"`
	Option3              *string   `json:"option3"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
	Taxable              bool      `json:"taxable"`
	Barcode              *string   `json:"barcode"`
	Grams                int       `json:"grams"`
	ImageID              *int64    `json:"image_id"`
	Weight               float64   `json:"weight"`
	WeightUnit           string    `json:"weight_unit"`
	InventoryItemID      int64     `json:"inventory_item_id"`
	InventoryQuantity    int       `json:"inventory_quantity"`
	OldInventoryQuantity int       `json:"old_inventory_quantity"`
	RequiresShipping     bool      `json:"requires_shipping"`
	AdminGraphQLAPIID    string    `json:"admin_graphql_api_id"`
}

// Image represents a product image
type Image struct {
	ID         int64     `json:"id"`
	ProductID  int64     `json:"product_id"`
	Position   int       `json:"position"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Alt        *string   `json:"alt"`
	Width      int       `json:"width"`
	Height     int       `json:"height"`
	Src        string    `json:"src"`
	VariantIDs []int64   `json:"variant_ids"`
	AdminGraphQLAPIID string `json:"admin_graphql_api_id"`
}

// Option represents a product option
type Option struct {
	ID       int64    `json:"id"`
	ProductID int64   `json:"product_id"`
	Name     string   `json:"name"`
	Position int      `json:"position"`
	Values   []string `json:"values"`
}

// Shop represents shop information
type Shop struct {
	ID                        int64     `json:"id"`
	Name                      string    `json:"name"`
	Email                     string    `json:"email"`
	Domain                    string    `json:"domain"`
	Province                  string    `json:"province"`
	Country                   string    `json:"country"`
	Address1                  string    `json:"address1"`
	Zip                       string    `json:"zip"`
	City                      string    `json:"city"`
	Source                    string    `json:"source"`
	Phone                     string    `json:"phone"`
	Latitude                  float64   `json:"latitude"`
	Longitude                 float64   `json:"longitude"`
	PrimaryLocale             string    `json:"primary_locale"`
	Address2                  *string   `json:"address2"`
	CreatedAt                 time.Time `json:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at"`
	CountryCode               string    `json:"country_code"`
	CountryName               string    `json:"country_name"`
	Currency                  string    `json:"currency"`
	CustomerEmail             string    `json:"customer_email"`
	Timezone                  string    `json:"timezone"`
	IanaTimezone              string    `json:"iana_timezone"`
	ShopOwner                 string    `json:"shop_owner"`
	MoneyFormat               string    `json:"money_format"`
	MoneyWithCurrencyFormat   string    `json:"money_with_currency_format"`
	WeightUnit                string    `json:"weight_unit"`
	ProvinceCode              string    `json:"province_code"`
	TaxesIncluded             bool      `json:"taxes_included"`
	AutoConfigureTaxInclusivity bool    `json:"auto_configure_tax_inclusivity"`
	TaxShipping               bool      `json:"tax_shipping"`
	CountyTaxes               bool      `json:"county_taxes"`
	PlanDisplayName           string    `json:"plan_display_name"`
	PlanName                  string    `json:"plan_name"`
	HasDiscounts              bool      `json:"has_discounts"`
	HasGiftCards              bool      `json:"has_gift_cards"`
	MyshopifyDomain           string    `json:"myshopify_domain"`
	GoogleAppsDomain          *string   `json:"google_apps_domain"`
	GoogleAppsLoginEnabled    bool      `json:"google_apps_login_enabled"`
	MoneyInEmailsFormat       string    `json:"money_in_emails_format"`
	MoneyWithCurrencyInEmailsFormat string `json:"money_with_currency_in_emails_format"`
	EligibleForPayments       bool      `json:"eligible_for_payments"`
	RequiresExtraPaymentsAgreement bool `json:"requires_extra_payments_agreement"`
	PasswordEnabled           bool      `json:"password_enabled"`
	HasStorefront             bool      `json:"has_storefront"`
	EligibleForCardReaderGiveaway bool  `json:"eligible_for_card_reader_giveaway"`
	Finances                  bool      `json:"finances"`
	PrimaryLocationID         int64     `json:"primary_location_id"`
	CookieConsentLevel         string    `json:"cookie_consent_level"`
	VisitorTrackingConsent    string    `json:"visitor_tracking_consent"`
	CheckoutAPISupported      bool      `json:"checkout_api_supported"`
	MultiLocationEnabled      bool      `json:"multi_location_enabled"`
	SetupRequired             bool      `json:"setup_required"`
	PreLaunchEnabled          bool      `json:"pre_launch_enabled"`
	EnabledPresentmentCurrencies []string `json:"enabled_presentment_currencies"`
}

// ProductsResponse represents the response from products API
type ProductsResponse struct {
	Products []Product `json:"products"`
	Link     *string   `json:"link,omitempty"`
}

// WebhookPayload represents a Shopify webhook payload
type WebhookPayload struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	BodyHTML    string    `json:"body_html"`
	Vendor      string    `json:"vendor"`
	ProductType string    `json:"product_type"`
	Handle      string    `json:"handle"`
	Status      string    `json:"status"`
	Tags        string    `json:"tags"`
	Variants    []Variant `json:"variants"`
	Images      []Image   `json:"images"`
	Options     []Option  `json:"options"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	PublishedAt *time.Time `json:"published_at"`
}
