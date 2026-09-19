CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    user_id TEXT UNIQUE NOT NULL,
    login TEXT UNIQUE NOT NULL,
    pwd_hash TEXT NOT NULL
);