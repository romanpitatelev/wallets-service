-- +migrate Up

CREATE TABLE wallets (
    walletId UUID PRIMARY KEY,
    walletName VARCHAR NOT NULL,
    balance NUMERIC NOT NULL,
    currency VARCHAR NOT NULL,
    createdAt TIMESTAMP NOT NULL,
    updatedAt TIMESTAMP NOT NULL,
    deletedAt TIMESTAMP,
    deleted BOOLEAN NOT NULL DEFAULT false
);

-- +migrate Down
DROP TABLE wallets CASCADE;