# Delcom Blog API — Praktikum 9

Backend REST API Delcom Blog menggunakan Go, Gin, GORM, PostgreSQL, dan JWT.

## Persiapan lokal

1. Instal Go sesuai versi pada `go.mod` dan PostgreSQL.
2. Salin `.env.example` menjadi `.env`.
3. Sesuaikan konfigurasi database dan rahasia aplikasi di `.env`.

PowerShell:

```powershell
Copy-Item .env.example .env
notepad .env
```

## Menjalankan

```powershell
go mod tidy
go run .
```

API berjalan pada port yang ditentukan oleh `APP_PORT` (default `8081`).

- Health check: `/health`
- Daftar kategori: `/api/public/categories`
- Daftar artikel publik: `/api/public/posts`
- Dokumentasi API: `/swagger/index.html`

## Akun admin

Akun admin awal menggunakan `ADMIN_USERNAME` dan `ADMIN_PASSWORD` dari `.env`. Gunakan password yang kuat sebelum deployment.

## Pengujian

```powershell
go test ./...
```

## Fitur

- Registrasi, login, logout, dan profil pengguna.
- Role admin dan member.
- CRUD artikel, kategori, slug, publikasi, dan estimasi waktu baca.
- Statistik penulis dan administrasi.
- Pembuatan draf artikel dengan integrasi AI.
- PostgreSQL, migrasi otomatis, CORS, JWT, dan response JSON.
