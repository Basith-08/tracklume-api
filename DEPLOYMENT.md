# Tracklume public repository dan VPS deployment

Repository: `git@github.com:Basith-08/tracklume-api.git`

## Yang perlu disiapkan

### GitHub

1. Buat repository `Basith-08/tracklume-api` dan set visibility menjadi **Public**.
2. Pastikan GitHub Actions aktif.
3. Di repository settings, buka **Actions → General → Workflow permissions** dan izinkan workflow membaca repository. Workflow deployment membutuhkan `packages: write`, yang sudah dideklarasikan di file workflow.
4. Siapkan Actions secrets:

   - `PROD_HOST`: IP atau hostname VPS.
   - `PROD_DEPLOY_USER`: user SSH deployment, bukan root jika memungkinkan.
   - `PROD_DEPLOY_KEY`: private SSH key untuk user tersebut.

5. Setelah image pertama dipush, ubah package `ghcr.io/Basith-08/tracklume-api` menjadi public, atau login-kan VPS ke GHCR dengan token `read:packages`.

### Domain dan VPS

Siapkan:

- VPS Linux dengan Docker Engine dan Docker Compose plugin.
- DNS record `api-tracklume.<domain-anda>` menuju IP VPS.
- Firewall hanya membuka TCP `22`, `80`, dan `443` sesuai kebutuhan.
- Traefik yang sudah berjalan dan memiliki external Docker network `edge`.
- Resolver TLS `letsencrypt` dan middleware `secure-headers@file` pada Traefik.

Buat network jika belum ada:

```bash
docker network create edge
```

Siapkan directory deployment:

```bash
sudo mkdir -p /srv/apps/tracklume-api
sudo chown -R "$USER":"$USER" /srv/apps/tracklume-api
```

Copy `compose.yaml` dan `.env` production ke directory tersebut. `.env` production minimal berisi:

```env
APP_ENV=production
APP_PORT=8080
APP_BASE_URL=https://api-tracklume.domain-anda
IMAGE_TAG=<commit-sha>
GHCR_OWNER=Basith-08
PLATFORM_DOMAIN=domain-anda

POSTGRES_USER=tracklume
POSTGRES_PASSWORD=<password-database-acak>
POSTGRES_DB=tracklume
DB_HOST=db
DB_PORT=5432
DB_SSLMODE=disable
DATABASE_URL=

JWT_SECRET=<secret-acak-minimal-32-karakter>
JWT_EXPIRATION=1h
CORS_ALLOWED_ORIGINS=https://frontend.domain-anda
REQUEST_TIMEOUT=15s
SHUTDOWN_TIMEOUT=10s
BODY_LIMIT_BYTES=1048576
AUTH_RATE_LIMIT_REQUESTS=10
AUTH_RATE_LIMIT_WINDOW=1m
```

Jangan commit file ini. Compose menggunakan named volume `tracklume-api-db-data` dan PostgreSQL tetap hanya berada pada network internal; binding `127.0.0.1:5432` tidak membuka database ke internet.

## Manual first deployment

Jika image sudah tersedia di GHCR:

```bash
cd /srv/apps/tracklume-api
docker compose pull api
docker compose up -d db
docker compose run --rm --no-deps --entrypoint /app/tracklume-migrate api up
docker compose up -d api
docker compose ps
```

Verifikasi:

```bash
curl -fsS https://api-tracklume.domain-anda/health
curl -fsS https://api-tracklume.domain-anda/ready
```

Push ke `main` akan menjalankan test, build, push image SHA ke GHCR, migration, dan deployment SSH. Rollback dilakukan melalui **Actions → Deploy Tracklume API → Run workflow**, isi input `sha` dengan commit SHA yang ingin dijalankan.

## Push pertama dari local

`.git` pada workspace template ini belum berisi repository Git aktif. Setelah review:

```bash
git init
git branch -M main
git remote add origin git@github.com:Basith-08/tracklume-api.git
git add .
git commit -m "build: initialize Tracklume API"
git push -u origin main
```

Sebelum push, pastikan:

```bash
git status --short
rg -n -i 'password|secret|token|DATABASE_URL' --hidden --glob '!.git/**' .env README.md DEPLOYMENT.md
```

Output tersebut hanya boleh berisi placeholder/documentation, bukan credential production. `.env` production harus tetap untracked.
