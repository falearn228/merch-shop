package main

import (
	api "avito-shop/internal/api"
	db "avito-shop/internal/db/sqlc"
	"avito-shop/internal/util"
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
)

func main() {
	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal("can't load configurations: ", err)
	}

	tokenConfig := api.TokenConfig{
		TokenSymmetricKey:   config.TokenKey,
		AccessTokenDuration: 24 * time.Hour,
	}

	conn, err := pgxpool.New(context.Background(), config.DBSource)
	if err != nil {
		log.Fatalln("can't connect to DB: ", err)
	}
	defer conn.Close()

	store := db.NewStore(conn)
	server, err := api.NewServer(store, tokenConfig)

	err = server.Start(config.Address)
	if err != nil {
		log.Fatalln("cant't start a server: ", err)
	}
}
