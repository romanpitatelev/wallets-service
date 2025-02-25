-- +migrate Up

CREATE TABLE wallets (
    walletId UUID PRIMARY KEY,
    userId UUID NOT NULL REFERENCES users(userid),
    walletName VARCHAR NOT NULL,
    balance NUMERIC NOT NULL,
    currency VARCHAR NOT NULL,
    createdAt TIMESTAMP NOT NULL,
    updatedAt TIMESTAMP NOT NULL,
    deletedAt TIMESTAMP NOT NULL
);

-- +migrate Down
DROP TABLE wallets CASCADE;