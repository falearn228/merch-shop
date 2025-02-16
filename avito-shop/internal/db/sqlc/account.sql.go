// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.28.0
// source: account.sql

package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const createItem = `-- name: CreateItem :one
INSERT INTO items (
    name,
    price
) VALUES (
    $1, $2
) RETURNING id, name, price
`

type CreateItemParams struct {
	Name  string `json:"name"`
	Price int32  `json:"price"`
}

func (q *Queries) CreateItem(ctx context.Context, arg CreateItemParams) (Item, error) {
	row := q.db.QueryRow(ctx, createItem, arg.Name, arg.Price)
	var i Item
	err := row.Scan(&i.ID, &i.Name, &i.Price)
	return i, err
}

const createPurchase = `-- name: CreatePurchase :one
INSERT INTO purchases (
    buyer_id,
    item_id,
    quantity,
    total_cost
) VALUES (
    $1, $2, $3, $4
) RETURNING id, buyer_id, item_id, quantity, total_cost, purchase_date
`

type CreatePurchaseParams struct {
	BuyerID   pgtype.Int4 `json:"buyer_id"`
	ItemID    pgtype.Int4 `json:"item_id"`
	Quantity  int32       `json:"quantity"`
	TotalCost int32       `json:"total_cost"`
}

func (q *Queries) CreatePurchase(ctx context.Context, arg CreatePurchaseParams) (Purchase, error) {
	row := q.db.QueryRow(ctx, createPurchase,
		arg.BuyerID,
		arg.ItemID,
		arg.Quantity,
		arg.TotalCost,
	)
	var i Purchase
	err := row.Scan(
		&i.ID,
		&i.BuyerID,
		&i.ItemID,
		&i.Quantity,
		&i.TotalCost,
		&i.PurchaseDate,
	)
	return i, err
}

const createTransfer = `-- name: CreateTransfer :one
INSERT INTO transactions (
    sender_id,
    receiver_id,
    amount
) VALUES (
    $1, $2, $3
) RETURNING id, sender_id, receiver_id, amount, timestamp
`

type CreateTransferParams struct {
	SenderID   pgtype.Int4 `json:"sender_id"`
	ReceiverID pgtype.Int4 `json:"receiver_id"`
	Amount     int32       `json:"amount"`
}

func (q *Queries) CreateTransfer(ctx context.Context, arg CreateTransferParams) (Transaction, error) {
	row := q.db.QueryRow(ctx, createTransfer, arg.SenderID, arg.ReceiverID, arg.Amount)
	var i Transaction
	err := row.Scan(
		&i.ID,
		&i.SenderID,
		&i.ReceiverID,
		&i.Amount,
		&i.Timestamp,
	)
	return i, err
}

const createUser = `-- name: CreateUser :one
INSERT INTO users (
  username,
  password_hash,
  balance
) VALUES (
  $1, $2, 1000
) RETURNING id, username, password_hash, balance, created_at, updated_at
`

type CreateUserParams struct {
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
}

