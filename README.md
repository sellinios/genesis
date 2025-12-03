# Genesis

Dynamic Business Platform - Everything is Data, Data Defines Everything.

## Quick Start

1. **Copy environment template:**
   ```bash
   cp .env.example .env
   ```

2. **Configure your settings in `.env`**

3. **Run:**
   ```bash
   docker-compose up -d
   ```

## Configuration

All settings are configured via environment variables. Copy `.env.example` to `.env` and adjust values for your environment.

### Required Variables

| Variable | Description |
|----------|-------------|
| `DB_HOST` | Database hostname |
| `DB_PORT` | Database port |
| `DB_USER` | Database username |
| `DB_PASSWORD` | Database password |
| `DB_NAME` | Database name |
| `JWT_SECRET` | Secret key for JWT signing (min 32 chars) |

### Optional Variables

| Variable | Description |
|----------|-------------|
| `PORT` | Server port (default: 8090) |
| `GIN_MODE` | Gin mode: debug/release |
| `CORS_ALLOWED_ORIGINS` | Comma-separated allowed origins |

## API Endpoints

- **Auth**: `/auth/login`, `/auth/register`, `/auth/refresh`, `/auth/logout`
- **Admin**: `/admin/tenants`, `/admin/modules`, `/admin/entities` (requires admin role)
- **Data**: `/api/data/:entity`
- **Health**: `/api/health`

## Build

```bash
# Docker
docker build -t genesis .

# Binary
go build -o genesis ./cmd/server
```

## Security Notes

- Change all default passwords before deploying
- Use strong JWT_SECRET (min 32 characters)
- Configure CORS_ALLOWED_ORIGINS for production
- Admin API requires authentication with admin role

## License

MIT
