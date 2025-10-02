package shopify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"lister/internal/logger"
)

type Client struct {
	shopDomain  string
	accessToken string
	httpClient  *http.Client
	logger      *logger.Logger
}

func NewClient(shopDomain, accessToken string, logger *logger.Logger) *Client {
	return &Client{
		shopDomain:  shopDomain,
		accessToken: accessToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// GetProducts fetches products from Shopify
func (c *Client) GetProducts(limit int, pageInfo string) (*ProductsResponse, error) {
	url := fmt.Sprintf("https://%s.myshopify.com/admin/api/2023-10/products.json", c.shopDomain)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication header
	req.Header.Set("X-Shopify-Access-Token", c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	// Add query parameters
	q := req.URL.Query()
	q.Set("limit", fmt.Sprintf("%d", limit))
	if pageInfo != "" {
		q.Set("page_info", pageInfo)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed: %d - %s", resp.StatusCode, string(body))
	}

	var productsResp ProductsResponse
	if err := json.NewDecoder(resp.Body).Decode(&productsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &productsResp, nil
}

// GetProduct fetches a single product by ID
func (c *Client) GetProduct(productID string) (*Product, error) {
	url := fmt.Sprintf("https://%s.myshopify.com/admin/api/2023-10/products/%s.json", c.shopDomain, productID)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Shopify-Access-Token", c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed: %d - %s", resp.StatusCode, string(body))
	}

	var productResp struct {
		Product Product `json:"product"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&productResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &productResp.Product, nil
}

// UpdateProduct updates a product in Shopify
func (c *Client) UpdateProduct(product *Product) error {
	url := fmt.Sprintf("https://%s.myshopify.com/admin/api/2023-10/products/%s.json", c.shopDomain, product.ID)
	
	payload := struct {
		Product Product `json:"product"`
	}{
		Product: *product,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal product: %w", err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Shopify-Access-Token", c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed: %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetShopInfo fetches shop information
func (c *Client) GetShopInfo() (*Shop, error) {
	url := fmt.Sprintf("https://%s.myshopify.com/admin/api/2023-10/shop.json", c.shopDomain)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Shopify-Access-Token", c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed: %d - %s", resp.StatusCode, string(body))
	}

	var shopResp struct {
		Shop Shop `json:"shop"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&shopResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &shopResp.Shop, nil
}
