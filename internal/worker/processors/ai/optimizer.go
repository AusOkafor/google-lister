package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"lister/internal/config"
	"lister/internal/logger"
)

type Optimizer struct {
	config *config.Config
	logger *logger.Logger
}

// OpenAI API structures
type OpenAIRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIResponse struct {
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Message Message `json:"message"`
}

// SEO Enhancement structures
type SEOEnhancement struct {
	SEOTitle       string   `json:"seo_title"`
	SEODescription string   `json:"seo_description"`
	Keywords       []string `json:"keywords"`
	MetaKeywords   string   `json:"meta_keywords"`
	AltText        string   `json:"alt_text"`
	SchemaMarkup   string   `json:"schema_markup"`
}

func New(cfg *config.Config, logger *logger.Logger) *Optimizer {
	return &Optimizer{
		config: cfg,
		logger: logger,
	}
}

func (o *Optimizer) OptimizeTitle(product interface{}) (string, error) {
	o.logger.Debug("Optimizing title for product: %+v", product)

	// Convert product to JSON for AI processing
	productJSON, err := json.Marshal(product)
	if err != nil {
		return "", fmt.Errorf("failed to marshal product: %v", err)
	}

	// Create AI prompt for title optimization
	prompt := fmt.Sprintf(`
You are an expert e-commerce SEO specialist. Optimize this product title for maximum search visibility and click-through rates.

Product data: %s

Requirements:
- Keep under 60 characters
- Include primary keywords
- Make it compelling and click-worthy
- Ensure it's SEO-optimized
- Maintain brand appeal

Return ONLY the optimized title, no explanations.
`, string(productJSON))

	optimizedTitle, err := o.callOpenAI(prompt)
	if err != nil {
		o.logger.Error("AI title optimization failed, using fallback: %v", err)
		// Fallback to simple optimization
		if title, ok := product.(map[string]interface{})["title"].(string); ok {
			if len(title) > 60 {
				return title[:57] + "...", nil
			}
			return title, nil
		}
		return "Optimized Product Title", nil
	}

	return strings.TrimSpace(optimizedTitle), nil
}

func (o *Optimizer) OptimizeDescription(product interface{}) (string, error) {
	o.logger.Debug("Optimizing description for product: %+v", product)

	// Convert product to JSON for AI processing
	productJSON, err := json.Marshal(product)
	if err != nil {
		return "", fmt.Errorf("failed to marshal product: %v", err)
	}

	// Create AI prompt for description optimization
	prompt := fmt.Sprintf(`
You are an expert e-commerce copywriter. Create an SEO-optimized product description that converts browsers into buyers.

Product data: %s

Requirements:
- Keep under 160 characters for meta description
- Include primary keywords naturally
- Highlight key benefits and features
- Create urgency and desire
- Use persuasive language
- Include a call-to-action

Return ONLY the optimized description, no explanations.
`, string(productJSON))

	optimizedDescription, err := o.callOpenAI(prompt)
	if err != nil {
		o.logger.Error("AI description optimization failed, using fallback: %v", err)
		// Fallback to simple optimization
		if desc, ok := product.(map[string]interface{})["description"].(string); ok {
			if len(desc) > 160 {
				return desc[:157] + "...", nil
			}
			return desc, nil
		}
		return "High-quality product with excellent features and great value.", nil
	}

	return strings.TrimSpace(optimizedDescription), nil
}

func (o *Optimizer) SuggestCategory(product interface{}) (string, error) {
	// TODO: Implement AI category suggestion
	// This would use ML to:
	// - Predict the best Google product category
	// - Suggest required attributes
	// - Validate against channel requirements

	o.logger.Debug("Suggesting category for product: %+v", product)

	// For now, return a default category
	return "Electronics > Audio & Video", nil
}

func (o *Optimizer) SuggestGTIN(product interface{}) (string, error) {
	o.logger.Debug("Suggesting GTIN for product: %+v", product)

	// Convert product to JSON for AI processing
	productJSON, err := json.Marshal(product)
	if err != nil {
		return "", fmt.Errorf("failed to marshal product: %v", err)
	}

	// Create AI prompt for GTIN suggestion
	prompt := fmt.Sprintf(`
You are an expert in product identification. Analyze this product and suggest a GTIN (Global Trade Item Number) if possible.

Product data: %s

Requirements:
- Return only the GTIN if you can determine it
- If no GTIN can be determined, return empty string
- GTIN should be 8, 12, 13, or 14 digits
- Consider product type, brand, and attributes

Return ONLY the GTIN or empty string, no explanations.
`, string(productJSON))

	gtin, err := o.callOpenAI(prompt)
	if err != nil {
		o.logger.Error("AI GTIN suggestion failed: %v", err)
		return "", nil
	}

	return strings.TrimSpace(gtin), nil
}

