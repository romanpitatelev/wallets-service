-- +migrate Up
CREATE TABLE transactions (
    id UUID NOT NULL UNIQUE PRIMARY KEY,
    transaction_type VARCHAR NOT NULL,
    to_wallet_id UUID REFERENCES wallets (wallet_id),
    from_wallet_id UUID REFERENCES wallets (wallet_id),
    amount NUMERIC NOT NULL CHECK (amount > 0),
    currency VARCHAR NOT NULL,
    committed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);



-- +migrate Down
DROP TABLE transactions CASCADE;