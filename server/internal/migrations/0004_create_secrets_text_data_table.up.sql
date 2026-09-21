CREATE TABLE secrets_text_data (
    id BIGSERIAL PRIMARY KEY,
    secret_version_id TEXT UNIQUE NOT NULL,
    secret_data TEXT NOT NULL
);