package shopify

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"lister/internal/config"
	"lister/internal/logger"
)

type OAuthService struct {
	config *config.Config
	logger *logger.Logger
}

func NewOAuthService(cfg *config.Config, logger *logger.Logger) *OAuthService {
	return &OAuthService{
		config: cfg,
		logger: logger,
	}
}

// GenerateAuthURL creates the Shopify OAuth authorization URL
func (s *OAuthService) GenerateAuthURL(shopDomain string, redirectURI string) (string, string, error) {
	// Generate a secure state parameter
	state, err := s.generateState()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate state: %w", err)
	}

	// Build the authorization URL with comprehensive scopes
	scopes := "read_products,write_products,read_product_listings,write_product_listings," +
		"read_inventory,write_inventory,read_locations," +
		"read_files,write_files," +
		"read_product_tags,write_product_tags,read_collections,write_collections," +
		"read_product_variants,write_product_variants,read_pricing," +
		"read_analytics,read_reports," +
		"read_orders,write_orders,read_fulfillments,write_fulfillments," +
		"read_shop,read_shopify_payments_payouts," +
		"read_apps,write_apps"

	// Clean the shop domain (remove .myshopify.com if present)
	cleanDomain := shopDomain
	if strings.HasSuffix(shopDomain, ".myshopify.com") {
		cleanDomain = strings.TrimSuffix(shopDomain, ".myshopify.com")
	}
	
	authURL := fmt.Sprintf(
		"https://%s.myshopify.com/admin/oauth/authorize?client_id=%s&scope=%s&redirect_uri=%s&state=%s",
		cleanDomain,
		s.config.ShopifyClientID,
		scopes,
		url.QueryEscape(redirectURI),
		state,
	)

	return authURL, state, nil
}

// ExchangeCodeForToken exchanges the authorization code for an access token
func (s *OAuthService) ExchangeCodeForToken(shopDomain, code string) (*TokenResponse, error) {
	// Prepare the request
	tokenURL := fmt.Sprintf("https://%s.myshopify.com/admin/oauth/access_token", shopDomain)

	data := url.Values{}
	data.Set("client_id", s.config.ShopifyClientID)
	data.Set("client_secret", s.config.ShopifyClientSecret)
	data.Set("code", code)

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status: %d", resp.StatusCode)
	}

	// Parse the response
	var tokenResp TokenResponse
	if err := s.parseJSONResponse(resp, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return &tokenResp, nil
}

// ValidateWebhook validates the Shopify webhook signature
func (s *OAuthService) ValidateWebhook(payload []byte, signature, secret string) bool {
	// Implement HMAC validation
	// This is a simplified version - in production, use proper HMAC validation
	return true // TODO: Implement proper HMAC validation
}

// generateState generates a cryptographically secure random state
func (s *OAuthService) generateState() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// parseJSONResponse parses JSON response (simplified for now)
func (s *OAuthService) parseJSONResponse(resp *http.Response, target interface{}) error {
	// TODO: Implement proper JSON parsing
	return nil
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
}
