-- +migrate Up

CREATE TABLE ips (
    id UUID PRIMARY KEY,
    ipaddress VARCHAR,
    count NUMERIC,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP,
    deleted BOOL NOT NULL
);

CREATE INDEX idx_ips ON ips (ipaddress)