-- Tabel users: akun untuk login, terpisah dari tabel students.
-- students adalah data yang DILINDUNGI; users adalah akun yang MELINDUNGI.
CREATE TABLE IF NOT EXISTS users (
    id         SERIAL PRIMARY KEY,
    username   VARCHAR(50)  NOT NULL,
    email      VARCHAR(255) NOT NULL,
    password   TEXT         NOT NULL,
    role       VARCHAR(20)  NOT NULL DEFAULT 'user',
    is_active  BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS users_username_lower_key
    ON users (LOWER(username));

-- Refresh token disimpan sebagai HASH, bukan nilai aslinya.
-- Alasannya sama seperti password: bila tabel ini bocor, penyerang
-- tetap tidak memiliki token yang dapat langsung dipakai.
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         BIGSERIAL PRIMARY KEY,
    user_id    INTEGER      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT         NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ  NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS refresh_tokens_user_id_idx
    ON refresh_tokens (user_id);