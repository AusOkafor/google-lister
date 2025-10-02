# 🏪 Shopify App Setup Guide

## Prerequisites
- Shopify Partner account
- Access to Partner Dashboard
- Your app running on `http://localhost:8081`

## Step 1: Create Shopify App

### 1.1 Access Partner Dashboard
1. Go to [partners.shopify.com](https://partners.shopify.com)
2. Sign in with your Shopify Partner account
3. If you don't have a Partner account, create one first

### 1.2 Create New App
1. Click **"Apps"** in the left sidebar
2. Click **"Create app"**
3. Choose **"Create app manually"**
4. Fill in the app details:
   - **App name:** `Lister - Product Listing Tool`
   - **App URL:** `http://localhost:8081`
   - **Allowed redirection URL(s):** `http://localhost:8081/api/v1/shopify/callback`

## Step 2: Configure App Settings

### 2.1 App Setup
```
App URL: http://localhost:8081
Allowed redirection URL(s): 
  - http://localhost:8081/api/v1/shopify/callback
  - https://yourdomain.com/api/v1/shopify/callback (for production)
```

### 2.2 App Proxy (Optional)
```
Subpath prefix: lister
Subpath: api
URL: http://localhost:8081/api/v1/shopify/proxy
```

### 2.3 Admin API Access Scopes
Enable these scopes for full product listing functionality:

#### Product Management:
- `read_products` - Read product information
- `write_products` - Create/update/delete products
- `read_product_listings` - Read product listings
- `write_product_listings` - Manage product listings

#### Inventory Management:
- `read_inventory` - Read inventory levels
- `write_inventory` - Update inventory levels
- `read_locations` - Read store locations

#### Media Management:
- `read_files` - Read product images/files
- `write_files` - Upload product images

#### Product Organization:
- `read_product_tags` - Read product tags
- `write_product_tags` - Manage product tags
- `read_collections` - Read product collections
- `write_collections` - Manage product collections

#### Pricing & Variants:
- `read_product_variants` - Read product variants
- `write_product_variants` - Manage product variants
- `read_pricing` - Read pricing information

#### Analytics & Reporting:
- `read_analytics` - Read store analytics
- `read_reports` - Read store reports

#### Order Management:
- `read_orders` - Read order information
- `write_orders` - Update order status
- `read_fulfillments` - Read fulfillment data
- `write_fulfillments` - Update fulfillment status

#### Store Information:
- `read_shop` - Read shop information
- `read_shopify_payments_payouts` - Read payment data

#### App Management:
- `read_apps` - Read app information
- `write_apps` - Manage app settings

## Step 3: Configure Webhooks

### 3.1 Add Webhooks
In the **Webhooks** section, add:

```
Webhook URL: http://localhost:8081/api/v1/shopify/webhook
Events:
  - products/create
  - products/update
  - products/delete
  - app/uninstalled
```

### 3.2 Webhook Format
```
Format: JSON
API Version: 2023-10
```

## Step 4: Get Your Credentials

After creating the app, you'll receive:

1. **Client ID** (API Key) - Copy this
2. **Client Secret** (API Secret Key) - Copy this  
3. **Webhook Secret** - Copy this

## Step 5: Update Environment Variables

Update your `.env` file with the credentials:

```env
# Shopify App Credentials
SHOPIFY_CLIENT_ID="your-client-id-from-partner-dashboard"
SHOPIFY_CLIENT_SECRET="your-client-secret-from-partner-dashboard"
SHOPIFY_WEBHOOK_SECRET="your-webhook-secret-from-partner-dashboard"
```

## Step 6: Test the Integration

### 6.1 Install App in Development Store
1. Create a development store in Partner Dashboard
2. Install your app in the development store
3. Test the OAuth flow

### 6.2 Test Webhooks
Use ngrok or similar tool to expose your local server:
```bash
ngrok http 8081
```
Then update your webhook URLs in Partner Dashboard to use the ngrok URL.

## Step 7: Production Deployment

### 7.1 Update URLs for Production
- App URL: `https://yourdomain.com`
- Redirect URL: `https://yourdomain.com/api/v1/shopify/callback`
- Webhook URL: `https://yourdomain.com/api/v1/shopify/webhook`

### 7.2 Submit for Review
Once your app is ready:
1. Complete all required sections in Partner Dashboard
2. Submit for Shopify review
3. Wait for approval before publishing

## Troubleshooting

### Common Issues:
1. **CORS errors:** Make sure your server has CORS middleware
2. **Webhook not receiving:** Check ngrok URL and webhook configuration
3. **OAuth errors:** Verify redirect URLs match exactly
4. **Permission errors:** Check API scopes in Partner Dashboard

### Testing Tools:
- **Shopify CLI:** For local development
- **ngrok:** For webhook testing
- **Postman:** For API testing
- **Shopify GraphQL Admin API:** For advanced queries

## Next Steps:
1. Complete the Partner Dashboard setup
2. Update your `.env` file with credentials
3. Test the OAuth flow
4. Test webhook reception
5. Deploy to production when ready
