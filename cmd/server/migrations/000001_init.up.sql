CREATE TABLE metrics (
    id VARCHAR(255) PRIMARY KEY,
    mtype VARCHAR(255) NOT NULL,
    delta BIGINT,
    value DOUBLE PRECISION,
    hash VARCHAR(255)
);