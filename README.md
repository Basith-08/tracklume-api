# Tracklume API

**Tracklume — Track tasks, bugs, and product ideas clearly.**

Kelola tugas, bug, dan ide produk dengan lebih terarah.

Tracklume adalah backend REST API untuk mencatat, mengatur, dan memantau pekerjaan dalam sebuah project. API ini mendukung task, bug, dan feature request, lalu menyediakan status Kanban, assignment, activity history, dan project dashboard.

Tracklume dirancang sebagai service backend terpisah yang dapat digunakan oleh web app, mobile app, atau frontend lain melalui HTTP dan JSON.

Repository: [`Basith-08/tracklume-api`](https://github.com/Basith-08/tracklume-api)

## Daftar isi

- [Fitur](#fitur)
- [Teknologi dan arsitektur](#teknologi-dan-arsitektur)
- [Menjalankan local](#menjalankan-local)
- [API dan dokumentasi](#api-dan-dokumentasi)
- [Konfigurasi](#konfigurasi)
- [Authorization](#authorization)
- [Database dan migration](#database-dan-migration)
- [Testing](#testing)
- [Deployment](#deployment)
- [Batas MVP](#batas-mvp)

## Fitur

- **Authentication** — register, login, profil pengguna, perubahan profil, dan perubahan password dengan JWT.
- **Project management** — membuat project, mengelola anggota, role owner/admin/member/viewer, archive, dan akses berbasis membership.
- **Issue tracking** — task, bug, feature, status Kanban, priority, assignee, reporter, due date, pencarian, filter, sorting, dan pagination.
- **Issue activity** — riwayat perubahan title, status, priority, assignee, due date, pembuatan, dan penghapusan issue.
- **Dashboard** — total issue aktif, distribusi status/priority/type, overdue, due dalam tujuh hari, issue terbaru, dan progress percentage.
- **Platform administration** — superadmin dapat melihat statistik pengguna, last login, aktivitas akun, menonaktifkan/mengaktifkan akun, serta memulihkan akun yang dihapus secara soft delete.
- **Operational readiness** — healthcheck, readiness database, structured logging, request ID, CORS, timeout, panic recovery, body limit, dan rate limit auth.

Tracklume tidak menyediakan UI frontend. Frontend terpisah menggunakan base URL API, mengirim `Authorization: Bearer <access_token>`, dan membaca response envelope yang dijelaskan di bawah.

## Teknologi dan arsitektur

| Area | Teknologi |
| --- | --- |
| Language | Go 1.25+ |
| HTTP router | chi |
| Database | PostgreSQL 16+ |
| PostgreSQL driver | pgx/v5 |
| Authentication | JWT |
| Password hashing | bcrypt |
| Validation | go-playground/validator |
| Logging | `log/slog` |
| Deployment | Docker multi-stage, GHCR, GitHub Actions, Traefik |

Alur request utama:

```text
HTTP request
    ↓
Handler → Service → Repository → PostgreSQL
```

Tanggung jawab layer dibuat sederhana dan eksplisit:

- **Handler** menangani HTTP, parsing, validasi awal, dan response.
- **Service** menangani business rule, authorization, dan transaction boundary.
- **Repository** menangani query PostgreSQL terparameterisasi.
- **Middleware** menangani request ID, logging, recovery, CORS, authentication, timeout, dan rate limit.

Struktur folder utama:

```text
cmd/api       # HTTP server
cmd/migrate   # database migration runner
internal/     # domain, service, repository, middleware, config
migrations/   # SQL migration up/down
build.env     # shared Docker/CI build contract
openapi.yaml  # API contract
tests/        # PostgreSQL integration tests
```

## Menjalankan local

### Requirement

- Go 1.25 atau lebih baru
- Docker Engine dan Docker Compose plugin
- PostgreSQL 16 atau lebih baru jika tidak memakai container database

### Quick start dengan PostgreSQL Compose

> `.env` tidak di-commit. Untuk command Go yang berjalan dari host, gunakan `DB_HOST=localhost`. Compose otomatis menggunakan hostname internal `db` untuk container API.

```bash
cp .env.example .env
docker compose up -d db
make migrate-up
make run
```

API tersedia di `http://localhost:8080`.

Verifikasi:

```bash
curl http://localhost:8080/health
curl http://localhost:8080/ready
```

### Menjalankan tanpa Makefile

```bash
set -a
. ./.env
set +a

go run ./cmd/migrate up
go run ./cmd/api
```

### Perintah Makefile

```bash
make run
make build
make test
make lint
make migrate-up
make migrate-down
make admin-create
make docker-build
```

`make run`, `make migrate-up`, dan `make migrate-down` membaca `.env` secara otomatis.
`make admin-create` membutuhkan `ADMIN_BOOTSTRAP_PASSWORD`; email dan nama dapat diubah dengan `ADMIN_EMAIL` dan `ADMIN_NAME`.

```bash
ADMIN_BOOTSTRAP_PASSWORD='change-this-password' \
  make admin-create ADMIN_EMAIL=admin@example.com ADMIN_NAME="Tracklume Admin"
```

## API dan dokumentasi

Base prefix API adalah `/api/v1`.

### Public endpoints

```text
GET  /health
GET  /ready
POST /api/v1/auth/register
POST /api/v1/auth/login
```

### Authenticated endpoints

Gunakan header berikut pada endpoint yang membutuhkan login:

```http
Authorization: Bearer <access_token>
```

```text
GET|PATCH /api/v1/me
PUT       /api/v1/me/password

GET|POST  /api/v1/projects
GET|PATCH|DELETE /api/v1/projects/{projectID}

GET|POST  /api/v1/projects/{projectID}/members
PATCH|DELETE /api/v1/projects/{projectID}/members/{userID}

GET|POST  /api/v1/projects/{projectID}/issues
GET|PATCH|DELETE /api/v1/projects/{projectID}/issues/{issueID}
PATCH     /api/v1/projects/{projectID}/issues/{issueID}/status
PATCH     /api/v1/projects/{projectID}/issues/{issueID}/position
GET       /api/v1/projects/{projectID}/issues/{issueID}/activities

GET       /api/v1/projects/{projectID}/dashboard

GET       /api/v1/admin/overview
GET       /api/v1/admin/users?status=active&page=1&per_page=20
GET       /api/v1/admin/users/{userID}
PATCH     /api/v1/admin/users/{userID}/status
DELETE    /api/v1/admin/users/{userID}                 # soft delete
POST      /api/v1/admin/users/{userID}/restore
```

Endpoint `/admin/*` hanya dapat digunakan oleh akun dengan `platform_role=superadmin`. Status akun dapat berupa `active`, `inactive`, atau `deleted`. Menonaktifkan akun memerlukan alasan; akun yang dinonaktifkan tidak dapat login atau memakai access token yang masih aktif. Penghapusan akun bersifat soft delete agar dapat dipulihkan.

Filter issue mendukung:

```text
search, status, priority, type, assignee_id, reporter_id,
due_before, due_after, sort, page, per_page
```

`per_page` dibatasi maksimum `100`.

### Response format

Response tunggal:

```json
{
  "data": {
    "id": "uuid"
  }
}
```

Response collection:

```json
{
  "data": [],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 0,
    "total_pages": 0
  }
}
```

Response error:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Request validation failed",
    "fields": {
      "title": ["Title is required"]
    },
    "request_id": "uuid"
  }
}
```

Dokumentasi tersedia di:

- Swagger UI: [http://localhost:8080/docs](http://localhost:8080/docs)
- OpenAPI: [openapi.yaml](openapi.yaml)
- Raw OpenAPI endpoint: `GET /openapi.yaml`

Swagger UI menggunakan asset CDN `unpkg.com`.

## Konfigurasi

Salin `.env.example` menjadi `.env`, lalu sesuaikan nilainya.

| Variable | Keterangan |
| --- | --- |
| `APP_ENV` | `development` atau `production` |
| `APP_PORT` | Port HTTP API, default `8080` |
| `APP_BASE_URL` | Base URL API |
| `DATABASE_URL` | DSN PostgreSQL opsional; diprioritaskan jika diisi |
| `POSTGRES_USER` | User database |
| `POSTGRES_PASSWORD` | Password database |
| `POSTGRES_DB` | Nama database |
| `DB_HOST` | `localhost` untuk command host, `db` di Compose |
| `DB_PORT` | Default `5432` |
| `DB_SSLMODE` | `disable` untuk local; sesuaikan production |
| `JWT_SECRET` | Secret JWT; wajib panjang dan random di production |
| `JWT_EXPIRATION` | Contoh `1h` |
| `CORS_ALLOWED_ORIGINS` | Origin frontend, bukan wildcard production |
| `REQUEST_TIMEOUT` | Contoh `15s` |
| `SHUTDOWN_TIMEOUT` | Contoh `10s` |
| `IMAGE_TAG` | Tag image GHCR untuk Compose production |
| `GHCR_OWNER` | Owner image lowercase, contoh `basith-08` |
| `PLATFORM_DOMAIN` | Domain tanpa `https://`, contoh `example.com` |

Production menolak `JWT_SECRET` yang terlalu pendek. Jangan commit `.env` atau memasukkan password, JWT secret, dan private key ke repository.

## Authorization

Resource project hanya terlihat oleh anggota project.

| Role | Hak akses |
| --- | --- |
| `owner` | Akses penuh, termasuk archive project dan pengelolaan owner policy |
| `admin` | Mengelola issue dan anggota, tetapi tidak dapat menghapus owner |
| `member` | Membuat dan mengubah issue |
| `viewer` | Hanya membaca |

Aturan tambahan:

- Pembuat project otomatis menjadi member dengan role `owner`.
- Hanya owner yang dapat mengarsipkan project.
- Assignee issue harus menjadi anggota project yang sama.
- Resource dari project lain tidak dapat diakses hanya dengan menebak UUID.
- `DELETE /projects/{id}` melakukan archive/soft delete.
- `DELETE /issues/{id}` melakukan soft delete agar activity history tetap tersedia.

### Platform admin

Akun superadmin tidak dibuat otomatis dari runtime API. Setelah migration, buat secara eksplisit:

```bash
ADMIN_BOOTSTRAP_PASSWORD='change-this-password' \
  go run ./cmd/admin create --email admin@example.com --name "Tracklume Admin"
```

Pada VPS:

```bash
read -rsp "Superadmin password: " ADMIN_BOOTSTRAP_PASSWORD; echo
docker compose run --rm --no-deps \
  -e ADMIN_BOOTSTRAP_PASSWORD="$ADMIN_BOOTSTRAP_PASSWORD" \
  --entrypoint /app/tracklume-admin api create \
  --email admin@example.com --name "Tracklume Admin"
unset ADMIN_BOOTSTRAP_PASSWORD
```

Command tersebut idempotent berdasarkan email: akun yang sudah ada akan dipromosikan menjadi superadmin dan akun soft-deleted akan dipulihkan. Jangan menaruh password bootstrap di `.env` atau repository.

## Database dan migration

Migration dijalankan secara eksplisit:

```bash
go run ./cmd/migrate up
go run ./cmd/migrate down
```

Tabel utama:

```text
users
projects
project_members
project_issue_counters
issues
issue_activities
```

Migration `003` menambahkan `users.deleted_at` untuk soft delete akun tanpa merusak database yang sudah menjalankan migration `002`.

Nomor issue dibuat per project secara atomic sehingga identifier berbentuk `PROJECTKEY-1`, `PROJECTKEY-2`, dan seterusnya tanpa race condition.

## Testing

```bash
go test ./...
go vet ./...
```

Unit dan handler test dapat dijalankan tanpa database. PostgreSQL integration test dijalankan jika `TEST_DATABASE_URL` tersedia; untuk menjalankannya secara lokal, siapkan PostgreSQL lalu jalankan migration terlebih dahulu.

Docker image dapat dibangun dengan:

```bash
make docker-build
```

`build.env` adalah sumber konfigurasi bersama untuk image build, runtime image, build command, start command, healthcheck, dan test command. Image menggunakan multi-stage build, binary static, runtime minimal, dan user non-root.

## Deployment

Deployment production menggunakan image yang sudah dibangun di GHCR, bukan build source di server.

Workflow GitHub Actions melakukan:

1. Test container berbasis `build.env` menjalankan `go vet` dan `go test`.
2. Build dan push image dengan tag commit SHA ke GHCR.
3. SSH ke VPS menggunakan `PROD_DEPLOY_KEY`.
4. Masuk ke `/srv/apps/tracklume-api`, memperbarui `IMAGE_TAG` di `.env`, lalu menjalankan `docker compose pull` dan `docker compose up -d`.

`.env` dan `compose.yaml` harus disiapkan manual di VPS. Tidak ada akun demo atau user production yang dibuat otomatis. Pengguna dapat mendaftar melalui endpoint register, sedangkan superadmin dibuat manual menggunakan `make admin-create`.

Health endpoints:

```text
GET /health  # process hidup
GET /ready   # process dan database siap
```

## Batas MVP

Fitur berikut sengaja belum termasuk dalam MVP:

- Workspace multi-tenant
- OAuth dan refresh token kompleks
- Email invitation
- Comment, attachment, notification, dan email
- WebSocket atau real-time collaboration
- Subtask, sprint, dan time tracking
- Custom status dan custom field
- Billing dan audit log organisasi
- Microservice

## License

Tracklume API dirilis dengan lisensi MIT. Lihat [LICENSE](LICENSE).
