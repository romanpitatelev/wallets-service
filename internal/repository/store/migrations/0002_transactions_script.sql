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

CREATE INDEX idx_to_wallet_id ON transactions(to_wallet_id);
CREATE INDEX idx_from_wallet_id ON transactions(from_wallet_id);

-- +migrate Down
DROP INDEX IF EXISTS idx_to_wallet_id;
DROP INDEX IF EXISTS idx_from_wallet_id;
DROP TABLE transactions CASCADE;