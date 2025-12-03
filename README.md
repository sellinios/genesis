# Genesis

Dynamic Business Platform - Everything is Data, Data Defines Everything.

## Quick Start

```bash
# Clone and run
git clone https://github.com/sellinios/genesis.git
cd genesis
docker-compose up -d
```

Server starts at `http://localhost:8090`

## Configuration

Environment variables (set in docker-compose.yml or pass directly):

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | postgres | Database host |
| `DB_PORT` | 5432 | Database port |
| `DB_USER` | genesis | Database user |
| `DB_PASSWORD` | genesis_secret_change_me | Database password |
| `DB_NAME` | genesis | Database name |
| `PORT` | 8090 | Server port |
| `JWT_SECRET` | (auto-generated) | JWT signing key |
| `CORS_ALLOWED_ORIGINS` | localhost:3000,localhost:8080 | Allowed CORS origins |

## API Endpoints

- **Auth**: `/auth/login`, `/auth/register`, `/auth/refresh`
- **Admin**: `/admin/tenants`, `/admin/modules`, `/admin/entities`
- **Data**: `/api/data/:entity`
- **Health**: `/api/health`

## Build

```bash
# Build Docker image
docker build -t genesis .

# Build binary
go build -o genesis ./cmd/server
```

## License

MIT
