APP_NAME=myapp
APP_PATH=cmd/api/main.go
GO_MOD_PATH=go.mod
LINT_THRESHOLD=10
GOBIN=$(CURDIR)/bin
POSTGRES_URL = "postgres://test:test@localhost:5432/testdb?sslmode=disable"

export GOBIN

.PHONY: up down test-integration migrate clean-db



default: build

build:
	@echo "Сборка приложения..."
	go build -o bin/$(APP_NAME) $(APP_PATH)
	goose -dir migrations postgres "postgres://user:password@localhost:5432/orders?sslmode=disable" up
	@echo "Сборка завершена. Исполняемый файл: bin/$(APP_NAME)"

deps:
	@echo "Установка зависимостей..."
	go mod tidy
	@echo "Зависимости установлены."

run:
	go run $(APP_PATH)

clean:
	@echo "Очистка билдов..."
	rm -rf bin/
	@echo "Очистка завершена."

 docker-services-up:
	@echo "Запуск сервисов метрик через docker-compose..."
	docker-compose -f docker-compose-services.yml up -d
	@echo "Docker-compose метрик запущен."

docker-services-down:
	@echo "Остановка и удаление сервисов метрик docker-compose..."
	docker-compose -f docker-compose-services.yml down
	@echo "Docker-compose метрик остановлен."

docker-tests-up:
	@echo "Запуск сервисов метрик через docker-compose..."
	docker-compose -f docker-compose-tests.yml up -d
	@echo "Docker-compose метрик запущен."

docker-tests-down:
	@echo "Остановка и удаление сервисов тестов docker-compose..."
	docker-compose -f docker-compose-tests.yml down
	@echo "Docker-compose тестов остановлен."

test-integration:
	go test -v -tags=integration ./tests/integration/...

test-unit:
	go test -v -tags=unit ./tests/unit/...

test-load:
	go test -v -timeout 10m -run TestLoad ./tests/load

migrate:
	goose -dir ./migrations postgres $(POSTGRES_URL) up


help:
	@echo "Доступные команды:"
	@echo "  make build         		- Собрать приложение"
	@echo "  make deps          		- Установить/обновить зависимости"
	@echo "  make run           		- Запустить приложение (если оно собрано)"
	@echo "  make clean         		- Очистить билды"
	@echo "  make docker-services-up    - Поднять сервисы с помощью docker-compose"
	@echo "  make docker-services-down  - Остановить и удалить сервисы docker-compose"
	@echo "  make docker-tests-up   	- Поднять сервисы тестов docker-compose"
	@echo "  make docker-tests-down   	- Остановить и удалить сервисы тестов docker-compose"
	@echo "  make test-integration  	- Интеграционные тесты"
	@echo "  make test-unit  			- Unit тесты"
	@echo "  make test-load             - Нагрузочный тест"
	@echo "  make help          		- Показать эту справку"
	@echo "  make migrate				- Накатить миграции для тестовой БД"
