-- +migrate Up

CREATE TABLE ips (
    id SERIAL PRIMARY KEY,
    ipaddress VARCHAR UNIQUE,
    count NUMERIC
);

CREATE INDEX idx_ips ON ips (ipaddress);

-- +migrate Down
DROP TABLE ips;