-- +goose Up

CREATE
EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users
(
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username      VARCHAR(50) UNIQUE NOT NULL,
    password_hash TEXT               NOT NULL,
    created_at    TIMESTAMPTZ      DEFAULT NOW(),
    updated_at    TIMESTAMPTZ      DEFAULT NOW()
);

CREATE TABLE user_data
(
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID REFERENCES users (id) ON DELETE CASCADE,
    data_type       VARCHAR(50) NOT NULL,
    data_name       VARCHAR(50) NOT NULL,
    largeobject_oid OID         NOT NULL,
    created_at      TIMESTAMPTZ      DEFAULT NOW()
);

CREATE TABLE user_sessions
(
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID REFERENCES users (id) ON DELETE CASCADE,
    session_token TEXT UNIQUE NOT NULL,
    expires_at    TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ      DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS user_sessions;
DROP TABLE IF EXISTS user_data;
DROP TABLE IF EXISTS users;
DROP EXTENSION IF EXISTS pgcrypto;