// EnhanceProductSEO - Comprehensive SEO enhancement using AI
func (o *Optimizer) EnhanceProductSEO(product interface{}) (*SEOEnhancement, error) {
	o.logger.Debug("Enhancing SEO for product: %+v", product)

	// Convert product to JSON for AI processing
	productJSON, err := json.Marshal(product)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal product: %v", err)
	}

	// Create comprehensive AI prompt for SEO enhancement
	prompt := fmt.Sprintf(`
You are an expert e-commerce SEO specialist. Analyze this product and provide comprehensive SEO optimization.

Product data: %s

Provide a JSON response with the following structure:
{
  "seo_title": "Optimized title under 60 characters",
  "seo_description": "Meta description under 160 characters",
  "keywords": ["keyword1", "keyword2", "keyword3"],
  "meta_keywords": "keyword1, keyword2, keyword3",
  "alt_text": "Descriptive alt text for product images",
  "schema_markup": "JSON-LD structured data for the product"
}

Requirements:
- SEO title: Under 60 characters, keyword-rich, compelling
- SEO description: Under 160 characters, persuasive, includes CTA
- Keywords: 5-10 relevant keywords from title, category, brand
- Alt text: Descriptive, includes product name and key features
- Schema markup: Valid JSON-LD for Product type with name, description, brand, category

Return ONLY the JSON response, no explanations.
`, string(productJSON))

	response, err := o.callOpenAI(prompt)
	if err != nil {
		o.logger.Error("AI SEO enhancement failed, using fallback: %v", err)
		return o.createFallbackSEO(product), nil
	}

	// Parse AI response
	var enhancement SEOEnhancement
	if err := json.Unmarshal([]byte(response), &enhancement); err != nil {
		o.logger.Error("Failed to parse AI SEO response, using fallback: %v", err)
		return o.createFallbackSEO(product), nil
	}

	return &enhancement, nil
}

// callOpenAI - Make API call to OpenAI
func (o *Optimizer) callOpenAI(prompt string) (string, error) {
	if o.config.OpenAIAPIKey == "" {
		return "", fmt.Errorf("OpenAI API key not configured")
	}

	client := &http.Client{Timeout: 30 * time.Second}

	request := OpenAIRequest{
		Model:       "gpt-3.5-turbo",
		Temperature: 0.7,
		MaxTokens:   500,
		Messages: []Message{
			{
				Role:    "system",
				Content: "You are an expert e-commerce SEO specialist and copywriter.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.config.OpenAIAPIKey)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI API error: %s", string(body))
	}

	var openAIResp OpenAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return openAIResp.Choices[0].Message.Content, nil
}

// createFallbackSEO - Create fallback SEO when AI fails
func (o *Optimizer) createFallbackSEO(product interface{}) *SEOEnhancement {
	// Extract basic product info
	title := "Product"
	category := "General"
	vendor := "Brand"
	description := "High-quality product"

	if p, ok := product.(map[string]interface{}); ok {
		if t, exists := p["title"].(string); exists {
			title = t
		}
		if c, exists := p["product_type"].(string); exists {
			category = c
		}
		if v, exists := p["vendor"].(string); exists {
			vendor = v
		}
		if d, exists := p["description"].(string); exists {
			description = d
		}
	}

	// Create fallback SEO
	seoTitle := title
	if len(seoTitle) > 60 {
		seoTitle = seoTitle[:57] + "..."
	}

	seoDescription := description
	if len(seoDescription) > 160 {
		seoDescription = seoDescription[:157] + "..."
	} else if seoDescription == "" {
		seoDescription = fmt.Sprintf("Shop %s online. High-quality %s from %s. Fast shipping and great customer service.", title, category, vendor)
	}

	keywords := []string{
		strings.ToLower(title),
		strings.ToLower(category),
		strings.ToLower(vendor),
		"online shopping",
		"buy online",
	}

	if category != "" {
		keywords = append(keywords, strings.ToLower(category)+" for sale")
	}

	return &SEOEnhancement{
		SEOTitle:       seoTitle,
		SEODescription: seoDescription,
		Keywords:       keywords,
		MetaKeywords:   strings.Join(keywords, ", "),
		AltText:        fmt.Sprintf("%s - %s product from %s", title, category, vendor),
		SchemaMarkup:   fmt.Sprintf(`{"@context":"https://schema.org","@type":"Product","name":"%s","description":"%s","brand":{"@type":"Brand","name":"%s"},"category":"%s"}`, title, description, vendor, category),
	}
}
