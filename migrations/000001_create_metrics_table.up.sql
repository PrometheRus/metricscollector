CREATE TABLE gauges (
    name VARCHAR(255) PRIMARY KEY,
    value DOUBLE PRECISION NOT NULL
);

CREATE TABLE counters (
    name VARCHAR(255) PRIMARY KEY,
    delta bigint NOT NULL
);