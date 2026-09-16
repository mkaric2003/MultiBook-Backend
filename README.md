# MultiBook Backend

Production-oriented REST API for the MultiBook marketplace. It powers business management, discovery, stays and service reservations, availability, promotions, reviews, chat, notifications, and reporting for the Flutter mobile application.

## Getting Started

Requirements: Go 1.24+, Docker Desktop, and Firebase credentials for local token verification.

```bash
cp .env.example .env
docker compose up -d postgres
make migrate-up
make run
```

## Development

```bash
make test
make test-integration
make lint
```

## Documentation

See [PROJECT_DOCUMENTATION.md](PROJECT_DOCUMENTATION.md) for the complete architecture, API, security, development, and deployment documentation. The mobile application is available in the [MultiBook-MobileApp repository](https://github.com/mkaric2003/MultiBook-MobileApp).
