package main

import (
	"bufio"
	"fmt"
	"os"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/delivery"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/service"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/storage/json_storage"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/config"
)

func main() {
	cfg := config.Load()

	storage := jsonstorage.NewJSONOrderStorage(cfg.OrderStoragePath)
	service := service.NewStorageService(storage, storage, storage)
	cliHandler := delivery.NewCLIHandler(service)

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("Введите команду (или 'help' для справки): ")
		scanner.Scan()
		input := scanner.Text()

		if input == "exit" {
			fmt.Print("До встречи!\n")
			return
		}

		if err := cliHandler.HandleCommand(input); err != nil {
			fmt.Printf("Ошибка: %v\n", err)
		}
	}
}
