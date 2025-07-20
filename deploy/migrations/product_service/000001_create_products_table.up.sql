CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS products
(
    id             UUID PRIMARY KEY      DEFAULT uuid_generate_v4(),
    name           VARCHAR(255) NOT NULL,
    price          BIGINT       NOT NULL,
    stock_quantity INTEGER      NOT NULL DEFAULT 0,
    version        INTEGER      NOT NULL DEFAULT 1,
    created_at     TIMESTAMP    NOT NULL DEFAULT NOW()
);
