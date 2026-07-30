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
   - `PROD_APP_DIR`: absolute path deployment, misalnya `/srv/apps/tracklume-api`.

5. Setelah image pertama dipush, ubah package `ghcr.io/basith-08/tracklume-api` menjadi public, atau login-kan VPS ke GHCR dengan token `read:packages`.

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

Siapkan directory deployment. User pada `PROD_DEPLOY_USER` harus memiliki write access ke directory ini:

```bash
sudo mkdir -p /srv/apps/tracklume-api
sudo chown -R "$USER":"$USER" /srv/apps/tracklume-api
```

Workflow akan meng-copy `compose.yaml` dan `.env.example` ke directory tersebut. Pada deployment pertama, jika `.env` belum ada, workflow membuat `.env` dari `.env.example` lalu berhenti agar Anda dapat mengisi konfigurasi production secara manual. Jalankan ulang workflow setelah selesai mengedit `.env`. Deployment berikutnya tidak menimpa `.env`.

`.env` production minimal berisi:

```env
APP_ENV=production
APP_PORT=8080
APP_BASE_URL=https://api-tracklume.domain-anda
IMAGE_TAG=<commit-sha>
GHCR_OWNER=basith-08
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

Setelah migration berhasil, buat akun platform superadmin secara manual:

```bash
read -rsp "Superadmin password: " ADMIN_BOOTSTRAP_PASSWORD; echo
docker compose run --rm --no-deps \
  -e ADMIN_BOOTSTRAP_PASSWORD="$ADMIN_BOOTSTRAP_PASSWORD" \
  --entrypoint /app/tracklume-admin api create \
  --email admin@example.com --name "Tracklume Admin"
unset ADMIN_BOOTSTRAP_PASSWORD
```

API tidak membuat superadmin otomatis. Endpoint `/api/v1/admin/*` hanya dapat dipakai oleh akun ini; akun regular tetap dibuat melalui endpoint register.

Jika instance ini memang ingin menyediakan demo publik, jalankan seeder secara manual setelah superadmin dibuat:

```bash
docker compose run --rm --no-deps \
  -e ALLOW_DEMO_SEED=true \
  --entrypoint /app/tracklume-seed api
```

Seeder production memerlukan `ALLOW_DEMO_SEED=true` sebagai konfirmasi eksplisit dan membuat:

```text
owner@tracklume.local  / Password123!
member@tracklume.local / Password123!
```

Jangan menjalankan langkah ini pada instance yang menyimpan data privat atau tenant sebenarnya.

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
