# Tracklume API

Tracklume adalah backend REST API untuk membantu tim mencatat, mengatur, dan memantau pekerjaan dalam sebuah project. Tracklume menangani tiga jenis pekerjaan utama: task, bug, dan feature request.

Project ini dirancang sebagai backend terpisah yang dapat digunakan oleh web app, mobile app, atau frontend lain melalui HTTP dan JSON. Fokusnya adalah menyediakan fondasi issue tracker yang sederhana, aman, dan mudah di-deploy ke VPS.

**Tracklume — Track tasks, bugs, and product ideas clearly.**

Kelola tugas, bug, dan ide produk dengan lebih terarah.

Repository: `github.com/Basith-08/tracklume-api`

## Apa yang disediakan

- Authentication berbasis JWT: register, login, profil pengguna, dan perubahan password.
- Project management: project, anggota, role owner/admin/member/viewer, archive, dan authorization berbasis membership.
- Issue tracking: task, bug, feature, status Kanban, priority, assignee, reporter, due date, pencarian, filter, sorting, dan pagination.
- Issue activity: riwayat perubahan penting pada issue seperti status, priority, assignee, due date, dan penghapusan.
- Project dashboard: ringkasan issue aktif, distribusi status/priority/type, overdue, due dalam tujuh hari, issue terbaru, dan progress.
- Operational endpoints: healthcheck, readiness database, structured logging, CORS, request ID, timeout, recovery, dan rate limit auth.

## Alur penggunaan

1. Pengguna melakukan register atau login dan memperoleh access token.
2. Pengguna membuat project; otomatis menjadi owner project tersebut.
3. Owner menambahkan anggota dan memberikan role sesuai kebutuhan.
4. Member membuat issue dan mengelolanya melalui status Kanban.
5. Dashboard dan activity endpoint digunakan frontend untuk menampilkan progres serta riwayat pekerjaan.

Tracklume tidak menyediakan UI frontend. Frontend terpisah cukup menggunakan base URL API, mengirim `Authorization: Bearer <access_token>`, dan mengikuti response envelope yang dijelaskan pada dokumentasi API.

## Teknologi dan arsitektur

- Go, chi, pgx/v5, PostgreSQL, JWT, bcrypt, validator, dan `slog`.
- Alur request: handler → service → repository → PostgreSQL.
- `internal/auth`, `internal/project`, `internal/issue`, dan `internal/dashboard` memisahkan domain; `internal/middleware` menangani request ID, logging, CORS, auth, timeout, recovery, dan rate limit.
- Migration SQL berada di `migrations/`; executable ada di `cmd/api`, `cmd/migrate`, dan `cmd/seed`.

## Requirement dan local run

Go 1.25+, Docker Compose, dan PostgreSQL 16+ diperlukan. `.env` tidak di-commit dan tidak dibaca otomatis oleh Go; salin template lalu export isinya ke shell:

Dengan PostgreSQL lokal:

```bash
cp .env.example .env
# Sesuaikan DB_HOST=localhost jika PostgreSQL berjalan di host.
set -a; . ./.env; set +a
export DATABASE_URL='postgres://tracklume:tracklume@localhost:5432/tracklume?sslmode=disable'
export JWT_SECRET='local-only-secret-with-at-least-32-characters'
go run ./cmd/migrate up
go run ./cmd/seed
go run ./cmd/api
```

Jika memakai Makefile, `make run`, `make migrate-up`, `make migrate-down`, dan `make seed` otomatis membaca `.env`. Cek konfigurasi dengan `docker compose config`; Compose akan gagal jelas jika `.env` belum dibuat atau variable image/domain belum diisi.

Dengan Compose, isi `.env` dan jalankan `docker compose up -d db`. PostgreSQL menggunakan named volume Docker `tracklume-api-db-data` dan publish loopback `127.0.0.1:5432`, sehingga command Go dari host dapat memakai `DB_HOST=localhost`; port database tidak dipublish ke interface eksternal. Volume lama dari template sebelumnya tidak disentuh. Jalankan migration melalui `docker compose run --rm --no-deps --entrypoint /app/tracklume-migrate api up` setelah image tersedia. Untuk development dari source, command Makefile berikut tersedia:

```text
make run | build | test | lint | migrate-up | migrate-down | seed | docker-build
```

Seeder bersifat idempotent dan hanya untuk local demo. Credential demo: `owner@tracklume.local` / `Password123!` dan `member@tracklume.local` / `Password123!`.

## Environment variables

