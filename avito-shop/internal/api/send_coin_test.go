package api

import (
	"bytes"
	"encoding/json"
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

func TestHandleSendCoin(t *testing.T) {
	amount := int32(100)
	sender := db.GetUserByUsernameRow{
		ID:           1,
		Username:     "sender",
		PasswordHash: "password",
	}
	receiver := db.GetUserByUsernameRow{
		ID:           2,
		Username:     "receiver",
		PasswordHash: "password",
	}

	testCases := []struct {
		name          string
		body          gin.H
		setupAuth     func(t *testing.T, request *http.Request, username string)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: gin.H{
				"toUser": receiver.Username,
				"amount": amount,
			},
			setupAuth: func(t *testing.T, request *http.Request, username string) {
				request.Header.Set("Authorization", "Bearer token")
			},
			buildStubs: func(store *mockdb.MockStore) {
				// Мок для получения отправителя
				store.EXPECT().
					GetUserByUsername(gomock.Any(), sender.Username).
					Return(sender, nil)

				// Мок для получения получателя
				store.EXPECT().
					GetUserByUsername(gomock.Any(), receiver.Username).
					Return(receiver, nil)

				arg := db.TransferTxParams{
					FromUserID: sender.ID,
					ToUserID:   receiver.ID,
					Amount:     amount,
				}

				// Мок для транзакции перевода
				store.EXPECT().
					TransferTx(gomock.Any(), arg).
					Return(db.TransferTxResult{}, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchMessage(t, recorder.Body.Bytes(), "transfer successful")
			},
		},
		{
			name: "BadRequest_InvalidJSON",
			body: gin.H{
				"toUser": 123, // Неверный тип данных
				"amount": "invalid",
			},
			setupAuth: func(t *testing.T, request *http.Request, username string) {
				request.Header.Set("Authorization", "Bearer token")
			},
			buildStubs: func(store *mockdb.MockStore) {
				// Не ожидаем вызовов к store, так как запрос не проходит валидацию JSON
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "BadRequest_NegativeAmount",
			body: gin.H{
				"toUser": receiver.Username,
				"amount": -1,
			},
			setupAuth: func(t *testing.T, request *http.Request, username string) {
				request.Header.Set("Authorization", "Bearer token")
			},
			buildStubs: func(store *mockdb.MockStore) {
				// Не ожидаем вызовов к store, так как запрос не проходит валидацию
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "BadRequest_SendToSelf",
			body: gin.H{
				"toUser": sender.Username,
				"amount": amount,
			},
			setupAuth: func(t *testing.T, request *http.Request, username string) {
				request.Header.Set("Authorization", "Bearer token")
			},
			buildStubs: func(store *mockdb.MockStore) {
				// Мок для получения отправителя
				store.EXPECT().
					GetUserByUsername(gomock.Any(), sender.Username).
					Return(sender, nil)

				// Мок для получения получателя (тот же пользователь)
				store.EXPECT().
					GetUserByUsername(gomock.Any(), sender.Username).
					Return(sender, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				requireBodyMatchError(t, recorder.Body.Bytes(), "cannot send coins to yourself")
			},
		},
		{
			name: "NotFound_ReceiverNotFound",
			body: gin.H{
				"toUser": "nonexistent",
				"amount": amount,
			},
			setupAuth: func(t *testing.T, request *http.Request, username string) {
				request.Header.Set("Authorization", "Bearer token")
			},
			buildStubs: func(store *mockdb.MockStore) {
				// Мок для получения отправителя
				store.EXPECT().
					GetUserByUsername(gomock.Any(), sender.Username).
					Return(sender, nil)

				// Мок для получения несуществующего получателя
				store.EXPECT().
					GetUserByUsername(gomock.Any(), "nonexistent").
					Return(db.GetUserByUsernameRow{}, errors.New("user not found"))
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "BadRequest_InsufficientBalance",
			body: gin.H{
				"toUser": receiver.Username,
				"amount": amount,
			},
			setupAuth: func(t *testing.T, request *http.Request, username string) {
				request.Header.Set("Authorization", "Bearer token")
			},
			buildStubs: func(store *mockdb.MockStore) {
				// Мок для получения отправителя
				store.EXPECT().
					GetUserByUsername(gomock.Any(), sender.Username).
					Return(sender, nil)

				// Мок для получения получателя
				store.EXPECT().
					GetUserByUsername(gomock.Any(), receiver.Username).
					Return(receiver, nil)

				arg := db.TransferTxParams{
					FromUserID: sender.ID,
					ToUserID:   receiver.ID,
					Amount:     amount,
				}

				// Мок для ошибки недостаточного баланса
				store.EXPECT().
					TransferTx(gomock.Any(), arg).
					Return(db.TransferTxResult{}, fmt.Errorf("CHECK constraint"))
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				requireBodyMatchError(t, recorder.Body.Bytes(), "insufficient balance")
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

			// Marshal body data to JSON
			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			url := "/send"
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
			require.NoError(t, err)

			// Установка заголовков и аутентификации
			request.Header.Set("Content-Type", "application/json")
			tc.setupAuth(t, request, sender.Username)

			// Создаем новый Gin контекст
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = request
			ctx.Set("username", sender.Username)

			server.handleSendCoin(ctx)
			tc.checkResponse(t, recorder)
		})
	}
}

// Вспомогательная функция для проверки сообщения в ответе
func requireBodyMatchMessage(t *testing.T, body []byte, message string) {
	var gotResponse struct {
		Message string `json:"message"`
	}
	err := json.Unmarshal(body, &gotResponse)
	require.NoError(t, err)
	require.Equal(t, message, gotResponse.Message)
}

// Вспомогательная функция для проверки сообщения об ошибке
func requireBodyMatchError(t *testing.T, body []byte, message string) {
	var gotResponse struct {
		Error string `json:"error"`
	}
	err := json.Unmarshal(body, &gotResponse)
	require.NoError(t, err)
	require.Equal(t, message, gotResponse.Error)
}
