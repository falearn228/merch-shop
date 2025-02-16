package api

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	db "avito-shop/internal/db/sqlc"
	mockdb "avito-shop/internal/mock"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestHandleBuyItem(t *testing.T) {
	user := db.GetUserByUsernameRow{
		ID:           1,
		Username:     "testuser",
		PasswordHash: "password",
	}

	item := db.Item{
		ID:    1,
		Name:  "t-shirt",
		Price: 100,
	}

	testCases := []struct {
		name          string
		itemName      string
		setupAuth     func(t *testing.T, request *http.Request, username string)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:     "OK",
			itemName: item.Name,
			setupAuth: func(t *testing.T, request *http.Request, username string) {
				request.Header.Set("Authorization", "Bearer token")
			},
			buildStubs: func(store *mockdb.MockStore) {
				// Мок для получения пользователя
				store.EXPECT().
					GetUserByUsername(gomock.Any(), user.Username).
					Return(user, nil)

				// Мок для получения товара
				store.EXPECT().
					GetItemByName(gomock.Any(), item.Name).
					Return(item, nil)

				arg := db.PurchaseTxParams{
					UserID: user.ID,
					ItemID: item.ID,
				}

				// Мок для транзакции покупки
				store.EXPECT().
					PurchaseTx(gomock.Any(), arg).
					Return(db.PurchaseTxResult{}, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchMessage(t, recorder.Body.Bytes(), "purchase successful")
			},
		},
		{
			name:     "NotFound_ItemNotFound",
			itemName: "nonexistent",
			setupAuth: func(t *testing.T, request *http.Request, username string) {
				request.Header.Set("Authorization", "Bearer token")
			},
			buildStubs: func(store *mockdb.MockStore) {
				// Мок для получения пользователя
				store.EXPECT().
					GetUserByUsername(gomock.Any(), user.Username).
					Return(user, nil)

				// Мок для получения несуществующего товара
				store.EXPECT().
					GetItemByName(gomock.Any(), "nonexistent").
					Return(db.Item{}, errors.New("item not found"))
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
				requireBodyMatchError(t, recorder.Body.Bytes(), "item not found")
			},
		},
		{
			name:     "InternalError_GetUserError",
			itemName: item.Name,
			setupAuth: func(t *testing.T, request *http.Request, username string) {
				request.Header.Set("Authorization", "Bearer token")
			},
			buildStubs: func(store *mockdb.MockStore) {
				// Мок для ошибки получения пользователя
				store.EXPECT().
					GetUserByUsername(gomock.Any(), user.Username).
					Return(db.GetUserByUsernameRow{}, errors.New("database error"))
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:     "BadRequest_InsufficientBalance",
			itemName: item.Name,
			setupAuth: func(t *testing.T, request *http.Request, username string) {
				request.Header.Set("Authorization", "Bearer token")
			},
			buildStubs: func(store *mockdb.MockStore) {
				// Мок для получения пользователя
				store.EXPECT().
					GetUserByUsername(gomock.Any(), user.Username).
					Return(user, nil)

				// Мок для получения товара
				store.EXPECT().
					GetItemByName(gomock.Any(), item.Name).
					Return(item, nil)

				arg := db.PurchaseTxParams{
					UserID: user.ID,
					ItemID: item.ID,
				}

				// Мок для ошибки недостаточного баланса
				store.EXPECT().
					PurchaseTx(gomock.Any(), arg).
					Return(db.PurchaseTxResult{}, fmt.Errorf("CHECK constraint"))
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				requireBodyMatchError(t, recorder.Body.Bytes(), "insufficient balance")
			},
		},
		{
			name:     "InternalError_PurchaseError",
			itemName: item.Name,
			setupAuth: func(t *testing.T, request *http.Request, username string) {
				request.Header.Set("Authorization", "Bearer token")
			},
			buildStubs: func(store *mockdb.MockStore) {
				// Мок для получения пользователя
				store.EXPECT().
					GetUserByUsername(gomock.Any(), user.Username).
					Return(user, nil)

				// Мок для получения товара
				store.EXPECT().
					GetItemByName(gomock.Any(), item.Name).
					Return(item, nil)

				arg := db.PurchaseTxParams{
					UserID: user.ID,
					ItemID: item.ID,
				}

				// Мок для внутренней ошибки при покупке
				store.EXPECT().
					PurchaseTx(gomock.Any(), arg).
					Return(db.PurchaseTxResult{}, errors.New("internal error"))
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

			// Создаем тестовый конфиг для токенов
			tokenConfig := TokenConfig{
				TokenSymmetricKey:   "12345678901234567890123456789012",
				AccessTokenDuration: 24,
			}

			server, err := NewServer(store, tokenConfig)
			require.NoError(t, err)

			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/buy/%s", tc.itemName)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			// Установка заголовков и аутентификации
			tc.setupAuth(t, request, user.Username)

			// Создаем новый Gin контекст
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = request
			ctx.Params = []gin.Param{{Key: "item", Value: tc.itemName}}
			ctx.Set("username", user.Username)

			server.handleBuyItem(ctx)
			tc.checkResponse(t, recorder)
		})
	}
}
