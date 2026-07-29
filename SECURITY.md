# Security Policy

## Supported versions

Only the latest version on the `main` branch is currently supported.

## Reporting a vulnerability

Do not open a public issue for a security vulnerability. Contact the repository owner through GitHub with reproduction details, impact, and a suggested mitigation. Do not include passwords, JWTs, production `.env` files, or database dumps in the report.

Tracklume is an MVP. Before production use, configure a long random `JWT_SECRET`, restrict `CORS_ALLOWED_ORIGINS`, keep PostgreSQL private, and store deployment credentials only in GitHub/VPS secret stores.
