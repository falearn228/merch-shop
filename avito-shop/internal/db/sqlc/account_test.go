package db

import (
	util "avito-shop/internal/util"
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func createRandomUser(t *testing.T) User {
	arg := CreateUserParams{
		Username:     util.RandomString(6),
		PasswordHash: util.RandomString(12),
	}

	user, err := testQueries.CreateUser(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, user)

	require.Equal(t, arg.Username, user.Username)
	require.Equal(t, arg.PasswordHash, user.PasswordHash)
	require.Equal(t, int32(1000), user.Balance.Int32) // Начальный баланс
	require.NotZero(t, user.CreatedAt)
	require.NotZero(t, user.UpdatedAt)

	return user
}

func createRandomItem(t *testing.T) Item {
	item := CreateItemParams{
		Name:  util.RandomString(6),
		Price: int32(util.RandomInt(10, 1000)),
	}

	// Предполагаем, что у вас есть метод для создания товара
	// Если нет, нужно добавить его в SQL queries
	createdItem, err := testQueries.CreateItem(context.Background(), item)
	require.NoError(t, err)
	require.NotEmpty(t, createdItem)

	require.Equal(t, item.Name, createdItem.Name)
	require.Equal(t, item.Price, createdItem.Price)

	return createdItem
}

func TestCreateUser(t *testing.T) {
	createRandomUser(t)
}

func TestGetUser(t *testing.T) {
	user1 := createRandomUser(t)
	user2, err := testQueries.GetUserByID(context.Background(), user1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, user2)

	require.Equal(t, user1.ID, user2.ID)
	require.Equal(t, user1.Username, user2.Username)
	require.Equal(t, user1.PasswordHash, user2.PasswordHash)
	require.Equal(t, user1.Balance, user2.Balance)
	require.Equal(t, user1.CreatedAt, user2.CreatedAt)
	require.Equal(t, user1.UpdatedAt, user2.UpdatedAt)
}

func TestGetUserByUsername(t *testing.T) {
	user1 := createRandomUser(t)
	user2, err := testQueries.GetUserByUsername(context.Background(), user1.Username)
	require.NoError(t, err)
	require.NotEmpty(t, user2)

	require.Equal(t, user1.ID, user2.ID)
	require.Equal(t, user1.Username, user2.Username)
	require.Equal(t, user1.PasswordHash, user2.PasswordHash)
}

func TestCreateTransfer(t *testing.T) {
	user1 := createRandomUser(t)
	user2 := createRandomUser(t)

	arg := CreateTransferParams{
		SenderID:   pgtype.Int4{Int32: user1.ID, Valid: true},
		ReceiverID: pgtype.Int4{Int32: user2.ID, Valid: true},
		Amount:     100,
	}

	transfer, err := testQueries.CreateTransfer(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, transfer)

	require.Equal(t, arg.SenderID, transfer.SenderID)
	require.Equal(t, arg.ReceiverID, transfer.ReceiverID)
	require.Equal(t, arg.Amount, transfer.Amount)
	require.NotZero(t, transfer.Timestamp)
}

func TestGetTransactions(t *testing.T) {
	user1 := createRandomUser(t)
	user2 := createRandomUser(t)

	// Создаем несколько транзакций
	for i := 0; i < 5; i++ {
		arg := CreateTransferParams{
			SenderID:   pgtype.Int4{Int32: user1.ID, Valid: true},
			ReceiverID: pgtype.Int4{Int32: user2.ID, Valid: true},
			Amount:     int32(10 * (i + 1)),
		}
		_, err := testQueries.CreateTransfer(context.Background(), arg)
		require.NoError(t, err)
	}

	// Получаем транзакции для первого пользователя
	transactions, err := testQueries.GetTransactions(context.Background(), pgtype.Int4{Int32: user1.ID, Valid: true})
	require.NoError(t, err)
	require.NotEmpty(t, transactions)
	require.Len(t, transactions, 5)

	for _, trx := range transactions {
		require.Equal(t, user1.Username, trx.SenderUsername)
		require.Equal(t, user2.Username, trx.ReceiverUsername)
		require.NotZero(t, trx.Amount)
		require.NotZero(t, trx.Timestamp)
	}
}

func TestCreatePurchase(t *testing.T) {
	user := createRandomUser(t)
	item := createRandomItem(t)

	arg := CreatePurchaseParams{
		BuyerID:   pgtype.Int4{Int32: user.ID, Valid: true},
		ItemID:    pgtype.Int4{Int32: item.ID, Valid: true},
		Quantity:  2,
		TotalCost: item.Price * 2,
	}

	purchase, err := testQueries.CreatePurchase(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, purchase)

	require.Equal(t, arg.BuyerID, purchase.BuyerID)
	require.Equal(t, arg.ItemID, purchase.ItemID)
	require.Equal(t, arg.Quantity, purchase.Quantity)
	require.Equal(t, arg.TotalCost, purchase.TotalCost)
	require.NotZero(t, purchase.PurchaseDate)
}

func TestGetPurchases(t *testing.T) {
	user := createRandomUser(t)
	item1 := createRandomItem(t)
	item2 := createRandomItem(t)

	// Создаем несколько покупок
	n := 5
	for i := 0; i < n; i++ {
		item := item1
		if i%2 == 0 {
			item = item2
		}

		arg := CreatePurchaseParams{
			BuyerID:   pgtype.Int4{Int32: user.ID, Valid: true},
			ItemID:    pgtype.Int4{Int32: item.ID, Valid: true},
			Quantity:  int32(i + 1),
			TotalCost: item.Price * int32(i+1),
		}

		_, err := testQueries.CreatePurchase(context.Background(), arg)
		require.NoError(t, err)
	}

	// Получаем все покупки пользователя
	purchases, err := testQueries.GetPurchases(context.Background(), pgtype.Int4{Int32: user.ID, Valid: true})
	require.NoError(t, err)
	require.NotEmpty(t, purchases)
	require.Len(t, purchases, n)

	// Проверяем, что покупки отсортированы по дате (от новых к старым)
	for i := 1; i < len(purchases); i++ {
		require.True(t, purchases[i-1].PurchaseDate.Time.After(purchases[i].PurchaseDate.Time))
	}
}

func TestUpdateBalanceForTransfer(t *testing.T) {
	user1 := createRandomUser(t)
	user2 := createRandomUser(t)

	// Запоминаем начальные балансы
	initialBalance1 := user1.Balance
	initialBalance2 := user2.Balance

	// Создаем трансфер
	amount := pgtype.Int4{Int32: 100, Valid: true}
	arg := UpdateBalanceForTransferParams{
		ID:      user1.ID,
		ID_2:    user2.ID,
		Balance: amount,
	}

	err := testQueries.UpdateBalanceForTransfer(context.Background(), arg)
	require.NoError(t, err)

	// Проверяем обновленные балансы
	updatedUser1, err := testQueries.GetUserByID(context.Background(), user1.ID)
	require.NoError(t, err)
	updatedUser2, err := testQueries.GetUserByID(context.Background(), user2.ID)
	require.NoError(t, err)

	// Проверяем, что балансы обновились правильно
	require.Equal(t, initialBalance1.Int32-amount.Int32, updatedUser1.Balance.Int32)
	require.Equal(t, initialBalance2.Int32+amount.Int32, updatedUser2.Balance.Int32)
}

func TestUpdateBalanceForPurchase(t *testing.T) {
	user := createRandomUser(t)
	item := createRandomItem(t)

	// Запоминаем начальный баланс
	initialBalance := user.Balance

	// Создаем покупку
	cost := pgtype.Int4{Int32: item.Price, Valid: true}
	arg := UpdateBalanceForPurchaseParams{
		ID:      user.ID,
		Balance: cost,
	}

	err := testQueries.UpdateBalanceForPurchase(context.Background(), arg)
	require.NoError(t, err)

	// Проверяем обновленный баланс
	updatedUser, err := testQueries.GetUserByID(context.Background(), user.ID)
	require.NoError(t, err)

	// Проверяем, что баланс обновился правильно
	require.Equal(t, initialBalance.Int32-cost.Int32, updatedUser.Balance.Int32)
}

func TestGetCurrentBalance(t *testing.T) {
	user := createRandomUser(t)

	balance, err := testQueries.GetCurrentBalance(context.Background(), user.ID)
	require.NoError(t, err)
	require.Equal(t, user.Balance, balance)
}

func TestGetItemByID(t *testing.T) {
	item1 := createRandomItem(t)
	item2, err := testQueries.GetItemByID(context.Background(), item1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, item2)

	require.Equal(t, item1.ID, item2.ID)
	require.Equal(t, item1.Name, item2.Name)
	require.Equal(t, item1.Price, item2.Price)
}

func TestGetItemByName(t *testing.T) {
	item1 := createRandomItem(t)
	item2, err := testQueries.GetItemByName(context.Background(), item1.Name)
	require.NoError(t, err)
	require.NotEmpty(t, item2)

	require.Equal(t, item1.ID, item2.ID)
	require.Equal(t, item1.Name, item2.Name)
	require.Equal(t, item1.Price, item2.Price)
}
