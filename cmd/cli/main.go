package main

import (
	"bufio"
	"fmt"
	"os"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/config"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/delivery"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/repository"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/service"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/storage/json_storage"
)

func main() {
	cfg := config.Load()

	jsonStorage := jsonstorage.NewJSONOrderStorage(cfg.OrderStoragePath)

	repo := repository.NewRepository(
		jsonStorage,
		jsonStorage,
		jsonStorage,
	)

	service := service.NewStorageService(repo)

	cli := delivery.NewCLIHandler(service)

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("Добро пожаловать в систему управления заказами!")
	fmt.Println("Введите 'help' для получения списка команд.")

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		input := scanner.Text()

		if input == "exit" {
			fmt.Println("Завершение работы...")
			return
		}

		if err := cli.HandleCommand(input); err != nil {
			fmt.Printf("Ошибка: %v\n", err)
		}
	}
}
