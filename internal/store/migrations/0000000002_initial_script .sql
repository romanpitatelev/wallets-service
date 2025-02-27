-- +migrate Up

CREATE TABLE users (
    userid UUID UNIQUE PRIMARY KEY,
    deleted BOOLEAN NOT NULL
);

-- +migrate Down
DROP TABLE users CASCADE;