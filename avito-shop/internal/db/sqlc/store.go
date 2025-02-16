package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store interface {
	Querier
	TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error)
	PurchaseTx(ctx context.Context, arg PurchaseTxParams) (PurchaseTxResult, error)
}

type SQLStore struct {
	connPool *pgxpool.Pool
	*Queries
}

func NewStore(connPool *pgxpool.Pool) Store {
	return &SQLStore{
		connPool: connPool,
		Queries:  New(connPool),
	}
}

type TransferTxParams struct {
	FromUserID int32 `json:"from_user_id"`
	ToUserID   int32 `json:"to_user_id"`
	Amount     int32 `json:"amount"`
}

type TransferTxResult struct {
	Transfer Transaction `json:"transfer"`
	FromUser User        `json:"from_user"`
	ToUser   User        `json:"to_user"`
}

type PurchaseTxParams struct {
	UserID   int32 `json:"user_id"`
	ItemID   int32 `json:"item_id"`
	Quantity int32 `json:"quantity"`
}

type PurchaseTxResult struct {
	Purchase Purchase `json:"purchase"`
	User     User     `json:"user"`
	Item     Item     `json:"item"`
}

func (store *SQLStore) execTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := store.connPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := New(tx)
	err = fn(qtx)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (store *SQLStore) TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error) {
	var result TransferTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		// 1. Создаем запись о транзакции
		result.Transfer, err = q.CreateTransfer(ctx, CreateTransferParams{
			SenderID:   pgtype.Int4{Int32: arg.FromUserID, Valid: true},
			ReceiverID: pgtype.Int4{Int32: arg.ToUserID, Valid: true},
			Amount:     arg.Amount,
		})
		if err != nil {
			return fmt.Errorf("error creating transfer: %v", err)
		}

		// 2. Обновляем балансы
		err = q.UpdateBalanceForTransfer(ctx, UpdateBalanceForTransferParams{
			ID:      arg.FromUserID,
			ID_2:    arg.ToUserID,
			Balance: pgtype.Int4{Int32: arg.Amount, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("error updating balances: %v", err)
		}

		// 3. Получаем обновленные данные отправителя
		result.FromUser, err = q.GetUserByID(ctx, arg.FromUserID)
		if err != nil {
			return fmt.Errorf("error getting sender: %v", err)
		}

		// 4. Получаем обновленные данные получателя
		result.ToUser, err = q.GetUserByID(ctx, arg.ToUserID)
		if err != nil {
			return fmt.Errorf("error getting receiver: %v", err)
		}

		return nil
	})

	if err != nil {
		return TransferTxResult{}, fmt.Errorf("transfer tx error: %v", err)
	}

	return result, nil
}

func (store *SQLStore) PurchaseTx(ctx context.Context, arg PurchaseTxParams) (PurchaseTxResult, error) {
	var result PurchaseTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		// 1. Получаем информацию о товаре
		item, err := q.GetItemByID(ctx, arg.ItemID)
		if err != nil {
			return fmt.Errorf("error getting item: %v", err)
		}

		// 2. Проверяем текущий баланс пользователя
		balance, err := q.GetCurrentBalance(ctx, arg.UserID)
		if err != nil {
			return fmt.Errorf("error getting balance: %v", err)
		}

		// Проверяем, достаточно ли денег
		if balance.Int32 < item.Price {
			return fmt.Errorf("insufficient balance")
		}

		// 3. Создаем запись о покупке
		createdPurchase, err := q.CreatePurchase(ctx, CreatePurchaseParams{
			BuyerID:   pgtype.Int4{Int32: arg.UserID, Valid: true},
			ItemID:    pgtype.Int4{Int32: arg.ItemID, Valid: true},
			Quantity:  1,
			TotalCost: item.Price,
		})
		if err != nil {
			return fmt.Errorf("error creating purchase: %v", err)
		}
		result.Purchase = createdPurchase

		// 4. Списываем деньги
		err = q.UpdateBalanceForPurchase(ctx, UpdateBalanceForPurchaseParams{
			ID:      arg.UserID,
			Balance: pgtype.Int4{Int32: item.Price, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("error updating balance: %v", err)
		}

		// 5. Получаем обновленные данные пользователя
		updatedUser, err := q.GetUserByID(ctx, arg.UserID)
		if err != nil {
			return fmt.Errorf("error getting updated user: %v", err)
		}
		result.User = updatedUser

		// 6. Сохраняем информацию о товаре в результате
		result.Item = item

		return nil
	})

	if err != nil {
		return PurchaseTxResult{}, fmt.Errorf("purchase tx error: %v", err)
	}

	return result, nil
}
