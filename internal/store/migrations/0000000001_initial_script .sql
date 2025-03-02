-- +migrate Up

CREATE TABLE users
(
    user_id    UUID UNIQUE PRIMARY KEY,
    deleted_at timestamp with time zone
);

CREATE TABLE wallets
(
    wallet_id   UUID PRIMARY KEY,
    wallet_name VARCHAR   NOT NULL,
    owner_id    UUID REFERENCES users (user_id),
    balance     NUMERIC   NOT NULL DEFAULT 0 CHECK ( balance >= 0 ),
    currency    VARCHAR   NOT NULL,
    created_at  TIMESTAMP NOT NULL default current_timestamp,
    updated_at  TIMESTAMP NOT NULL default current_timestamp,
    deleted_at  timestamp with time zone
);

-- +migrate Down
DROP TABLE wallets CASCADE;
DROP TABLE users CASCADE;
