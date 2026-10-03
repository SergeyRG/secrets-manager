CREATE TYPE file_state AS ENUM ('UPLOADING', 'READY');

CREATE TABLE files_state (
    id BIGSERIAL PRIMARY KEY,
    file_id TEXT UNIQUE NOT NULL,
    state file_state NOT NULL
);