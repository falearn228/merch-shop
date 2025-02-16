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
	"avito-shop/internal/util"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

// Создаем свой матчер для проверки хеша пароля
type eqCreateUserParamsMatcher struct {
	arg      db.CreateUserParams
	password string
}

func (e eqCreateUserParamsMatcher) Matches(x interface{}) bool {
	arg, ok := x.(db.CreateUserParams)
	if !ok {
		return false
	}

	err := util.CheckPassword(e.password, arg.PasswordHash)
	if err != nil {
		return false
	}

	e.arg.PasswordHash = arg.PasswordHash
	return e.arg.Username == arg.Username
}

func (e eqCreateUserParamsMatcher) String() string {
	return fmt.Sprintf("matches arg %v and password %v", e.arg, e.password)
}

func EqCreateUserParams(arg db.CreateUserParams, password string) gomock.Matcher {
	return eqCreateUserParamsMatcher{arg, password}
}

func TestHandleLogin(t *testing.T) {
	password := "secret"
	hashedPassword, err := util.HashPassword(password)
	require.NoError(t, err)

	user := db.GetUserByUsernameRow{
		ID:           1,
		Username:     "existing_user",
		PasswordHash: hashedPassword,
	}

	testCases := []struct {
		name          string
		body          gin.H
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK_ExistingUser",
			body: gin.H{
				"username": user.Username,
				"password": password,
			},
			buildStubs: func(store *mockdb.MockStore) {
				// Мок для получения существующего пользователя
				store.EXPECT().
					GetUserByUsername(gomock.Any(), user.Username).
					Return(user, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchLoginResponse(t, recorder.Body.Bytes())
			},
		},
		{
			name: "OK_NewUser",
			body: gin.H{
				"username": "new_user",
				"password": password,
			},
			buildStubs: func(store *mockdb.MockStore) {
				// Мок для попытки получения несуществующего пользователя
				store.EXPECT().
					GetUserByUsername(gomock.Any(), "new_user").
					Return(db.GetUserByUsernameRow{}, pgx.ErrNoRows)

				arg := db.CreateUserParams{
					Username: "new_user",
				}

				// Мок для создания нового пользователя
				store.EXPECT().
					CreateUser(gomock.Any(), EqCreateUserParams(arg, password)).
					Return(db.User{
						Username: "new_user",
					}, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchLoginResponse(t, recorder.Body.Bytes())
			},
		},
		{
			name: "BadRequest_InvalidJSON",
			body: gin.H{
				"username": 123, // Неверный тип данных
				"password": true,
			},
			buildStubs: func(store *mockdb.MockStore) {
				// Не ожидаем вызовов к store, так как запрос не проходит валидацию JSON
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "Unauthorized_WrongPassword",
			body: gin.H{
				"username": user.Username,
				"password": "wrong_password",
			},
			buildStubs: func(store *mockdb.MockStore) {
				// Мок для получения пользователя
				store.EXPECT().
					GetUserByUsername(gomock.Any(), user.Username).
					Return(user, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "InternalError_GetUserError",
			body: gin.H{
				"username": user.Username,
				"password": password,
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
			name: "InternalError_CreateUserError",
			body: gin.H{
				"username": "new_user",
				"password": password,
			},
			buildStubs: func(store *mockdb.MockStore) {
				// Мок для попытки получения несуществующего пользователя
				store.EXPECT().
					GetUserByUsername(gomock.Any(), "new_user").
					Return(db.GetUserByUsernameRow{}, pgx.ErrNoRows)

				// Мок для ошибки создания пользователя
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(db.User{}, errors.New("database error"))
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "BadRequest_MissingUsername",
			body: gin.H{
				"password": password,
			},
			buildStubs: func(store *mockdb.MockStore) {
				// Не ожидаем вызовов к store, так как запрос не проходит валидацию
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "BadRequest_MissingPassword",
			body: gin.H{
				"username": user.Username,
			},
			buildStubs: func(store *mockdb.MockStore) {
				// Не ожидаем вызовов к store, так как запрос не проходит валидацию
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
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

			url := "/login"
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
			require.NoError(t, err)

			request.Header.Set("Content-Type", "application/json")

			server.handleLogin(getTestGinContext(t, recorder, request))
			tc.checkResponse(t, recorder)
		})
	}
}

// Вспомогательная функция для проверки ответа при входе
func requireBodyMatchLoginResponse(t *testing.T, body []byte) {
	var gotResponse struct {
		Token string `json:"token"`
	}
	err := json.Unmarshal(body, &gotResponse)
	require.NoError(t, err)
	require.NotEmpty(t, gotResponse.Token)
}

// Вспомогательная функция для создания тестового Gin контекста
func getTestGinContext(t *testing.T, recorder *httptest.ResponseRecorder, request *http.Request) *gin.Context {
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request
	return ctx
}
