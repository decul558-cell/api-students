-- Index khusus untuk cursor pagination pada students.
-- PENTING: urutan kolom pada index HARUS SAMA PERSIS dengan urutan
-- ORDER BY pada query, termasuk arah DESC-nya. Bila arahnya tidak cocok,
-- PostgreSQL tidak bisa memakai index ini secara optimal (akan melakukan
-- full scan lalu sort manual, meski index-nya "ada").
CREATE INDEX IF NOT EXISTS students_created_at_id_desc_idx
    ON students (created_at DESC, id DESC);