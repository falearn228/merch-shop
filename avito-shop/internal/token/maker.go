package token

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt"
)

// Различные ошибки при работе с токенами
var (
	ErrInvalidToken = errors.New("token is invalid")
	ErrExpiredToken = errors.New("token has expired")
)

// Payload содержит данные JWT токена
type Payload struct {
	Username  string    `json:"username"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Maker - интерфейс для управления токенами
type Maker interface {
	CreateToken(username string, duration time.Duration) (string, error)
	VerifyToken(token string) (*Payload, error)
}

// JWTMaker - реализация JWT токенов
type JWTMaker struct {
	secretKey string
}

func NewJWTMaker(secretKey string) (Maker, error) {
	if len(secretKey) < 32 {
		return nil, errors.New("secret key must be at least 32 characters")
	}
	return &JWTMaker{secretKey}, nil
}

func (maker *JWTMaker) CreateToken(username string, duration time.Duration) (string, error) {
	payload := &Payload{
		Username:  username,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(duration),
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username":   payload.Username,
		"issued_at":  payload.IssuedAt.Unix(),
		"expires_at": payload.ExpiresAt.Unix(),
	})

	return jwtToken.SignedString([]byte(maker.secretKey))
}

func (maker *JWTMaker) VerifyToken(token string) (*Payload, error) {
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		_, ok := token.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			return nil, ErrInvalidToken
		}
		return []byte(maker.secretKey), nil
	}

	jwtToken, err := jwt.Parse(token, keyFunc)
	if err != nil {
		return nil, err
	}

	claims, ok := jwtToken.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	expiresAt := time.Unix(int64(claims["expires_at"].(float64)), 0)
	if time.Now().After(expiresAt) {
		return nil, ErrExpiredToken
	}

	return &Payload{
		Username:  claims["username"].(string),
		IssuedAt:  time.Unix(int64(claims["issued_at"].(float64)), 0),
		ExpiresAt: expiresAt,
	}, nil
}
