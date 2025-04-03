-- +migrate Up
CREATE TABLE wallet_history (
    wallet_id UUID NOT NULL,
    user_id UUID NOT NULL,
    wallet_name VARCHAR NOT NULL,
    balance NUMERIC NOT NULL,
    currency VARCHAR NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP,
    active BOOL NOT NULL,
    history_created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    operation_type VARCHAR(10) NOT NULL
);

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION log_wallet_insert()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO wallet_history (
        wallet_id, user_id, wallet_name, balance, currency, 
        created_at, updated_at, deleted_at, active, operation_type 
    ) VALUES (
        NEW.wallet_id, NEW.user_id, NEW.wallet_name, NEW.balance, NEW.currency, 
        NEW.created_at, NEW.updated_at, NEW.deleted_at, NEW.active, 'INSERT'
    );
    RETURN NULL;
END;
$$
    LANGUAGE plpgsql;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION log_wallet_update()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO wallet_history (
        wallet_id, user_id, wallet_name, balance, currency, 
        created_at, updated_at, deleted_at, active, operation_type 
    ) VALUES (
        NEW.wallet_id, NEW.user_id, NEW.wallet_name, NEW.balance, NEW.currency, 
        NEW.created_at, NEW.updated_at, NEW.deleted_at, NEW.active, 'UPDATE'
    );
    RETURN NULL;
END;
$$
LANGUAGE plpgsql;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION log_wallet_delete()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.deleted_at IS NULL THEN
        INSERT INTO wallet_history (
            wallet_id, user_id, wallet_name, balance, currency, 
            created_at, updated_at, deleted_at, active, operation_type 
        ) VALUES (
            OLD.wallet_id, OLD.user_id, OLD.wallet_name, OLD.balance, OLD.currency, 
            OLD.created_at, OLD.updated_at, NOW(), false, 'DELETE'
        );
    END IF;
    RETURN NULL;
END;
$$
LANGUAGE plpgsql;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE OR REPLACE TRIGGER wallet_after_insert
AFTER INSERT ON wallets
FOR EACH ROW EXECUTE FUNCTION log_wallet_insert();

CREATE OR REPLACE TRIGGER wallet_after_update
AFTER UPDATE ON wallets
FOR EACH ROW EXECUTE FUNCTION log_wallet_update();

CREATE OR REPLACE TRIGGER log_wallet_delete
AFTER DELETE ON wallets
FOR EACH ROW EXECUTE FUNCTION log_wallet_delete();
-- +migrate StatementEnd


-- +migrate Down
DROP TABLE IF EXISTS wallet_history CASCADE;