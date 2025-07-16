-- name: Create :one
INSERT INTO products (name,
                      price,
                      stock_quantity
                      )
VALUES ($1, $2, $3)
RETURNING *;

-- name: FindByID :one
SELECT *
FROM products
WHERE id = $1;

-- name: FindByIDs :many
SELECT * FROM products
WHERE id = ANY(@ids::uuid[]);

-- name: FindAll :many
SELECT *
FROM products
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: Update :one
UPDATE products
SET name           = $2,
    price          = $3,
    stock_quantity = $4,
    version        = version + 1
WHERE id = $1 AND VERSION = $5
RETURNING *;

-- name: Delete :execrows
DELETE
FROM products
WHERE id = $1 AND VERSION = $2;

-- name: UpdateStock :one
UPDATE products
SET stock_quantity = $2,
    version        = version + 1
WHERE id = $1 AND VERSION = $3
RETURNING *;
