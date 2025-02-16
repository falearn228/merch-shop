package api

import (
	db "avito-shop/internal/db/sqlc"
	middleware "avito-shop/internal/middleware"
	"avito-shop/internal/token"
	"avito-shop/internal/util"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type TokenConfig struct {
	TokenSymmetricKey   string        `mapstructure:"TOKEN_SYMMETRIC_KEY"`
	AccessTokenDuration time.Duration `mapstructure:"ACCESS_TOKEN_DURATION"`
}

type Server struct {
	config     TokenConfig
	store      db.Store
	tokenMaker token.Maker
	Router     *gin.Engine
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func NewServer(store db.Store, config TokenConfig) (*Server, error) {
	tokenMaker, err := token.NewJWTMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, err
	}

	server := &Server{
		config:     config,
		store:      store,
		tokenMaker: tokenMaker,
	}

	server.setupRouter()
	return server, nil
}

func (server *Server) setupRouter() {
	router := gin.Default()

	// Публичные маршруты
	router.POST("/api/auth", server.handleLogin)

	// Защищенные маршруты
	protected := router.Group("/api").Use(middleware.AuthMiddleware(server.tokenMaker))
	{
		protected.GET("/info", server.handleGetInfo)
		protected.GET("/buy/:item", server.handleBuyItem)
		protected.POST("/sendCoin", server.handleSendCoin)
	}

	server.Router = router
}

func (server *Server) handleLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var username string
	var hashedPassword string

	user, err := server.store.GetUserByUsername(c, req.Username)
	if err != nil {
		if err == pgx.ErrNoRows {
			// Пользователь не найден, создаем нового
			hashedPassword, err = util.HashPassword(req.Password)
			if err != nil {
				c.JSON(http.StatusInternalServerError, errorResponse(err))
				return
			}

			arg := db.CreateUserParams{
				Username:     req.Username,
				PasswordHash: hashedPassword,
			}

			newUser, err := server.store.CreateUser(c, arg)
			if err != nil {
				c.JSON(http.StatusInternalServerError, errorResponse(err))
				return
			}
			username = newUser.Username

		} else {
			c.JSON(http.StatusInternalServerError, errorResponse(err))
			return
		}
	} else {
		// Пользователь существует, проверяем пароль
		err = util.CheckPassword(req.Password, user.PasswordHash)
		if err != nil {
			c.JSON(http.StatusUnauthorized, errorResponse(err))
			return
		}
		username = user.Username
	}

	token, err := server.tokenMaker.CreateToken(
		username,
		server.config.AccessTokenDuration,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
	})
}

func (server *Server) Start(address string) error {
	return server.Router.Run(address)
}

func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}
