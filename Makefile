APP_NAME=myapp
APP_PATH=cmd/api/main.go
GO_MOD_PATH=go.mod
LINT_THRESHOLD=10
GOBIN=$(CURDIR)/bin

export GOBIN

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

docker-up:
	@echo "Запуск сервисов через docker-compose..."
	docker-compose up -d
	@echo "Docker-compose запущен."

docker-down:
	@echo "Остановка и удаление сервисов docker-compose..."
	docker-compose down
	@echo "Docker-compose остановлен."

help:
	@echo "Доступные команды:"
	@echo "  make build         - Собрать приложение"
	@echo "  make deps          - Установить/обновить зависимости"
	@echo "  make run           - Запустить приложение (если оно собрано)"
	@echo "  make clean         - Очистить билды"
	@echo "  make docker-up     - Поднять сервисы с помощью docker-compose"
	@echo "  make docker-down   - Остановить и удалить сервисы docker-compose"
	@echo "  make help          - Показать эту справку"
