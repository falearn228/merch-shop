# merch-shop
A store created for the purchase of merch by users and the ability to transfer coins to others.

## Основные моменты

Используемые технологии -> Docker, Docker-compose, Grafana, PostgreSQL, Golang (SQLC, gomock, testify, Gin, Viper) 

`migrations` - файл с миграциями для БД \
`internal` - внутренний код проекта 
* `internal/tests` - E2E тесты для покупки мерча и передачи монет сотрудникам. 

`cmd/server` - основной файл сервиса 

Реализованы юнит-тесты для SQLC кода, API бизнес логики. 

## Требования
- Docker
- Docker Compose

## Быстрый старт

1. Клонируйте репозиторий любым удобным спосом, пример с https:
```bash
git clone https://github.com/falearn228/merch-shop.git
cd merch-shop/avito-shop
```

В **Makefile** описаны все возможные команды. \
в **app.config.env** установлены переменные окружения для подключения к БД и настройки токена. 

2. Собираем, скачиваем контейнеры, перейдя в папку **avito-shop**
```bash
make build
```

3. Поднимаем контейнеры:
```bash
make up

# Проверяем, что все запустилось
docker-compose ps

# Остановка всех сервисов, при необходимости завершить работу
make down
```

4. Мониторинг:
```bash
# Grafana доступна по адресу
open http://localhost:3000

# Логин: admin
# Пароль: admin (установлен в docker-compose.yml)
```

5. Использование **API**:
```bash
# Регистрация/вход пользователя
curl -X POST http://localhost:8080/api/auth \
  -H "Content-Type: application/json" \
  -d '{"username":"test_user","password":"password123"}'

# Получаем в ответ токен и сохраняем его
TOKEN="полученный_токен"

# Проверка баланса и инвентаря
curl http://localhost:8080/api/info \
  -H "Authorization: Bearer $TOKEN"

# Покупка товара
curl http://localhost:8080/api/buy/t-shirt \
  -H "Authorization: Bearer $TOKEN"

# Отправка монет другому пользователю
curl -X POST http://localhost:8080/api/sendCoin \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"toUser":"другой_пользователь","amount":100}'
```

7. Нагрузочное тестирование (для некоторых команд потребуется режим **sudo**):
```bash
# Установка k6
make install-k6

# Запуск всех тестов, сценарии находятся в load-tests
make load-test

# Или конкретного сценария
make load-test-auth
make load-test-purchase
make load-test-transfer
```

8. Юнит и E2E тесты
```bash
# После старта контейнеров 
make test
```

9. Линтер для Go (Скорее всего потребуется **sudo** для установки)
```bash
sudo make lint
```
