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
	goose -dir internal/db/migrations postgres "postgres://user:password@localhost:5432/orders?sslmode=disable" up
	@echo "Сборка завершена. Исполняемый файл: bin/$(APP_NAME)"

deps:
	@echo "Установка зависимостей..."
	go mod tidy
	@echo "Зависимости установлены."

run:
	@if [ ! -f bin/$(APP_NAME) ]; then \
		echo "Ошибка: приложение не собрано. Сначала выполните 'make build'"; \
		exit 1; \
	fi
	@echo "Запуск приложения..."
	./bin/$(APP_NAME)
clean:
	@echo "Очистка билдов..."
	rm -rf bin/
	@echo "Очистка завершена."

help:
	@echo "Доступные команды:"
	@echo "  make build    - Собрать приложение"
	@echo "  make deps     - Установить/обновить зависимости"
	@echo "  make run      - Запустить приложение (если оно собрано)"
	@echo "  make clean    - Очистить билды"
	@echo "  make help     - Показать эту справку"
