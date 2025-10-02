package shopify

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"lister/internal/models"
)

type Transformer struct{}

func NewTransformer() *Transformer {
	return &Transformer{}
}

// TransformProduct converts a Shopify product to our canonical format
func (t *Transformer) TransformProduct(shopifyProduct *Product) (*models.Product, error) {
	// Get the primary variant (first variant or the one with position 1)
	var primaryVariant *Variant
	for _, variant := range shopifyProduct.Variants {
		if variant.Position == 1 {
			primaryVariant = &variant
			break
		}
	}
	if primaryVariant == nil && len(shopifyProduct.Variants) > 0 {
		primaryVariant = &shopifyProduct.Variants[0]
	}

	if primaryVariant == nil {
		return nil, fmt.Errorf("no variants found for product %d", shopifyProduct.ID)
	}

	// Parse price
	price, err := strconv.ParseFloat(primaryVariant.Price, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid price format: %w", err)
	}

	// Transform images
	images := make([]string, len(shopifyProduct.Images))
	for i, img := range shopifyProduct.Images {
		images[i] = img.Src
	}

	// Transform variants
	variants := make([]models.ProductVariant, len(shopifyProduct.Variants))
	for i, variant := range shopifyProduct.Variants {
		variantPrice, _ := strconv.ParseFloat(variant.Price, 64)
		variants[i] = models.ProductVariant{
			ID:    fmt.Sprintf("%d", variant.ID),
			SKU:   variant.Sku,
			Price: variantPrice,
			Attributes: map[string]interface{}{
				"title":              variant.Title,
				"position":           variant.Position,
				"inventory_quantity": variant.InventoryQuantity,
				"weight":             variant.Weight,
				"weight_unit":        variant.WeightUnit,
				"barcode":            variant.Barcode,
			},
		}
	}

	// Transform shipping info
	shipping := &models.ShippingInfo{
		Weight: &primaryVariant.Weight,
		Dimensions: &models.Dimensions{
			Length: 0, // Shopify doesn't provide dimensions by default
			Width:  0,
			Height: 0,
			Unit:   "cm",
		},
	}

	// Transform custom labels from tags
	customLabels := []string{}
	if shopifyProduct.Tags != "" {
		customLabels = strings.Split(shopifyProduct.Tags, ",")
		for i, label := range customLabels {
			customLabels[i] = strings.TrimSpace(label)
		}
	}

	// Create metadata
	metadata := map[string]interface{}{
		"shopify_id":   shopifyProduct.ID,
		"handle":       shopifyProduct.Handle,
		"product_type": shopifyProduct.ProductType,
		"status":       shopifyProduct.Status,
		"created_at":   shopifyProduct.CreatedAt,
		"updated_at":   shopifyProduct.UpdatedAt,
		"published_at": shopifyProduct.PublishedAt,
	}

	// Determine availability
	availability := string(models.AvailabilityInStock)
	if primaryVariant.InventoryQuantity <= 0 {
		availability = string(models.AvailabilityOutOfStock)
	}

	// Create canonical product
	canonicalProduct := &models.Product{
		ExternalID:   fmt.Sprintf("shopify_%d", shopifyProduct.ID),
		SKU:          primaryVariant.Sku,
		Title:        shopifyProduct.Title,
		Description:  &shopifyProduct.BodyHTML,
		Brand:        &shopifyProduct.Vendor,
		Category:     &shopifyProduct.ProductType,
		Price:        price,
		Currency:     "USD", // Default currency, should be fetched from shop info
		Availability: availability,
		Images:       images,
		Variants:     variants,
		Shipping:     shipping,
		CustomLabels: customLabels,
		Metadata:     metadata,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	return canonicalProduct, nil
}

// TransformVariants converts Shopify variants to our format
func (t *Transformer) TransformVariants(shopifyVariants []Variant) ([]models.ProductVariant, error) {
	variants := make([]models.ProductVariant, len(shopifyVariants))

	for i, variant := range shopifyVariants {
		price, err := strconv.ParseFloat(variant.Price, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid price for variant %d: %w", variant.ID, err)
		}

		variants[i] = models.ProductVariant{
			ID:    fmt.Sprintf("%d", variant.ID),
			SKU:   variant.Sku,
			Price: price,
			Attributes: map[string]interface{}{
				"title":              variant.Title,
				"position":           variant.Position,
				"inventory_quantity": variant.InventoryQuantity,
				"weight":             variant.Weight,
				"weight_unit":        variant.WeightUnit,
				"barcode":            variant.Barcode,
				"option1":            variant.Option1,
				"option2":            variant.Option2,
				"option3":            variant.Option3,
			},
		}
	}

	return variants, nil
}

// TransformToShopify converts our canonical product back to Shopify format
func (t *Transformer) TransformToShopify(canonicalProduct *models.Product) (*Product, error) {
	// This would be used for updating products back to Shopify
	// Implementation depends on your specific needs
	return nil, fmt.Errorf("not implemented yet")
}

// ExtractGTIN extracts GTIN from product metadata
func (t *Transformer) ExtractGTIN(shopifyProduct *Product) *string {
	// Look for GTIN in variants' barcodes
	for _, variant := range shopifyProduct.Variants {
		if variant.Barcode != nil && *variant.Barcode != "" {
			// Basic GTIN validation (12-14 digits)
			barcode := *variant.Barcode
			if len(barcode) >= 12 && len(barcode) <= 14 {
				// Check if it's all digits
				isNumeric := true
				for _, char := range barcode {
					if char < '0' || char > '9' {
						isNumeric = false
						break
					}
				}
				if isNumeric {
					return &barcode
				}
			}
		}
	}
	return nil
}

// ExtractMPN extracts MPN from product metadata
func (t *Transformer) ExtractMPN(shopifyProduct *Product) *string {
	// Look for MPN in product title or tags
	// This is a simplified implementation
	if shopifyProduct.Tags != "" {
		tags := strings.Split(shopifyProduct.Tags, ",")
		for _, tag := range tags {
			tag = strings.TrimSpace(tag)
			// Simple heuristic: if tag looks like a part number
			if len(tag) > 3 && len(tag) < 20 {
				return &tag
			}
		}
	}
	return nil
}
