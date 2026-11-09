# Noodu Login Service

REST API login berbasis Go, PostgreSQL, bcrypt, dan JWT. Struktur mengikuti pola sederhana: konfigurasi dan domain dipisah ke `internal/`, migrasi database berada di `migrations/`, dan executable pembuatan akun berada di `cmd/create-user/`.

## Kebutuhan

- Go 1.22 atau lebih baru
- PostgreSQL yang dapat diakses dari komputer pengembangan
- Extension DBCode di VS Code untuk menjalankan migrasi dan melihat tabel

## Menyiapkan database dengan DBCode

1. Pilih database `website_loan_banking` dan koneksi PostgreSQL yang sama dengan `DATABASE_URL`.
2. Pastikan tabel `public.users` sudah tersedia.
3. Jalankan `migrations/001_users.sql`, lalu `migrations/002_move_auth_users_to_users.sql` melalui DBCode.
4. Migrasi kedua memindahkan akun lama yang tidak bentrok dari `auth_users` ke `users`, kemudian menghapus `auth_users`. Jika menemukan bentrok ID, username, atau email, migrasi akan berhenti tanpa menghapus tabel lama.
5. Jangan simpan password database pada file migrasi atau commit connection string.

## Menjalankan service

Salin `.env.example` menjadi `.env`, lalu isi `DATABASE_URL` dengan connection string database PostgreSQL yang sama. Untuk layanan cloud, gunakan connection string resmi provider dan konfigurasi TLS yang mereka sarankan. Ganti `JWT_SECRET` dengan nilai acak minimal 32 karakter; file `.env` tidak dilacak Git.

```powershell
go mod download
go run ./cmd/create-user --username alice --email alice@example.com
go run .
```

Perintah `create-user` meminta password dua kali tanpa menampilkannya dan menyimpan hash bcrypt ke `public.users`. Password harus 12-72 byte. Jalankan backend dari folder proyek agar `.env` terbaca.

## Endpoint

- `GET /healthz`: proses hidup
- `GET /readyz`: koneksi PostgreSQL siap
- `POST /api/v1/auth/register`: membuat akun dan langsung mengembalikan access token
- `POST /api/v1/auth/login`: menerima JSON `{"email":"alice@example.com","password":"..."}`
- `GET /api/v1/auth/me`: membaca profil akun sendiri
- `PATCH /api/v1/auth/me`: mengubah `username` dan/atau `email` akun sendiri
- `DELETE /api/v1/auth/me`: menghapus permanen akun sendiri (hard delete)
- `DELETE /api/v1/auth/account`: hard delete dengan verifikasi email dan password, termasuk akun yang sudah soft-deleted

Semua endpoint `/api/v1/auth/me` memerlukan header `Authorization: Bearer <access_token>`. Token didapat dari respons register atau login. Belum ada endpoint untuk mendaftar atau menghapus semua user; akses tersebut perlu role admin.

### Thunder Client

Jalankan backend dengan `go run .`. Buat request baru di Thunder Client menggunakan URL `http://127.0.0.1:8080`. Untuk request dengan body, pilih **Body > JSON**. Untuk `/auth/me`, atur **Auth > Bearer** dan tempel `data.access_token` dari respons register atau login.

#### Health check

`GET http://127.0.0.1:8080/healthz`

Body: tidak ada.

Respons sukses:

```json
{"status":"ok"}
```

#### Database readiness

`GET http://127.0.0.1:8080/readyz`

Body: tidak ada.

Respons sukses:

```json
{"status":"ready"}
```

#### Register

`POST http://127.0.0.1:8080/api/v1/auth/register`

Body JSON:

```json
{
	"username": "alice",
	"email": "alice@example.com",
	"password": "GantiDenganPassword12!"
}
```

Username harus 3-50 karakter. Password harus 12-72 byte. Respons `201 Created` berisi user dan `data.access_token`.

#### Login

`POST http://127.0.0.1:8080/api/v1/auth/login`

Body JSON:

```json
{
	"email": "alice@example.com",
	"password": "GantiDenganPassword12!"
}
```

Gunakan kredensial akun yang sudah dibuat. Respons `200 OK` berisi `data.access_token`.

#### Get profil sendiri

`GET http://127.0.0.1:8080/api/v1/auth/me`

Auth: **Bearer Token** dari respons login/register. Body: tidak ada.

#### Update profil sendiri

`PATCH http://127.0.0.1:8080/api/v1/auth/me`

Auth: **Bearer Token**. Body JSON, kirim salah satu atau kedua field:

```json
{
	"username": "alice_baru",
	"email": "alice-baru@example.com"
}
```

#### Delete akun sendiri

`DELETE http://127.0.0.1:8080/api/v1/auth/me`

Auth: **Bearer Token**. Body: tidak ada. Respons sukses `204 No Content`. Ini hard delete: baris akun dihapus permanen dari `users`, sehingga email dan username bisa didaftarkan lagi. Tindakan ini tidak dapat dibatalkan.

#### Hapus akun tanpa Bearer

`DELETE http://127.0.0.1:8080/api/v1/auth/account`

Tidak memerlukan Bearer Token. Endpoint ini memverifikasi email dan password, sehingga bisa digunakan untuk menghapus akun yang sudah soft-deleted dan tidak bisa login. Pilih **Body > JSON**:

```json
{
	"email": "alice@example.com",
	"password": "PasswordAkunYangBenar"
}
```

Respons sukses `204 No Content`; email dapat didaftarkan ulang. Password salah menghasilkan `401 Unauthorized`. Endpoint dibatasi 10 percobaan per IP per menit. Penghapusan permanen tidak dapat dibatalkan.

Token menggunakan HS256 dan memvalidasi `iss`, `aud`, `iat`, serta `exp`. Default masa berlaku 15 menit. Login dibatasi 10 percobaan per IP dalam satu menit. Isi `CORS_ORIGIN` dengan origin frontend yang diizinkan.

## Pemeriksaan

```powershell
go test ./...
go vet ./...
go run .
```

`README-WINDOWS.md` berisi langkah PowerShell dan contoh pengujian endpoint.
