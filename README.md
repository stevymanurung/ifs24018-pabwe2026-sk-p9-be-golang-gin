# ifs24018-pabwe2026-sk-p9-be-golang-gin — Delcom Blog API

Backend REST API Delcom Blog menggunakan Go, Gin, GORM, PostgreSQL, JWT, dan Clean Architecture Feature-First.

## Persiapan

1. Instal Go 1.27.4 (sesuai direktif `go.mod`) dan PostgreSQL 17.
2. Buat database PostgreSQL bernama `pabwe_p9_blog` melalui pgAdmin atau `psql`.
3. Salin `.env.example` menjadi `.env`.
4. Sesuaikan `DB_PASSWORD` dengan password PostgreSQL di laptop.

### PowerShell

```powershell
Copy-Item .env.example .env
notepad .env
```

Konfigurasi database default:

```env
DB_CONNECTION=pgsql
DB_HOST=127.0.0.1
DB_PORT=5432
DB_DATABASE=pabwe_p9_blog
DB_USERNAME=postgres
DB_PASSWORD=postgres
DB_SSLMODE=disable
```

Pastikan service PostgreSQL sedang berjalan. Backend akan menjalankan migrasi tabel dan seed akun awal saat pertama kali dijalankan.

## Menjalankan backend

```powershell
go mod tidy
go run ./cmd/server/main.go
```

- API: http://localhost:8081
- Health check: http://localhost:8081/
- Swagger UI: http://localhost:8081/swagger/index.html

## Akun awal

- Admin: username `admin`, password `admin`
- Member demo: username `member`, password `member`

Ganti password default sebelum aplikasi dipublikasikan.

## Pengujian

```powershell
go test -v -cover ./internal/...
```

## Fitur

- Autentikasi register, login, logout, dan profil dengan JWT.
- Role admin dan member.
- CRUD artikel, kategori, slug unik, status publikasi, dan estimasi waktu baca.
- Statistik penulis serta halaman administrasi.
- AI article draft dengan integrasi Search Engine dan OpenAI-compatible endpoint.
- Swagger/OpenAPI, CORS, rate limiter, dan response JSON standar.
