# Menjalankan Login Service di Windows

Panduan ini menggunakan **Windows PowerShell 5.1 atau PowerShell 7**. Jalankan perintah dari folder proyek yang berisi `go.mod`, `main.go`, dan `.env.example`. Penjelasan arsitektur tersedia di [README.md](README.md).

## 1. Install Go

1. Unduh installer Go untuk Windows (`.msi`) dari [situs resmi Go](https://go.dev/dl/), sesuai arsitektur komputer Anda.
2. Install **Go 1.22 atau lebih baru**, mengikuti versi minimum di `go.mod`.
3. Buka ulang PowerShell dan VS Code setelah instalasi, kemudian periksa:

   ```powershell
   go version
   ```

Jika `go` belum dikenali, periksa PATH instalasi Go, biasanya `C:\Program Files\Go\bin`.

## 2. Buka folder proyek

Buka folder proyek ini di VS Code, lalu buka terminal dengan profil **PowerShell**. Atau ganti path berikut dengan lokasi proyek Anda:

```powershell
Set-Location 'C:\path\ke\pertemuan1-praktik'
Get-ChildItem
```

Pastikan folder tersebut berisi `main.go`, `go.mod`, dan `.env.example`.

## 3. Siapkan file .env

Untuk konfigurasi pertama kali:

```powershell
Copy-Item .env.example .env
notepad .env
```

Jika `.env` sudah ada dan sudah diisi, langsung edit file tersebut tanpa menyalin ulang template.

Buat secret JWT acak menggunakan fasilitas kriptografi bawaan Windows/.NET; OpenSSL tidak diperlukan:

```powershell
$secretBytes = New-Object byte[] 32
$randomGenerator = [System.Security.Cryptography.RandomNumberGenerator]::Create()
try {
    $randomGenerator.GetBytes($secretBytes)
} finally {
    $randomGenerator.Dispose()
}
$jwtSecret = [System.BitConverter]::ToString($secretBytes).Replace('-', '').ToLowerInvariant()
$jwtSecret
```

Salin hasil 64 karakter tersebut ke `JWT_SECRET` pada `.env`. Isi konfigurasi berikut:

| Variabel | Nilai |
| --- | --- |
| `DATABASE_URL` | Connection string PostgreSQL dari provider/database Anda dengan konfigurasi TLS yang sesuai |
| `JWT_SECRET` | Secret acak yang baru dibuat |
| `HTTP_ADDRESS` | `127.0.0.1:8080` untuk pengembangan lokal |
| `CORS_ORIGIN` | Origin frontend, misalnya `http://localhost:3000` |
| `JWT_TTL` | `15m` |

Nilai `JWT_ISSUER` dan `JWT_AUDIENCE` dapat mengikuti template. Simpan `.env` sebagai **UTF-8 tanpa BOM**, dengan nama persis `.env`, bukan `.env.txt`. Di VS Code, gunakan menu encoding pada status bar jika perlu.

DBCode menyimpan kredensial koneksinya secara terpisah; aplikasi membaca `DATABASE_URL` dari `.env` atau environment dan tidak dapat mengambil password dari Secret Storage DBCode. URL-encode karakter khusus password jika merangkai URL secara manual. `.env` sudah diabaikan Git. Environment yang sudah diekspor mengalahkan nilai dari `.env`.

## 4. Jalankan migrasi melalui DBCode

1. Di VS Code Windows, install extension DBCode dan hubungkan ke database PostgreSQL yang dituju.
2. Pastikan koneksi yang dipilih mengarah ke database `website_loan_banking` dan tabel `public.users` tersedia.
3. Jalankan seluruh [migrations/001_users.sql](migrations/001_users.sql), lalu [migrations/002_move_auth_users_to_users.sql](migrations/002_move_auth_users_to_users.sql), termasuk `BEGIN` dan `COMMIT`.

Migrasi pertama menambahkan index unik case-insensitive pada username dan email. Migrasi kedua memindahkan akun lama dari `auth_users` ke `users` dan menghapus `auth_users`. Jika menemukan bentrok ID, username, atau email, migrasi kedua berhenti tanpa menghapus tabel lama. Jika migrasi gagal, jalankan `ROLLBACK;` sebelum mengulang.

Tidak perlu memasang PostgreSQL lokal jika memakai database cloud. Migrasi tidak berjalan otomatis saat server dimulai.

## 5. Unduh dependensi dan siapkan akun

```powershell
go mod download
go mod verify
go run ./cmd/create-user --username alice --email alice@example.com
```

Jalankan perintah pembuatan akun dalam terminal interaktif. Masukkan password lalu konfirmasinya; karakter password tidak ditampilkan. Password akun baru minimal 12 karakter dan maksimal 72 byte. Database menyimpan hash bcrypt.

Jika akun sudah dibuat pada database yang sama, gunakan akun tersebut dan lewati perintah `create-user`.

## 6. Jalankan backend

```powershell
go run .
```

Biarkan terminal ini berjalan. Buka terminal PowerShell kedua untuk mencoba API. Gunakan **Ctrl+C** untuk menghentikan backend.

## 7. Coba health check, login, dan profil

Di terminal kedua:

```powershell
$baseUrl = 'http://127.0.0.1:8080'
Invoke-RestMethod -Method Get -Uri "$baseUrl/healthz"
Invoke-RestMethod -Method Get -Uri "$baseUrl/readyz"
```

Hasil yang diharapkan adalah `ok` dan `ready`. Selanjutnya, masukkan password akun melalui prompt kredensial agar tidak ditulis langsung dalam riwayat perintah:

```powershell
$credentials = Get-Credential -UserName 'alice@example.com' -Message 'Masukkan password akun login service'
$loginBody = @{
    email = $credentials.UserName
    password = $credentials.GetNetworkCredential().Password
} | ConvertTo-Json

$loginParameters = @{
    Method = 'Post'
    Uri = "$baseUrl/api/v1/auth/login"
    ContentType = 'application/json; charset=utf-8'
    Body = [System.Text.Encoding]::UTF8.GetBytes($loginBody)
}

try {
    $loginResponse = Invoke-RestMethod @loginParameters
} finally {
    Remove-Variable credentials, loginBody, loginParameters -ErrorAction SilentlyContinue
}

$loginResponse.data.user | Format-List
```

Jika berhasil, hasil menampilkan `id`, `username`, dan `email`. Gunakan token dari hasil login untuk mengambil profil:

```powershell
$profileParameters = @{
    Method = 'Get'
    Uri = "$baseUrl/api/v1/auth/me"
    Headers = @{ Authorization = "Bearer $($loginResponse.data.access_token)" }
}

$profileResponse = Invoke-RestMethod @profileParameters
$profileResponse.data | Format-List
```

Contoh ini memakai `Invoke-RestMethod` agar penulisan JSON dan header sesuai PowerShell. Token berlaku default 15 menit; login ulang jika kedaluwarsa.

## 8. Jalankan pemeriksaan dan buat executable

Dari folder proyek:

```powershell
go test ./...
go vet ./...
New-Item -ItemType Directory -Path .\bin -Force | Out-Null
go build -o .\bin\server.exe .
.\bin\server.exe
```

Hentikan `go run .` terlebih dahulu agar port tidak digunakan dua proses. Jalankan executable dari folder proyek agar `.env` ditemukan. Pada deployment, konfigurasi juga dapat disediakan melalui environment.

Untuk pengujian race detector pada Windows/amd64:

```powershell
go test -race ./...
```

Race detector memerlukan CGO dan compiler C yang kompatibel, dengan runtime mingw-w64 versi 8 atau lebih baru. Jika toolchain tersebut belum tersedia, `go test ./...` tetap dapat digunakan untuk pengujian biasa; pemeriksaan race dapat dijalankan pada Mac atau CI yang mendukungnya. Lihat [persyaratan resmi race detector Go](https://go.dev/doc/articles/race_detector#Requirements).

## Kendala umum

| Kendala | Langkah pengecekan |
| --- | --- |
| `go` tidak dikenali | Buka ulang terminal/VS Code dan periksa PATH Go |
| `.env` tidak terbaca | Jalankan dari folder proyek; periksa nama file dan encoding tanpa BOM |
| `JWT_SECRET` tidak valid | Isi secret acak 64 karakter dari langkah 3 |
| Tidak terhubung ke database | Periksa connection string, password, akses jaringan, dan `sslmode=verify-full` |
| Tabel `users` tidak ditemukan | Pastikan database `website_loan_banking` yang dipilih dan skema awal database sudah tersedia |
| `401 Unauthorized` | Periksa email/password, `is_active`, hash bcrypt, atau masa berlaku token |
| `429 Too Many Requests` | Tunggu sekitar satu menit sebelum mencoba login lagi |
| Port `8080` sudah digunakan | Hentikan proses sebelumnya atau ubah `HTTP_ADDRESS` dan `$baseUrl` |

Referensi perintah HTTP: [Microsoft — Invoke-RestMethod](https://learn.microsoft.com/en-us/powershell/module/microsoft.powershell.utility/invoke-restmethod?view=powershell-5.1).
