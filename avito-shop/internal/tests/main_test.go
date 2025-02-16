package e2e

import (
	"context"
	"log"
	"os"
	"testing"

	db "avito-shop/internal/db/sqlc"
	"avito-shop/internal/util"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	testDB    *pgxpool.Pool
	testStore db.Store
)

var testItems = []struct {
	name  string
	price int32
}{
	{"t-shirt", 80},
	{"cup", 20},
	{"book", 50},
	{"pen", 10},
	{"powerbank", 200},
	{"hoody", 300},
	{"umbrella", 200},
	{"socks", 10},
	{"wallet", 50},
	{"pink-hoody", 500},
}

func TestMain(m *testing.M) {
	// Устанавливаем тестовый режим Gin до всех операций
	gin.SetMode(gin.TestMode)

	var err error

	// Загружаем конфигурацию
	config, err := util.LoadConfig("../../")
	if err != nil {
		log.Fatalf("cannot load config: %v", err)
	}

	// Подключаемся к БД
	testDB, err = pgxpool.New(context.Background(), config.DBSourceTest)
	if err != nil {
		log.Fatalf("cannot connect to db: %v", err)
	}

	// Проверяем подключение
	if err := testDB.Ping(context.Background()); err != nil {
		log.Fatalf("cannot ping db: %v", err)
	}

	// Инициализируем store
	testStore = db.NewStore(testDB)

	// Инициализируем тестовые данные
	if err := initTestData(context.Background()); err != nil {
		log.Fatalf("cannot initialize test  %v", err)
	}

	// Запускаем тесты
	exitCode := m.Run()

	// Очищаем данные
	if err := cleanupTestData(context.Background()); err != nil {
		log.Printf("warning: failed to cleanup test  %v", err)
	}

	// Закрываем соединение с БД
	testDB.Close()

	os.Exit(exitCode)
}

func initTestData(ctx context.Context) error {
	// Очищаем существующие данные
	if err := cleanupTestData(ctx); err != nil {
		return err
	}

	// Создаем тестовые товары
	for _, item := range testItems {
		_, err := testStore.CreateItem(ctx, db.CreateItemParams{
			Name:  item.name,
			Price: item.price,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func cleanupTestData(ctx context.Context) error {
	queries := []string{
		"DELETE FROM purchases",
		"DELETE FROM transactions",
		"DELETE FROM items",
		"DELETE FROM users",
	}

	for _, query := range queries {
		if _, err := testDB.Exec(ctx, query); err != nil {
			return err
		}
	}

	return nil
}
