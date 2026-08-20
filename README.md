# API Students

REST API untuk manajemen data mahasiswa, dibangun dengan Go dan Fiber v2.
Tugas Mandiri Pertemuan 2 — Praktikum Pemrograman Backend Lanjut.

## Cara Menjalankan

    go run .

Server berjalan di `http://localhost:3000`.

## Batas Atas Limit Paginasi

Batas atas `limit` ditetapkan sebesar **50**. Data akademik seperti daftar
mahasiswa biasanya ditampilkan dalam jumlah lebih kecil per halaman
dibandingkan data umum, supaya tabel nilai tetap mudah dibaca dan tidak
membebani server saat memproses banyak baris sekaligus.

## Kontrak API

| Metode | Endpoint | Parameter | Contoh Body | Status | Contoh Respons |
|---|---|---|---|---|---|
| GET | /api/v1/students | `page`, `limit`, `search`, `sort`, `order`, `is_active`, `min_grade`, `max_grade` (query, semua opsional) | - | 200 | `{"success":true,"message":"daftar mahasiswa berhasil diambil","data":[...],"meta":{"page":1,"limit":10,"total":3,"total_pages":1}}` |
| GET | /api/v1/students/:id | `id` (path) | - | 200 / 400 / 404 | 200: `{"success":true,"message":"mahasiswa ditemukan","data":{...}}` |
| POST | /api/v1/students | - | `{"nim":"434241001","name":"Sari Wijaya","grade":85}` | 201 / 415 / 422 / 409 | 201: `{"success":true,"message":"mahasiswa berhasil ditambahkan","data":{...}}` (header `Location` disertakan) |
| PUT | /api/v1/students/:id | `id` (path) | `{"nim":"434241002","name":"Budi Santoso Baru","grade":75,"is_active":false}` (semua field wajib) | 200 / 400 / 404 / 409 / 422 | 200: `{"success":true,"message":"mahasiswa berhasil diganti seluruhnya","data":{...}}` |
| PATCH | /api/v1/students/:id | `id` (path) | `{"is_active":true}` (hanya field yang ingin diubah) | 200 / 400 / 404 / 409 / 422 | 200: `{"success":true,"message":"mahasiswa berhasil diperbarui sebagian","data":{...}}` |
| DELETE | /api/v1/students/:id | `id` (path) | - | 204 / 400 / 404 | 204: tanpa body |

## Status HTTP yang Diterapkan

| Status | Situasi |
|---|---|
| 200 | Pengambilan atau perubahan data berhasil |
| 201 | Mahasiswa baru berhasil ditambahkan, disertai header Location |
| 204 | Penghapusan berhasil, tanpa body |
| 400 | Body bukan JSON yang sah, atau id bukan angka |
| 404 | Data mahasiswa tidak ditemukan |
| 409 | NIM bertentangan dengan data yang sudah ada |
| 415 | Content-Type bukan application/json |
| 422 | Validasi isi gagal (misalnya grade di luar 0–100) |