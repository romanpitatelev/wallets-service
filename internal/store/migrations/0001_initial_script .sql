-- +migrate Up
CREATE TABLE users (
    user_id UUID PRIMARY KEY,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE wallets (
    wallet_id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users (user_id),
    wallet_name VARCHAR NOT NULL,
    balance NUMERIC NOT NULL DEFAULT 0 CHECK (balance >= 0),
    currency VARCHAR NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    active BOOLEAN NOT NULL DEFAULT TRUE
);

-- Add index on user_id
CREATE INDEX idx_user_id ON wallets(user_id);

-- +migrate Down
DROP INDEX IF EXISTS idx_user_id;
DROP INDEX IF EXISTS idx_active;
DROP TABLE wallets CASCADE;
DROP TABLE users CASCADE;