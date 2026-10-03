CREATE TYPE secret_type AS ENUM (
 'SECRET_TYPE_FREE_TEXT',
 'SECRET_TYPE_BINARY',
 'SECRET_TYPE_AUTH_DATA',
 'SECRET_TYPE_BANK_CARD');

CREATE TABLE secrets_metadata (
    id BIGSERIAL PRIMARY KEY,
    user_id TEXT NOT NULL,
    secret_id TEXT NOT NULL,
    secret_name TEXT NOT NULL,
    secret_type secret_type NOT NULL,
    time_creation TIMESTAMPTZ NOT NULL,
    version BIGINT NOT NULL,
    version_id TEXT UNIQUE NOT NULL,

    CONSTRAINT uq_user_secret_name UNIQUE (user_id, secret_name, version)

);
