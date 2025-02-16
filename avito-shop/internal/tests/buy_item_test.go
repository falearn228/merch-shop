package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"avito-shop/internal/api"
	"avito-shop/internal/util"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestE2EMerchandisePurchase(t *testing.T) {
	require.NotNil(t, testStore)

	config := api.TokenConfig{
		TokenSymmetricKey:   "12345678901234567890123456789012",
		AccessTokenDuration: time.Hour * 24,
	}

	server, err := api.NewServer(testStore, config)
	require.NoError(t, err)
	require.NotNil(t, server)
	require.NotNil(t, server.Router)

	username := util.RandomString(6)
	password := "password123"

	// Шаг 1: Регистрация/Логин
	loginBody := api.LoginRequest{
		Username: username,
		Password: password,
	}
	loginJSON, err := json.Marshal(loginBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/auth", bytes.NewBuffer(loginJSON))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	server.Router.ServeHTTP(recorder, req)

	t.Logf("Login response status: %d", recorder.Code)
	t.Logf("Login response body: %s", recorder.Body.String())
	require.Equal(t, http.StatusOK, recorder.Code)

	var loginResponse struct {
		Token string `json:"token"`
	}
	err = json.Unmarshal(recorder.Body.Bytes(), &loginResponse)
	require.NoError(t, err)
	require.NotEmpty(t, loginResponse.Token)

	token := loginResponse.Token
	authHeader := fmt.Sprintf("Bearer %s", token)

	// Шаг 2: Проверяем начальный баланс
	req = httptest.NewRequest(http.MethodGet, "/api/info", nil)
	req.Header.Set("Authorization", authHeader)
	recorder = httptest.NewRecorder()
	server.Router.ServeHTTP(recorder, req)

	t.Logf("Info response status: %d", recorder.Code)
	t.Logf("Info response body: %s", recorder.Body.String())
	require.Equal(t, http.StatusOK, recorder.Code)

	var infoResponse struct {
		Coins     int32                    `json:"coins"`
		Inventory []map[string]interface{} `json:"inventory"`
	}
	err = json.Unmarshal(recorder.Body.Bytes(), &infoResponse)
	require.NoError(t, err)
	initialBalance := infoResponse.Coins
	require.Equal(t, int32(1000), initialBalance)

	// Шаг 3: Покупаем футболку
	req = httptest.NewRequest(http.MethodGet, "/api/buy/t-shirt", nil)
	req.Header.Set("Authorization", authHeader)
	recorder = httptest.NewRecorder()
	server.Router.ServeHTTP(recorder, req)

	t.Logf("Purchase response status: %d", recorder.Code)
	t.Logf("Purchase response body: %s", recorder.Body.String())
	require.Equal(t, http.StatusOK, recorder.Code)

	var purchaseResponse struct {
		Message string `json:"message"`
	}
	err = json.Unmarshal(recorder.Body.Bytes(), &purchaseResponse)
	require.NoError(t, err)
	require.Equal(t, "purchase successful", purchaseResponse.Message)

	time.Sleep(time.Millisecond * 100)

	// Шаг 4: Проверяем обновленный баланс
	req = httptest.NewRequest(http.MethodGet, "/api/info", nil)
	req.Header.Set("Authorization", authHeader)
	recorder = httptest.NewRecorder()
	server.Router.ServeHTTP(recorder, req)

	t.Logf("Updated info response status: %d", recorder.Code)
	t.Logf("Updated info response body: %s", recorder.Body.String())
	require.Equal(t, http.StatusOK, recorder.Code)

	var updatedInfoResponse struct {
		Coins     int32                    `json:"coins"`
		Inventory []map[string]interface{} `json:"inventory"`
	}
	err = json.Unmarshal(recorder.Body.Bytes(), &updatedInfoResponse)
	require.NoError(t, err)

	t.Logf("Initial balance: %d, Updated balance: %d", initialBalance, updatedInfoResponse.Coins)
	require.Equal(t, initialBalance-80, updatedInfoResponse.Coins)

	found := false
	for _, item := range updatedInfoResponse.Inventory {
		t.Logf("Inventory item: %+v", item)
		if item["type"] == "t-shirt" {
			require.Equal(t, float64(1), item["quantity"])
			found = true
			break
		}
	}
	require.True(t, found, "purchased item not found in inventory")

	// Шаг 5: Проверяем покупку слишком дорогого товара
	req = httptest.NewRequest(http.MethodGet, "/api/buy/pink-hoody", nil)
	req.Header.Set("Authorization", authHeader)
	recorder = httptest.NewRecorder()
	server.Router.ServeHTTP(recorder, req)
	t.Logf("Initial balance: %d, Updated balance: %d", initialBalance-500, updatedInfoResponse.Coins)

	req = httptest.NewRequest(http.MethodGet, "/api/buy/pink-hoody", nil)
	req.Header.Set("Authorization", authHeader)
	recorder = httptest.NewRecorder()
	server.Router.ServeHTTP(recorder, req)

	t.Logf("Expensive purchase response status: %d", recorder.Code)
	t.Logf("Expensive purchase response body: %s", recorder.Body.String())
	require.Equal(t, http.StatusInternalServerError, recorder.Code)

	var errorResponse struct {
		Error string `json:"error"`
	}
	err = json.Unmarshal(recorder.Body.Bytes(), &errorResponse)
	require.NoError(t, err)
	require.Equal(t, "purchase tx error: insufficient balance", errorResponse.Error)

	// Шаг 6: Проверяем покупку несуществующего товара
	req = httptest.NewRequest(http.MethodGet, "/api/buy/nonexistent-item", nil)
	req.Header.Set("Authorization", authHeader)
	recorder = httptest.NewRecorder()
	server.Router.ServeHTTP(recorder, req)

	t.Logf("Nonexistent item response status: %d", recorder.Code)
	t.Logf("Nonexistent item response body: %s", recorder.Body.String())
	require.Equal(t, http.StatusNotFound, recorder.Code)

	// Шаг 7: Проверяем финальное состояние
	req = httptest.NewRequest(http.MethodGet, "/api/info", nil)
	req.Header.Set("Authorization", authHeader)
	recorder = httptest.NewRecorder()
	server.Router.ServeHTTP(recorder, req)

	t.Logf("Final info response status: %d", recorder.Code)
	t.Logf("Final info response body: %s", recorder.Body.String())
	require.Equal(t, http.StatusOK, recorder.Code)

	var finalInfoResponse struct {
		Coins     int32                    `json:"coins"`
		Inventory []map[string]interface{} `json:"inventory"`
	}
	err = json.Unmarshal(recorder.Body.Bytes(), &finalInfoResponse)
	require.NoError(t, err)

	require.Equal(t, initialBalance-80-500, finalInfoResponse.Coins)
}
