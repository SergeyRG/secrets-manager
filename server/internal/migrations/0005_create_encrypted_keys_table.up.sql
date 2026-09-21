CREATE TABLE encrypted_keys (
    id BIGSERIAL PRIMARY KEY,
    user_id TEXT UNIQUE NOT NULL,
    encrypted_key TEXT NOT NULL
);