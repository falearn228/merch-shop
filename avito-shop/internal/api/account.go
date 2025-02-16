package api

import (
	db "avito-shop/internal/db/sqlc"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

// Структуры запросов и ответов
// type SendCoinRequest struct {
// 	ReceiverUsername string `json:"receiver_username" binding:"required"`
// 	Amount           int32  `json:"amount" binding:"required,gt=0"`
// }

type SendCoinRequest struct {
	ToUser string `json:"toUser" binding:"required"`
	Amount int32  `json:"amount" binding:"required,gt=0"`
}

// type InfoResponse struct {
// 	Balance      int32                   `json:"balance"`
// 	Purchases    []db.GetPurchasesRow    `json:"purchases"`
// 	Transactions []db.GetTransactionsRow `json:"transactions"`
// }

type InfoResponse struct {
	Coins     int32 `json:"coins"`
	Inventory []struct {
		Type     string `json:"type"`
		Quantity int32  `json:"quantity"`
	} `json:"inventory"`
	CoinHistory struct {
		Received []struct {
			FromUser string `json:"fromUser"`
			Amount   int32  `json:"amount"`
		} `json:"received"`
		Sent []struct {
			ToUser string `json:"toUser"`
			Amount int32  `json:"amount"`
		} `json:"sent"`
	} `json:"coinHistory"`
}

// GET /api/info
func (server *Server) handleGetInfo(c *gin.Context) {
	username := c.MustGet("username").(string)
	user, err := server.store.GetUserByUsername(c, username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Получаем транзакции
	userIDPg := pgtype.Int4{Int32: user.ID, Valid: true}
	transactions, err := server.store.GetTransactions(c, userIDPg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	balance, err := server.store.GetCurrentBalance(c, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Формируем историю монет
	coinHistory := struct {
		Received []struct {
			FromUser string `json:"fromUser"`
			Amount   int32  `json:"amount"`
		} `json:"received"`
		Sent []struct {
			ToUser string `json:"toUser"`
			Amount int32  `json:"amount"`
		} `json:"sent"`
	}{}

	for _, t := range transactions {
		if t.ReceiverUsername == username {
			coinHistory.Received = append(coinHistory.Received, struct {
				FromUser string `json:"fromUser"`
				Amount   int32  `json:"amount"`
			}{
				FromUser: t.SenderUsername,
				Amount:   t.Amount,
			})
		} else {
			coinHistory.Sent = append(coinHistory.Sent, struct {
				ToUser string `json:"toUser"`
				Amount int32  `json:"amount"`
			}{
				ToUser: t.ReceiverUsername,
				Amount: t.Amount,
			})
		}
	}

	// Получаем и группируем инвентарь
	purchases, err := server.store.GetPurchases(c, userIDPg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	inventory := make(map[string]int32)
	for _, p := range purchases {
		inventory[p.Name] += p.Quantity
	}

	var inventoryResponse []struct {
		Type     string `json:"type"`
		Quantity int32  `json:"quantity"`
	}
	for itemType, quantity := range inventory {
		inventoryResponse = append(inventoryResponse, struct {
			Type     string `json:"type"`
			Quantity int32  `json:"quantity"`
		}{
			Type:     itemType,
			Quantity: quantity,
		})
	}

	response := gin.H{
		"coins":       balance,
		"inventory":   inventoryResponse,
		"coinHistory": coinHistory,
	}

	c.JSON(http.StatusOK, response)
}

// POST /api/sendCoin
func (server *Server) handleSendCoin(c *gin.Context) {
	var req SendCoinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	senderUsername := c.MustGet("username").(string)
	sender, err := server.store.GetUserByUsername(c, senderUsername)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	receiver, err := server.store.GetUserByUsername(c, req.ToUser)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse(err))
		return
	}

	if sender.ID == receiver.ID {
		c.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("cannot send coins to yourself")))
		return
	}

	arg := db.TransferTxParams{
		FromUserID: sender.ID,
		ToUserID:   receiver.ID,
		Amount:     req.Amount,
	}

	_, err = server.store.TransferTx(c, arg)
	if err != nil {
		if strings.Contains(err.Error(), "CHECK constraint") {
			c.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("insufficient balance")))
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "transfer successful",
	})
}

// GET /api/buy/:item
func (server *Server) handleBuyItem(c *gin.Context) {
	itemName := c.Param("item")

	username := c.MustGet("username").(string)
	user, err := server.store.GetUserByUsername(c, username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	item, err := server.store.GetItemByName(c, itemName)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse(fmt.Errorf("item not found")))
		return
	}

	arg := db.PurchaseTxParams{
		UserID: user.ID,
		ItemID: item.ID,
	}

	_, err = server.store.PurchaseTx(c, arg)
	if err != nil {
		if strings.Contains(err.Error(), "CHECK constraint") {
			c.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("insufficient balance")))
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "purchase successful",
	})
}
