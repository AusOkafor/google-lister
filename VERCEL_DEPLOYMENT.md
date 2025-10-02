# 🚀 Vercel Deployment Guide for Shopify Integration

## Prerequisites
- Vercel account (free tier available)
- GitHub repository with your code
- Production PostgreSQL database

## Step 1: Deploy to Vercel

### 1.1 Connect to Vercel
1. Go to [vercel.com](https://vercel.com)
2. Sign in with GitHub
3. Click "New Project"
4. Import your GitHub repository

### 1.2 Configure Build Settings
```
Framework Preset: Other
Root Directory: backend
Build Command: go build -o main cmd/api/main.go
Output Directory: .
Install Command: go mod download
```

## Step 2: Set Environment Variables

In Vercel Dashboard > Settings > Environment Variables, add:

```
DATABASE_URL=postgresql://username:password@host:port/database?schema=public
API_PORT=8080
API_HOST=0.0.0.0
JWT_SECRET=your-production-jwt-secret
ENCRYPTION_KEY=your-32-byte-encryption-key
SHOPIFY_CLIENT_ID=2787c88ee709ae377cd7288794ccb40d
SHOPIFY_CLIENT_SECRET=your-shopify-client-secret
SHOPIFY_WEBHOOK_SECRET=your-shopify-webhook-secret
ENV=production
LOG_LEVEL=info
```

## Step 3: Update Shopify Partner Dashboard

After deployment, you'll get a URL like:
```
https://your-app-name.vercel.app
```

### 3.1 Update App URLs
```
App URL: https://your-app-name.vercel.app
Allowed redirection URL(s): 
  - https://your-app-name.vercel.app/api/v1/shopify/callback
```

### 3.2 Update Webhooks
```
Webhook URL: https://your-app-name.vercel.app/api/v1/shopify/webhook
Events:
  - products/create
  - products/update
  - products/delete
  - app/uninstalled
```

## Step 4: Test the Integration

### 4.1 Test OAuth Flow
```
POST https://your-app-name.vercel.app/api/v1/shopify/install
Content-Type: application/json

{
  "shop_domain": "austus-themes",
  "redirect_uri": "https://your-app-name.vercel.app/api/v1/shopify/callback"
}
```

### 4.2 Test Webhooks
```
POST https://your-app-name.vercel.app/api/v1/shopify/webhook
Content-Type: application/json
X-Shopify-Topic: products/create
X-Shopify-Shop-Domain: austus-themes.myshopify.com

{
  "id": 123,
  "title": "Test Product",
  "body_html": "<p>Test Description</p>",
  "vendor": "Test Vendor",
  "product_type": "Test Type",
  "handle": "test-product",
  "status": "active",
  "tags": "test, sample",
  "variants": [...],
  "images": [...]
}
```

## Step 5: Production Database

### 5.1 Recommended Databases
- **Neon** (free tier available)
- **Supabase** (free tier available)
- **Railway** (free tier available)
- **PlanetScale** (free tier available)

### 5.2 Database Setup
1. Create a PostgreSQL database
2. Get the connection string
3. Add to Vercel environment variables
4. Run database migrations

## Benefits of Vercel Deployment

✅ **Public URL** - Shopify can reach your app
✅ **HTTPS** - Required for Shopify webhooks
✅ **Auto-scaling** - Handles traffic spikes
✅ **Easy deployment** - Git push to deploy
✅ **Environment variables** - Secure credential storage
✅ **Free tier** - Perfect for development

## Next Steps

1. **Deploy to Vercel** using this guide
2. **Update Partner Dashboard** with the new URLs
3. **Test the OAuth flow** with the real store
4. **Sync products** from the store
5. **Test webhooks** for real-time updates

Your Shopify integration will work perfectly with Vercel! 🎯
