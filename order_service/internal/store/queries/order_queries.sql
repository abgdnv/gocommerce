-- name: CreateOrder :one
INSERT INTO orders (user_id, status)
VALUES ($1, $2)
RETURNING id, user_id, status, version, created_at;

-- name: FindOrderByID :one
SELECT id, user_id, status, version, created_at
FROM orders
WHERE id = $1;

-- name: FindOrdersByUserID :many
SELECT id, user_id, status, version, created_at
FROM orders
where user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateOrder :one
UPDATE orders
SET status  = $2,
    version = version + 1
WHERE id = $1
  AND version = $3
RETURNING id, user_id, status, version, created_at;

-- name: CreateOrderItem :one
INSERT INTO order_items (order_id, product_id, quantity, price_per_item, price)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, order_id, product_id, quantity, price_per_item, price, version, created_at;

-- name: FindOrderItemsByOrderID :many
SELECT id,
       order_id,
       product_id,
       quantity,
       price_per_item,
       price,
       version,
       created_at
FROM order_items
WHERE order_id = $1;
