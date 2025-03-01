
APP_NAME=myapp
APP_PATH=cmd/cli/main.go
GO_MOD_PATH=go.mod
LINT_THRESHOLD=10

default: build

build: lint
	@echo "Сборка приложения..."
	go build -o bin/$(APP_NAME) $(APP_PATH)
	@echo "Сборка завершена. Исполняемый файл: bin/$(APP_NAME)"

deps:
	@echo "Установка зависимостей..."
	go mod tidy
	@echo "Зависимости установлены."

run: build
	@echo "Запуск приложения..."
	./bin/$(APP_NAME)

lint:
	@echo "Запуск линтеров..."
	@if ! command -v gocyclo > /dev/null; then \
		echo "Установка gocyclo..."; \
		go install github.com/fzipp/gocyclo/cmd/gocyclo@latest; \
	fi
	@if ! command -v gocognit > /dev/null; then \
		echo "Установка gocognit..."; \
		go install github.com/uudashr/gocognit/cmd/gocognit@latest; \
	fi
	@echo "Запуск gocyclo..."
	@gocyclo -over $(LINT_THRESHOLD) .
	@echo "Запуск gocognit..."
	@gocognit -over $(LINT_THRESHOLD) .
	@echo "Проверка линтерами завершена."

clean:
	@echo "Очистка билдов..."
	rm -rf bin/
	@echo "Очистка завершена."

help:
	@echo "Доступные команды:"
	@echo "  make build    - Собрать приложение"
	@echo "  make deps     - Установить/обновить зависимости"
	@echo "  make run      - Запустить приложение"
	@echo "  make lint     - Запустить линтеры"
	@echo "  make clean    - Очистить билды"
	@echo "  make help     - Показать эту справку"
