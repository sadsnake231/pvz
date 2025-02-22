package main

import (
	"bufio"
	"fmt"
	"os"

	"gitlab.ozon.dev/sadsnake2311/homework/hw-1/internal/delivery"
	"gitlab.ozon.dev/sadsnake2311/homework/hw-1/internal/service"
	"gitlab.ozon.dev/sadsnake2311/homework/hw-1/internal/storage/json_storage"
	"gitlab.ozon.dev/sadsnake2311/homework/hw-1/internal/config"
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

		if input == "help" {
			cliHandler.HandleHelp()
			continue
		}

		if err := cliHandler.HandleCommand(input); err != nil {
			fmt.Printf("Ошибка: %v\n", err)
		}
	}
}
