package db

import (
	util "avito-shop/internal/util"
	"context"
	"log"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

var testQueries *Queries
var testDB *pgxpool.Pool

func TestMain(m *testing.M) {

	var err error
	config, err := util.LoadConfig("../../../.")
	if err != nil {
		log.Fatal("can't load configurations: ", err)
	}

	dbSource := config.DBSourceTest

	testDB, err = pgxpool.New(context.Background(), dbSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}

	testQueries = New(testDB)

	os.Exit(m.Run())
}