func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) (User, error) {
	row := q.db.QueryRow(ctx, createUser, arg.Username, arg.PasswordHash)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Username,
		&i.PasswordHash,
		&i.Balance,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getCurrentBalance = `-- name: GetCurrentBalance :one
SELECT balance 
FROM users 
WHERE id = $1
`

func (q *Queries) GetCurrentBalance(ctx context.Context, id int32) (pgtype.Int4, error) {
	row := q.db.QueryRow(ctx, getCurrentBalance, id)
	var balance pgtype.Int4
	err := row.Scan(&balance)
	return balance, err
}

const getItemByID = `-- name: GetItemByID :one
SELECT id, name, price FROM items
WHERE id = $1 LIMIT 1
`

func (q *Queries) GetItemByID(ctx context.Context, id int32) (Item, error) {
	row := q.db.QueryRow(ctx, getItemByID, id)
	var i Item
	err := row.Scan(&i.ID, &i.Name, &i.Price)
	return i, err
}

const getItemByName = `-- name: GetItemByName :one
SELECT id, name, price FROM items
WHERE name = $1 LIMIT 1
`

func (q *Queries) GetItemByName(ctx context.Context, name string) (Item, error) {
	row := q.db.QueryRow(ctx, getItemByName, name)
	var i Item
	err := row.Scan(&i.ID, &i.Name, &i.Price)
	return i, err
}

const getPurchases = `-- name: GetPurchases :many
SELECT 
    i.name,
    p.quantity,
    p.purchase_date
FROM purchases p
JOIN items i ON p.item_id = i.id
WHERE p.buyer_id = $1
ORDER BY p.purchase_date DESC
`

type GetPurchasesRow struct {
	Name         string           `json:"name"`
	Quantity     int32            `json:"quantity"`
	PurchaseDate pgtype.Timestamp `json:"purchase_date"`
}

func (q *Queries) GetPurchases(ctx context.Context, buyerID pgtype.Int4) ([]GetPurchasesRow, error) {
	rows, err := q.db.Query(ctx, getPurchases, buyerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []GetPurchasesRow{}
	for rows.Next() {
		var i GetPurchasesRow
		if err := rows.Scan(&i.Name, &i.Quantity, &i.PurchaseDate); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getTransactions = `-- name: GetTransactions :many
SELECT 
    t.timestamp,
    t.amount,
    sender.username as sender_username,
    receiver.username as receiver_username
FROM transactions t
JOIN users sender ON t.sender_id = sender.id
JOIN users receiver ON t.receiver_id = receiver.id
WHERE t.sender_id = $1 OR t.receiver_id = $1
ORDER BY t.timestamp DESC
`

type GetTransactionsRow struct {
	Timestamp        pgtype.Timestamp `json:"timestamp"`
	Amount           int32            `json:"amount"`
	SenderUsername   string           `json:"sender_username"`
	ReceiverUsername string           `json:"receiver_username"`
}

func (q *Queries) GetTransactions(ctx context.Context, senderID pgtype.Int4) ([]GetTransactionsRow, error) {
	rows, err := q.db.Query(ctx, getTransactions, senderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []GetTransactionsRow{}
	for rows.Next() {
		var i GetTransactionsRow
		if err := rows.Scan(
			&i.Timestamp,
			&i.Amount,
			&i.SenderUsername,
			&i.ReceiverUsername,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getUserByID = `-- name: GetUserByID :one
SELECT id, username, password_hash, balance, created_at, updated_at
FROM users
WHERE id = $1 LIMIT 1
`

func (q *Queries) GetUserByID(ctx context.Context, id int32) (User, error) {
	row := q.db.QueryRow(ctx, getUserByID, id)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Username,
		&i.PasswordHash,
		&i.Balance,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getUserByUsername = `-- name: GetUserByUsername :one
SELECT id, username, password_hash 
FROM users 
WHERE username = $1 
LIMIT 1
`

type GetUserByUsernameRow struct {
	ID           int32  `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
}

func (q *Queries) GetUserByUsername(ctx context.Context, username string) (GetUserByUsernameRow, error) {
	row := q.db.QueryRow(ctx, getUserByUsername, username)
	var i GetUserByUsernameRow
	err := row.Scan(&i.ID, &i.Username, &i.PasswordHash)
	return i, err
}

const updateBalanceForPurchase = `-- name: UpdateBalanceForPurchase :exec
UPDATE users 
SET 
    balance = balance - $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
`

type UpdateBalanceForPurchaseParams struct {
	ID      int32       `json:"id"`
	Balance pgtype.Int4 `json:"balance"`
}

func (q *Queries) UpdateBalanceForPurchase(ctx context.Context, arg UpdateBalanceForPurchaseParams) error {
	_, err := q.db.Exec(ctx, updateBalanceForPurchase, arg.ID, arg.Balance)
	return err
}

const updateBalanceForTransfer = `-- name: UpdateBalanceForTransfer :exec
UPDATE users 
SET 
    balance = CASE 
        WHEN id = $1 THEN balance - $3 -- отправитель
        WHEN id = $2 THEN balance + $3 -- получатель
    END,
    updated_at = CURRENT_TIMESTAMP
WHERE id IN ($1, $2)
`

type UpdateBalanceForTransferParams struct {
	ID      int32       `json:"id"`
	ID_2    int32       `json:"id_2"`
	Balance pgtype.Int4 `json:"balance"`
}

func (q *Queries) UpdateBalanceForTransfer(ctx context.Context, arg UpdateBalanceForTransferParams) error {
	_, err := q.db.Exec(ctx, updateBalanceForTransfer, arg.ID, arg.ID_2, arg.Balance)
	return err
}
