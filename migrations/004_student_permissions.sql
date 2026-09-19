-- Permission baru khusus untuk entitas students
INSERT INTO permissions (name, description) VALUES
    ('student:list',       'Melihat daftar seluruh mahasiswa'),
    ('student:read:any',   'Melihat data mahasiswa mana pun'),
    ('student:create',     'Menambahkan data mahasiswa baru'),
    ('student:update:any', 'Mengubah data mahasiswa mana pun'),
    ('student:delete',     'Menghapus data mahasiswa')
ON CONFLICT (name) DO NOTHING;

-- Pemetaan permission ke role, sesuai kebijakan tugas mandiri:
-- admin: akses penuh
-- staff: boleh lihat & tambah, TIDAK boleh ubah/hapus punya orang lain
-- user: tidak dapat permission apapun (haknya cuma lewat kepemilikan)
INSERT INTO role_permissions (role_name, permission_name) VALUES
    ('admin', 'student:list'),
    ('admin', 'student:read:any'),
    ('admin', 'student:create'),
    ('admin', 'student:update:any'),
    ('admin', 'student:delete'),
    ('staff', 'student:list'),
    ('staff', 'student:read:any'),
    ('staff', 'student:create')
ON CONFLICT DO NOTHING;

-- Kolom owner_id: menandai user mana yang mendaftarkan data mahasiswa ini.
-- Inilah dasar pemeriksaan kepemilikan nanti di layer service.
ALTER TABLE students ADD COLUMN IF NOT EXISTS owner_id INTEGER;

-- PENTING: data lama belum punya owner_id (NULL semua).
-- Kita tetapkan owner_id data lama ke admin pertama yang terdaftar,
-- supaya tidak ada baris "yatim piatu" sebelum constraint dipasang.
UPDATE students
SET owner_id = (SELECT id FROM users ORDER BY id ASC LIMIT 1)
WHERE owner_id IS NULL;

ALTER TABLE students
    ALTER COLUMN owner_id SET NOT NULL;

ALTER TABLE students DROP CONSTRAINT IF EXISTS students_owner_id_fkey;
ALTER TABLE students
    ADD CONSTRAINT students_owner_id_fkey
    FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS students_owner_id_idx ON students (owner_id);