-- name: CreateUser :one
INSERT INTO users (
  username,
  password_hash,
  balance
) VALUES (
  $1, $2, 1000
) RETURNING *;

-- name: CreateItem :one
INSERT INTO items (
    name,
    price
) VALUES (
    $1, $2
) RETURNING id, name, price;

-- name: GetUserByUsername :one
SELECT id, username, password_hash 
FROM users 
WHERE username = $1 
LIMIT 1;

-- name: GetUserByID :one
SELECT id, username, password_hash, balance, created_at, updated_at
FROM users
WHERE id = $1 LIMIT 1;

-- name: GetPurchases :many
SELECT 
    i.name,
    p.quantity,
    p.purchase_date
FROM purchases p
JOIN items i ON p.item_id = i.id
WHERE p.buyer_id = $1
ORDER BY p.purchase_date DESC;

-- name: GetCurrentBalance :one 
SELECT balance 
FROM users 
WHERE id = $1;

-- name: CreateTransfer :one
INSERT INTO transactions (
    sender_id,
    receiver_id,
    amount
) VALUES (
    $1, $2, $3
) RETURNING id, sender_id, receiver_id, amount, timestamp;;

-- name: GetTransactions :many
SELECT 
    t.timestamp,
    t.amount,
    sender.username as sender_username,
    receiver.username as receiver_username
FROM transactions t
JOIN users sender ON t.sender_id = sender.id
JOIN users receiver ON t.receiver_id = receiver.id
WHERE t.sender_id = $1 OR t.receiver_id = $1
ORDER BY t.timestamp DESC;

-- name: UpdateBalanceForTransfer :exec
UPDATE users 
SET 
    balance = CASE 
        WHEN id = $1 THEN balance - $3 -- отправитель
        WHEN id = $2 THEN balance + $3 -- получатель
    END,
    updated_at = CURRENT_TIMESTAMP
WHERE id IN ($1, $2);

-- name: UpdateBalanceForPurchase :exec
UPDATE users 
SET 
    balance = balance - $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: GetItemByID :one
SELECT * FROM items
WHERE id = $1 LIMIT 1;

-- name: GetItemByName :one
SELECT * FROM items
WHERE name = $1 LIMIT 1;

-- name: CreatePurchase :one
INSERT INTO purchases (
    buyer_id,
    item_id,
    quantity,
    total_cost
) VALUES (
    $1, $2, $3, $4
) RETURNING *;