# Penggunaan AI dalam Tugas Ini

## Alat yang digunakan
Claude (Anthropic) — digunakan melalui chat interface.

## Bagian yang dibantu AI
- Penjelasan konsep REST API: anatomi HTTP, metode safe/idempotent, perbedaan
  PUT dan PATCH, pemilihan status HTTP yang tepat, dan perancangan query string.
- Struktur kode dasar (model.go, helper.go, handler.go, main.go) diadaptasi
  dari pola yang sama dengan project latihan-fiber, disesuaikan untuk entitas
  Student.
- Debugging error saat struct di model.go dan main.go sempat terduplikasi
  akibat kesalahan copy-paste.
- Panduan pengujian tiap endpoint dan status HTTP menggunakan curl.

## Bagian yang saya kerjakan sendiri
- Mengetik ulang seluruh kode berdasarkan penjelasan yang diberikan.
- Menjalankan dan menguji setiap endpoint (GET, POST, PUT, PATCH, DELETE)
  di komputer sendiri, termasuk memverifikasi status HTTP yang dihasilkan.
- Menentukan sendiri nilai batas atas (limit) untuk paginasi dan alasannya.
- Menambahkan filter rentang nilai (min_grade, max_grade) sebagai filter
  tambahan pada endpoint daftar mahasiswa.

## Catatan
AI digunakan sebagai tutor yang menjelaskan konsep sebelum saya menulis kode
sendiri, bukan untuk menulis kode secara otomatis tanpa pemahaman.