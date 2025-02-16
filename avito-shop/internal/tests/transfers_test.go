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

func TestE2ETransferCoins(t *testing.T) {
	config := api.TokenConfig{
		TokenSymmetricKey:   "12345678901234567890123456789012",
		AccessTokenDuration: time.Hour * 24,
	}

	server, err := api.NewServer(testStore, config)
	require.NoError(t, err)

	// Создаем двух тестовых пользователей
	sender := util.RandomString(6)
	receiver := util.RandomString(6)
	password := "password123"

	// Шаг 1: Регистрируем отправителя и получателя
	users := []string{sender, receiver}
	tokens := make(map[string]string)

	for _, username := range users {
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

		t.Logf("Login response for %s - status: %d, body: %s", username, recorder.Code, recorder.Body.String())
		require.Equal(t, http.StatusOK, recorder.Code)

		var loginResponse struct {
			Token string `json:"token"`
		}
		err = json.Unmarshal(recorder.Body.Bytes(), &loginResponse)
		require.NoError(t, err)
		require.NotEmpty(t, loginResponse.Token)

		tokens[username] = loginResponse.Token
	}

	// Шаг 2: Проверяем начальные балансы обоих пользователей
	initialBalances := make(map[string]int32)

	for _, username := range users {
		req := httptest.NewRequest(http.MethodGet, "/api/info", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokens[username]))
		recorder := httptest.NewRecorder()
		server.Router.ServeHTTP(recorder, req)

		t.Logf("Initial balance response for %s - status: %d, body: %s",
			username, recorder.Code, recorder.Body.String())
		require.Equal(t, http.StatusOK, recorder.Code)

		var infoResponse struct {
			Coins int32 `json:"coins"`
		}
		err = json.Unmarshal(recorder.Body.Bytes(), &infoResponse)
		require.NoError(t, err)
		initialBalances[username] = infoResponse.Coins
		require.Equal(t, int32(1000), infoResponse.Coins) // Проверяем начальный баланс
	}

	// Шаг 3: Отправляем монеты
	transferAmount := int32(200)
	transferBody := gin.H{
		"toUser": receiver,
		"amount": transferAmount,
	}
	transferJSON, err := json.Marshal(transferBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/sendCoin", bytes.NewBuffer(transferJSON))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokens[sender]))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	server.Router.ServeHTTP(recorder, req)

	t.Logf("Transfer response - status: %d, body: %s", recorder.Code, recorder.Body.String())
	require.Equal(t, http.StatusOK, recorder.Code)

	var transferResponse struct {
		Message string `json:"message"`
	}
	err = json.Unmarshal(recorder.Body.Bytes(), &transferResponse)
	require.NoError(t, err)
	require.Equal(t, "transfer successful", transferResponse.Message)

	// Шаг 4: Проверяем обновленные балансы
	time.Sleep(time.Millisecond * 100) // Даем время на обновление БД

	// Проверяем баланс отправителя
	req = httptest.NewRequest(http.MethodGet, "/api/info", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokens[sender]))
	recorder = httptest.NewRecorder()
	server.Router.ServeHTTP(recorder, req)

	t.Logf("Sender updated balance response - status: %d, body: %s",
		recorder.Code, recorder.Body.String())
	require.Equal(t, http.StatusOK, recorder.Code)

	var senderInfo struct {
		Coins       int32                  `json:"coins"`
		CoinHistory map[string]interface{} `json:"coinHistory"`
	}
	err = json.Unmarshal(recorder.Body.Bytes(), &senderInfo)
	require.NoError(t, err)
	require.Equal(t, initialBalances[sender]-transferAmount, senderInfo.Coins)

	// Проверяем баланс получателя
	req = httptest.NewRequest(http.MethodGet, "/api/info", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokens[receiver]))
	recorder = httptest.NewRecorder()
	server.Router.ServeHTTP(recorder, req)

	t.Logf("Receiver updated balance response - status: %d, body: %s",
		recorder.Code, recorder.Body.String())
	require.Equal(t, http.StatusOK, recorder.Code)

	var receiverInfo struct {
		Coins       int32                  `json:"coins"`
		CoinHistory map[string]interface{} `json:"coinHistory"`
	}
	err = json.Unmarshal(recorder.Body.Bytes(), &receiverInfo)
	require.NoError(t, err)
	require.Equal(t, initialBalances[receiver]+transferAmount, receiverInfo.Coins)

	// Шаг 5: Проверяем историю транзакций
	require.NotNil(t, senderInfo.CoinHistory)
	require.NotNil(t, receiverInfo.CoinHistory)

	// Шаг 6: Проверяем ошибочные ситуации

	// Попытка отправить больше монет, чем есть
	largeTransferBody := gin.H{
		"toUser": receiver,
		"amount": 2000, // Больше начального баланса
	}
	largeTransferJSON, err := json.Marshal(largeTransferBody)
	require.NoError(t, err)

	req = httptest.NewRequest(http.MethodPost, "/api/sendCoin", bytes.NewBuffer(largeTransferJSON))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokens[sender]))
	req.Header.Set("Content-Type", "application/json")
	recorder = httptest.NewRecorder()
	server.Router.ServeHTTP(recorder, req)

	t.Logf("Large transfer response - status: %d, body: %s", recorder.Code, recorder.Body.String())
	require.Equal(t, http.StatusInternalServerError, recorder.Code)

	// Попытка отправить монеты несуществующему пользователю
	invalidTransferBody := gin.H{
		"toUser": "nonexistent_user",
		"amount": 100,
	}
	invalidTransferJSON, err := json.Marshal(invalidTransferBody)
	require.NoError(t, err)

	req = httptest.NewRequest(http.MethodPost, "/api/sendCoin", bytes.NewBuffer(invalidTransferJSON))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokens[sender]))
	req.Header.Set("Content-Type", "application/json")
	recorder = httptest.NewRecorder()
	server.Router.ServeHTTP(recorder, req)

	t.Logf("Invalid user transfer response - status: %d, body: %s",
		recorder.Code, recorder.Body.String())
	require.Equal(t, http.StatusNotFound, recorder.Code)

	// Попытка отправить монеты самому себе
	selfTransferBody := gin.H{
		"toUser": sender,
		"amount": 100,
	}
	selfTransferJSON, err := json.Marshal(selfTransferBody)
	require.NoError(t, err)

	req = httptest.NewRequest(http.MethodPost, "/api/sendCoin", bytes.NewBuffer(selfTransferJSON))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokens[sender]))
	req.Header.Set("Content-Type", "application/json")
	recorder = httptest.NewRecorder()
	server.Router.ServeHTTP(recorder, req)

	t.Logf("Self transfer response - status: %d, body: %s", recorder.Code, recorder.Body.String())
	require.Equal(t, http.StatusBadRequest, recorder.Code)
}
