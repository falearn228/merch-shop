# Создание контейнера с БД образом
postgres:
	docker run --name postgres -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=password -d postgres:17-alpine
# Старт контейнера с БД	образом
startdb:
	docker start postgres
# Создание контейнера БД в контейнере 
createdb:
	docker exec -it postgres createdb --username=root --owner=root shop
# Дропнуть существующую БД в контейнере	
dropdb:
	docker exec -it postgres dropdb shop
# Первоначальная миграция `вверх`
migrateup:
	migrate -path migrations -database "postgresql://root:password@db:5432/shop?sslmode=disable" -verbose up
# Первоначальная миграция `вниз`, очистка всех таблиц	
migratedown:
	migrate -path migrations -database "postgresql://root:password@db:5432/shop?sslmode=disable" -verbose down
# Генерация SQLC кода	
sqlc:
	sqlc generate
# Прогон всех тестов, а именно два E2E тестов, находящихся в папке internal/tests, тестов для SQLC кода и тестов для бизнес логики из папки /internal/api	
test:
	go test -cover ./...
# Простой запуск сервера без контейнера	
server:
	go run main.go
# Создание МОК объекта для Store.	
mock:
	mockgen -package mockdb -destination internal/mock/store.go avito-shop/internal/db/sqlc Store
####################################################################################################################################	
# Подготовка линтера
lint-prepare:
	@which golangci-lint || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.54.2

# Проверка кода линтером
lint: lint-prepare
	golangci-lint run ./... --timeout=5m

# Исправление проблем линтером
lint-fix: lint-prepare
	golangci-lint run --fix ./...

# Проверка с выводом в JSON
lint-json: lint-prepare
	golangci-lint run ./... --out-format=json > lint-report.json

# Быстрая проверка только критических линтеров
lint-fast: lint-prepare
	golangci-lint run ./... --fast --disable-all -E errcheck,gosimple,govet

# Проверка с подробным выводом
lint-verbose: lint-prepare
	golangci-lint run ./... -v --timeout=5m	
####################################################################################################################################
# Установка k6
install-k6:
	curl -sSL https://github.com/grafana/k6/releases/download/v0.45.0/k6-v0.45.0-linux-amd64.tar.gz | tar -xz
	sudo mv k6-v0.45.0-linux-amd64/k6 /usr/local/bin/

# Запуск всех тестов
load-test: install-k6
	k6 run load-tests/scenarios.js

# Запуск теста авторизации
load-test-auth: install-k6
	k6 run --scenario auth_test load-tests/scenarios.js

# Запуск теста покупок
load-test-purchase: install-k6
	k6 run --scenario purchase_test load-tests/scenarios.js

# Запуск теста передачи монет
load-test-transfer: install-k6
	k6 run --scenario transfer_test load-tests/scenarios.js

# Запуск с экспортом метрик в InfluxDB
load-test-metrics: install-k6
	k6 run --out influxdb=http://localhost:8086/k6 load-tests/scenarios.js	
####################################################################################################################################
# Сборка образов
build:
	docker-compose build

# Запуск всех сервисов
up:
	docker-compose up -d

# Остановка всех сервисов
down:
	docker-compose down

# Просмотр логов
logs:
	docker-compose logs -f

# Статус сервисов
ps:
	docker-compose ps

# Очистка всех данных
clean:
	docker-compose down -v
	docker system prune -f

# Перезапуск конкретного сервиса
restart-service:
	docker-compose restart $(service)
####################################################################################################################################
.PHONY: postgres start createdb dropdb migrateup migratedown sqlc test server mock lint lint-fix lint-check load-test load-test-auth load-test-purchase load-test-transfer build up down logs ps clean
