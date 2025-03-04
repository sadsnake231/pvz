APP_NAME=myapp
APP_PATH=cmd/cli/main.go
GO_MOD_PATH=go.mod
LINT_THRESHOLD=10
GOBIN=$(CURDIR)/bin

export GOBIN

default: build

build:
	@echo "Сборка приложения..."
	go build -o bin/$(APP_NAME) $(APP_PATH)
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

lint:
	@echo "Запуск линтеров..."
	@mkdir -p $(GOBIN)
	@if ! [ -x "$(GOBIN)/gocyclo" ]; then \
		echo "Установка gocyclo..."; \
		GOBIN=$(GOBIN) go install github.com/fzipp/gocyclo/cmd/gocyclo@latest; \
	fi
	@if ! [ -x "$(GOBIN)/gocognit" ]; then \
		echo "Установка gocognit..."; \
		GOBIN=$(GOBIN) go install github.com/uudashr/gocognit/cmd/gocognit@latest; \
	fi
	@echo "Запуск gocyclo..."
	@$(GOBIN)/gocyclo -over $(LINT_THRESHOLD) .
	@echo "Запуск gocognit..."
	@$(GOBIN)/gocognit -over $(LINT_THRESHOLD) .
	@echo "Проверка линтерами завершена."

clean:
	@echo "Очистка билдов..."
	rm -rf bin/
	@echo "Очистка завершена."

help:
	@echo "Доступные команды:"
	@echo "  make build    - Собрать приложение"
	@echo "  make deps     - Установить/обновить зависимости"
	@echo "  make run      - Запустить приложение (если оно собрано)"
	@echo "  make lint     - Запустить линтеры (установятся в ./bin, если нет)"
	@echo "  make clean    - Очистить билды"
	@echo "  make help     - Показать эту справку"
