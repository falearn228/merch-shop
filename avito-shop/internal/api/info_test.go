package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	db "avito-shop/internal/db/sqlc"
	mockdb "avito-shop/internal/mock"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestHandleGetInfo(t *testing.T) {
	testCases := []struct {
		name          string
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			buildStubs: func(store *mockdb.MockStore) {
				username := "test_user"
				userID := int32(1)
				balance := pgtype.Int4{Int32: 1000, Valid: true}

				// Создаем timestamp для тестов
				now := time.Now()
				pgTimestamp := pgtype.Timestamp{
					Time:  now,
					Valid: true,
				}

				transactions := []db.GetTransactionsRow{
					{
						Timestamp:        pgTimestamp,
						Amount:           100,
						SenderUsername:   "user1",
						ReceiverUsername: username,
					},
					{
						Timestamp:        pgTimestamp,
						Amount:           200,
						SenderUsername:   username,
						ReceiverUsername: "user2",
					},
				}

				purchases := []db.GetPurchasesRow{
					{
						Name:         "t-shirt",
						Quantity:     2,
						PurchaseDate: pgTimestamp,
					},
					{
						Name:         "cup",
						Quantity:     3,
						PurchaseDate: pgTimestamp,
					},
				}

				// Мок для основного пользователя
				store.EXPECT().
					GetUserByUsername(gomock.Any(), username).
					Return(db.GetUserByUsernameRow{
						ID:           userID,
						Username:     username,
						PasswordHash: "hashed_password",
					}, nil)

				store.EXPECT().
					GetTransactions(gomock.Any(), pgtype.Int4{Int32: userID, Valid: true}).
					Return(transactions, nil)

				store.EXPECT().
					GetCurrentBalance(gomock.Any(), userID).
					Return(balance, nil)

				store.EXPECT().
					GetPurchases(gomock.Any(), pgtype.Int4{Int32: userID, Valid: true}).
					Return(purchases, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				// Преобразуем ожидаемый и фактический ответы в JSON и обратно для нормализации типов
				expectedJSON := `{
					"coins": 1000,
					"inventory": [
						{"type": "t-shirt", "quantity": 2},
						{"type": "cup", "quantity": 3}
					],
					"coinHistory": {
						"received": [
							{"fromUser": "user1", "amount": 100}
						],
						"sent": [
							{"toUser": "user2", "amount": 200}
						]
					}
				}`

				var expected map[string]interface{}
				err := json.Unmarshal([]byte(expectedJSON), &expected)
				require.NoError(t, err)

				var actual map[string]interface{}
				err = json.Unmarshal(recorder.Body.Bytes(), &actual)
				require.NoError(t, err)

				require.Equal(t, expected, actual)
			},
		},
		{
			name: "GetUserError",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUserByUsername(gomock.Any(), gomock.Any()).
					Return(db.GetUserByUsernameRow{}, errors.New("database error"))
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "GetTransactionsError",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUserByUsername(gomock.Any(), gomock.Any()).
					Return(db.GetUserByUsernameRow{
						ID:           1,
						Username:     "test_user",
						PasswordHash: "hashed_password",
					}, nil)

				store.EXPECT().
					GetTransactions(gomock.Any(), gomock.Any()).
					Return([]db.GetTransactionsRow{}, errors.New("database error"))
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "GetBalanceError",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUserByUsername(gomock.Any(), gomock.Any()).
					Return(db.GetUserByUsernameRow{
						ID:           1,
						Username:     "test_user",
						PasswordHash: "hashed_password",
					}, nil)

				store.EXPECT().
					GetTransactions(gomock.Any(), gomock.Any()).
					Return([]db.GetTransactionsRow{}, nil)

				store.EXPECT().
					GetCurrentBalance(gomock.Any(), gomock.Any()).
					Return(pgtype.Int4{}, errors.New("database error"))
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "GetPurchasesError",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUserByUsername(gomock.Any(), gomock.Any()).
					Return(db.GetUserByUsernameRow{
						ID:           1,
						Username:     "test_user",
						PasswordHash: "hashed_password",
					}, nil)

				store.EXPECT().
					GetTransactions(gomock.Any(), gomock.Any()).
					Return([]db.GetTransactionsRow{}, nil)

				store.EXPECT().
					GetCurrentBalance(gomock.Any(), gomock.Any()).
					Return(pgtype.Int4{Int32: 1000, Valid: true}, nil)

				store.EXPECT().
					GetPurchases(gomock.Any(), gomock.Any()).
					Return([]db.GetPurchasesRow{}, errors.New("database error"))
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildStubs(store)

			server := &Server{store: store}
			recorder := httptest.NewRecorder()

			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Set("username", "test_user")

			server.handleGetInfo(ctx)
			tc.checkResponse(t, recorder)
		})
	}
}
