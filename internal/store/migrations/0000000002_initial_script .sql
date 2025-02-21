-- +migrate Up

CREATE TABLE users (
    userid NUMERIC UNIQUE PRIMARY KEY,
    deleted BOOLEAN NOT NULL
);

CREATE INDEX idx_users ON users (userid);

-- +migrate Down
DROP TABLE users CASCADE;