-- roles — daftar role yang diakui sistem
CREATE TABLE IF NOT EXISTS roles (
    name        VARCHAR(20) PRIMARY KEY,
    description VARCHAR(150) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO roles (name, description) VALUES
    ('admin', 'Akses penuh terhadap seluruh data dan pengaturan'),
    ('staff', 'Boleh melihat data seluruh mahasiswa, tetapi tidak boleh mengubah'),
    ('user', 'Hanya boleh mengelola datanya sendiri')
ON CONFLICT (name) DO NOTHING;

-- permissions — daftar tindakan yang dapat diberikan kepada role
CREATE TABLE IF NOT EXISTS permissions (
    name        VARCHAR(50) PRIMARY KEY,
    description VARCHAR(150) NOT NULL
);

-- role_permissions — tabel penghubung, inti dari model RBAC
CREATE TABLE IF NOT EXISTS role_permissions (
    role_name       VARCHAR(20) NOT NULL
        REFERENCES roles(name)       ON DELETE CASCADE,
    permission_name VARCHAR(50) NOT NULL
        REFERENCES permissions(name) ON DELETE CASCADE,
    PRIMARY KEY (role_name, permission_name)
);

-- Kunci column role pada users agar hanya berisi role yang dikenal.
-- Sebelumnya column ini VARCHAR biasa: apa pun bisa masuk.
UPDATE users SET role = 'user' WHERE role NOT IN (SELECT name FROM roles);

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_fkey;
ALTER TABLE users
    ADD CONSTRAINT users_role_fkey
    FOREIGN KEY (role) REFERENCES roles(name) ON UPDATE CASCADE;

CREATE INDEX IF NOT EXISTS users_role_idx ON users (role);