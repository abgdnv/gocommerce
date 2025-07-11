CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS orders
(
    id         UUID PRIMARY KEY     DEFAULT uuid_generate_v4(),
    user_id    UUID        NOT NULL,
    status     VARCHAR(50) NOT NULL,
    version    INTEGER     NOT NULL DEFAULT 1,
    created_at TIMESTAMP   NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS order_items
(
    id             UUID PRIMARY KEY   DEFAULT uuid_generate_v4(),
    order_id       UUID      NOT NULL,
    product_id     UUID      NOT NULL,
    quantity       INTEGER   NOT NULL DEFAULT 1,
    price_per_item BIGINT    NOT NULL,
    price          BIGINT    NOT NULL,
    version        INTEGER   NOT NULL DEFAULT 1,
    created_at     TIMESTAMP NOT NULL DEFAULT NOW(),
    FOREIGN KEY (order_id) REFERENCES orders (id) ON DELETE CASCADE
);