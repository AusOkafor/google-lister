# Lister - Next-Gen Product Listing Platform

A developer-friendly, AI-assisted, insight-driven product feed platform that syncs product catalogs to multiple channels (Google, Bing, Meta, Pinterest, TikTok).

## Features

- **Smart Feed Optimizer**: AI-powered title and description optimization
- **Self-Healing Feed**: Auto-detect and fix common issues
- **Multi-Channel Support**: Unified feeds for Google, Bing, Meta, Pinterest, TikTok
- **Real-time Sync**: Event-driven updates via webhooks
- **A/B Testing**: Test feed variants and pick winners automatically
- **Actionable Diagnostics**: Plain-English error reporting with revenue impact
- **Developer-Friendly**: REST/GraphQL APIs, headless mode

## Architecture

```
[Connectors] --> [Ingestion API] --> [Event Bus] --> [Transform/Normalize] --> [Validation] --> [Feed Store]
                                                             |                    |
                                                             v                    v
                                                       [AI Optimizer]         [Exporters]
                                                             |                    |
                                                         [FeedVariants] --> [Channels]
                                                             |
                                                         [A/B Test Manager]
                                                             |
                                                     [Analytics / Insights]
```

## Tech Stack

- **Backend**: Go (Gin framework)
- **Database**: PostgreSQL with Prisma ORM
- **Cache**: Redis
- **Message Queue**: Kafka
- **AI/ML**: OpenAI/Anthropic APIs
- **Container**: Docker & Docker Compose

## Quick Start

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- Node.js (for Prisma)

### 1. Clone and Setup

```bash
git clone <repository>
cd backend
```

### 2. Environment Setup

```bash
cp env.example .env
# Edit .env with your configuration
```

### 3. Start Services

```bash
# Start Docker services (PostgreSQL, Redis, Kafka)
make docker-up

# Install dependencies
make deps

# Generate Prisma client
make generate
```

### 4. Run the Application

#### Windows (Batch)
```cmd
# Run API server
scripts\run-api.bat

# In another terminal, run worker
scripts\run-worker.bat
```

#### Windows (PowerShell)
```powershell
# Run API server
.\scripts\run-api.ps1

# In another terminal, run worker
.\scripts\run-worker.ps1
```

#### Linux/macOS
```bash
# Run API server
make run

# In another terminal, run worker
make run-worker
```

### 5. Access Services

- **API**: http://localhost:8080
- **Adminer (DB)**: http://localhost:8080
- **Redis Commander**: http://localhost:8081

## Development

### Project Structure

```
backend/
├── cmd/                    # Application entry points
│   ├── api/               # API server
│   └── worker/            # Background worker
├── internal/              # Private application code
│   ├── api/               # HTTP handlers and middleware
│   ├── config/            # Configuration management
│   ├── database/          # Database connection
│   ├── logger/            # Logging utilities
│   ├── models/            # Data models
│   ├── connectors/        # Platform connectors
│   └── worker/            # Background processing
├── prisma/                # Database schema
└── docker-compose.yml     # Local development services
```

### Available Commands

#### Windows (Batch Files)
```cmd
scripts\build.bat         # Build the application
scripts\run-api.bat        # Run API server
scripts\run-worker.bat     # Run worker
scripts\test.bat           # Run tests
scripts\clean.bat          # Clean build artifacts
scripts\docker-up.bat     # Start Docker services
scripts\docker-down.bat    # Stop Docker services
scripts\deps.bat           # Install dependencies
scripts\dev.bat            # Start development environment
```

#### Windows (PowerShell)
```powershell
.\scripts\build.ps1         # Build the application
.\scripts\run-api.ps1        # Run API server
.\scripts\run-worker.ps1     # Run worker
.\scripts\test.ps1           # Run tests
.\scripts\clean.ps1          # Clean build artifacts
.\scripts\docker-up.ps1      # Start Docker services
.\scripts\docker-down.ps1    # Stop Docker services
.\scripts\deps.ps1           # Install dependencies
.\scripts\dev.ps1            # Start development environment
```

#### Linux/macOS (Makefile)
```bash
make help          # Show available commands
make build         # Build the application
make run           # Run API server
make run-worker    # Run worker
make test          # Run tests
make clean         # Clean build artifacts
make docker-up     # Start Docker services
make docker-down   # Stop Docker services
make generate      # Generate Prisma client
make deps          # Install dependencies
make fmt           # Format code
make lint          # Lint code
make dev           # Start development environment
```

## API Endpoints

### Products
- `GET /api/v1/products` - List products
- `GET /api/v1/products/:id` - Get product
- `POST /api/v1/products` - Create product
- `PUT /api/v1/products/:id` - Update product
- `DELETE /api/v1/products/:id` - Delete product

### Connectors
- `GET /api/v1/connectors` - List connectors
- `GET /api/v1/connectors/:id` - Get connector
- `POST /api/v1/connectors` - Create connector
- `PUT /api/v1/connectors/:id` - Update connector
- `DELETE /api/v1/connectors/:id` - Delete connector
- `POST /api/v1/connectors/:id/sync` - Sync connector

### Channels
- `GET /api/v1/channels` - List channels
- `GET /api/v1/channels/:id` - Get channel
- `POST /api/v1/channels` - Create channel
- `PUT /api/v1/channels/:id` - Update channel
- `DELETE /api/v1/channels/:id` - Delete channel
- `POST /api/v1/channels/:id/sync` - Sync channel

### Issues
- `GET /api/v1/issues` - List issues
- `GET /api/v1/issues/:id` - Get issue
- `POST /api/v1/issues/:id/resolve` - Resolve issue

## Database Schema

The application uses Prisma with PostgreSQL. Key models:

- **Product**: Canonical product model
- **FeedVariant**: A/B test variants
- **Issue**: Diagnostics and errors
- **ABTest**: A/B testing data
- **Connector**: Platform connectors
- **Channel**: Export channels
- **Organization**: Multi-tenant support
- **User**: User management

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

MIT License