`APP_ENV`, `APP_PORT` (default `8080`), `APP_BASE_URL`, `JWT_SECRET`, `JWT_EXPIRATION`, `CORS_ALLOWED_ORIGINS`, `REQUEST_TIMEOUT`, `SHUTDOWN_TIMEOUT`, `BODY_LIMIT_BYTES`, `AUTH_RATE_LIMIT_REQUESTS`, dan `AUTH_RATE_LIMIT_WINDOW` mengatur API. Database dapat memakai `DATABASE_URL`, atau `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`, `DB_HOST`, `DB_PORT`, dan `DB_SSLMODE`. Compose production juga membutuhkan `IMAGE_TAG`, `GHCR_OWNER`, dan `PLATFORM_DOMAIN`.

Jangan commit `.env`. Production wajib memakai JWT secret acak; aplikasi menolak secret pendek ketika `APP_ENV=production`.

## API

Base prefix: `/api/v1`. Public: `GET /health`, `GET /ready`, `POST /api/v1/auth/register`, `POST /api/v1/auth/login`. Endpoint lain memakai `Authorization: Bearer <access_token>`:

- User: `GET/PATCH /api/v1/me`, `PUT /api/v1/me/password`.
- Project: `GET/POST /api/v1/projects`, `GET/PATCH/DELETE /api/v1/projects/{projectID}`.
- Member: `GET/POST /api/v1/projects/{projectID}/members`, `PATCH/DELETE .../members/{userID}`.
- Issue: `GET/POST /api/v1/projects/{projectID}/issues`, `GET/PATCH/DELETE .../issues/{issueID}`.
- Workflow: `PATCH .../issues/{issueID}/status`, `PATCH .../issues/{issueID}/position`, `GET .../issues/{issueID}/activities`.
- Dashboard: `GET /api/v1/projects/{projectID}/dashboard`.

Response tunggal berbentuk `{ "data": ... }`, collection memakai `{ "data": [...], "meta": {...} }`, dan error memakai `{ "error": { "code", "message", "fields", "request_id" } }`. Filter issue mendukung `search`, `status`, `priority`, `type`, `assignee_id`, `reporter_id`, `due_before`, `due_after`, `sort`, `page`, dan `per_page` (maksimum 100). Detail kontrak ada di [openapi.yaml](openapi.yaml).

Panduan public repository, GitHub Actions secrets, DNS, Traefik, dan VPS ada di [DEPLOYMENT.md](DEPLOYMENT.md).

Swagger UI tersedia di `GET /docs`, misalnya `http://localhost:8080/docs`. Spesifikasi mentah tersedia di `GET /openapi.yaml`. Halaman Swagger UI memakai asset CDN `unpkg.com`, jadi browser memerlukan akses internet untuk memuat tampilannya.

## Authorization dan keputusan desain

Project hanya terlihat oleh member. Owner memiliki akses penuh; admin mengelola issue/member tetapi tidak owner; member mengelola issue; viewer hanya membaca. Hanya owner yang dapat mengarsipkan project. `DELETE /projects/{id}` melakukan soft archive, sedangkan `DELETE /issues/{id}` melakukan soft delete agar activity history tetap tersedia. Assignee harus member project.

Sequence issue memakai row counter dan `SELECT ... FOR UPDATE`-equivalent atomic update dalam transaction; identifier menjadi `PROJECTKEY-N`. Activity dibuat untuk create, perubahan title/status/priority/assignee/due date, dan delete. Dashboard mengecualikan issue cancelled dari progress serta aman ketika denominator nol.

## Migration, health, dan deployment

Migration dijalankan eksplisit dengan `go run ./cmd/migrate up|down`. API melakukan retry koneksi database saat startup. `/health` hanya memeriksa process, sedangkan `/ready` melakukan ping database. Shutdown menangani SIGINT/SIGTERM, timeout, dan penutupan pool.

Dockerfile memakai build multi-stage, binary statis, runtime Alpine non-root, dan membawa migration files. Compose mempertahankan network Traefik `edge`, private `tracklume-api-internal`, volume PostgreSQL persisten, hostname `api-tracklume.${PLATFORM_DOMAIN}`, dan port `8080`. Workflow GitHub Actions menjalankan tidy verification, vet, test PostgreSQL, build, push image ke GHCR, migration eksplisit, deploy SSH, dan health verification. Production menjalankan image yang sudah dibangun, bukan source.

Frontend dapat memakai `APP_BASE_URL` atau hostname production sebagai base URL, menyimpan access token secara aman, mengirim Bearer token, dan membaca envelope response di atas. Tidak ada refresh token server-side pada MVP.

## Testing dan batas MVP

`go test ./...` menjalankan unit/handler tests. Integration test PostgreSQL dijalankan jika `TEST_DATABASE_URL` di-set; workflow menyediakan PostgreSQL service dan menjalankan migration lebih dahulu. `go vet ./...` adalah lint baseline.

Di luar MVP: workspace multi-tenant, OAuth, refresh token kompleks, invitation email, comment, attachment, notification, WebSocket, subtask, sprint, time tracking, custom status/field, billing, dan microservice.
