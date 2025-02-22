package delivery

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rivo/tview"
	"gitlab.ozon.dev/sadsnake2311/homework/hw-1/internal/domain"
	"gitlab.ozon.dev/sadsnake2311/homework/hw-1/internal/service"
)

type CLIHandler interface {
	HandleCommand(input string) error
	HandleHelp()
}

type cliHandler struct {
	storageService *service.StorageService
}

func NewCLIHandler(storageService *service.StorageService) CLIHandler {
	return &cliHandler{storageService: storageService}
}

func (h *cliHandler) HandleCommand(input string) error {
	args := strings.Fields(input)
	if len(args) == 0 {
		return fmt.Errorf("введите команду")
	}

	command := args[0]
	args = args[1:]

	switch command {
		case "accept":
			err := h.HandleAcceptOrder(args)
			if err != nil {
				return err
			}
			fmt.Printf("Заказ принят на склад!\n")
		case "return":
			id, err := h.HandleReturnOrder(args)
			if err != nil {
				return err
			}
			fmt.Printf("Заказ %s успешно возвращен курьеру!\n", id)
		case "issue":
			userID, orders, err := h.HandleIssueOrders(args[:1], args[1:])
			fmt.Printf("Пользователь: %s\nУспешно выданные заказы: %s\n Ошибка: %v\n", userID, strings.Join(orders, ", "), err)
		case "refund":
			userID, orders, err := h.HandleRefundOrders(args[:1], args[1:])
			fmt.Printf("Пользователь: %s\nУспешно возвращенные заказы: %s\n Ошибка: %v\n", userID, strings.Join(orders, ", "), err)
		case "list":
			userID := args[0]
			n := -1
			if len(args) > 1 {
				n, _ = strconv.Atoi(args[1])
			}
			status := ""
			if len(args) > 2 && args[2] == "yes" {
				status = "stored"
			}
			orders, err := h.storageService.GetUserOrders(userID, n, status, 0)
			if err != nil {
				return err
			}
			app := tview.NewApplication()
			list := tview.NewList().
			ShowSecondaryText(false).
			SetDoneFunc(func() {
				app.Stop()
			})
			for _, order := range orders {
				list.AddItem(formatOrder(order), "", 0, nil)
			}
			if err := app.SetRoot(list, true).Run(); err != nil {
				return fmt.Errorf("ошибка при запуске интерфейса: %v", err)
			}
		case "refunded":
			limit := 10
			if len(args) > 0 {
				limit, _ = strconv.Atoi(args[0])
			}
			offset := 0
			for {
				orders, err := h.HandleGetRefundedOrders(limit, offset)
				if err != nil {
					return err
				}
				if len(orders) == 0 {
					fmt.Println("Больше нет возвращенных заказов.")
					break
				}
				for _, order := range orders {
					fmt.Println(formatOrder(order))
				}
				fmt.Print("Нажмите Enter для следующей страницы или 'q' для выхода: ")
				var input string
				fmt.Scanln(&input)
				if input == "q" {
					break
				}
				offset += limit
			}
		case "history":
			orders, err := h.HandleGetOrderHistory()
			if err != nil {
				return err
			}
			for _, order := range orders {
				fmt.Println(formatOrder(order))
			}
		case "json":
			if len(args) == 0 {
				return fmt.Errorf("укажите имя файла")
			}
			if err := h.HandleAcceptOrdersFromJSONFile(args[0]); err != nil {
				return err
			}
			fmt.Println("Заказы успешно приняты!")
		default:
			return fmt.Errorf("неизвестная команда: %s", command)
	}

	return nil
}

func (h *cliHandler) HandleHelp() {
	helpText := `
	Доступные команды:
	accept <ID> <RecipientID> <Expiry> - Принять заказ
	return <ID> - Вернуть заказ доставке
	issue <UserID> <OrderID1> <OrderID2> ... - Выдать заказы пользователю
	refund <UserID> <OrderID1> <OrderID2> ... - Вернуть заказы от пользователя
	list <UserID> [n] [yes] - Получить список заказов пользователя со скроллом (выход через Ctrl+C)
	refunded [limit] - Получить список возвращенных заказов с постраничной пагинацией
	history - Получить историю заказов
	json <filename> - Принять заказы из JSON файла. Файл должен лежать в корне. Пример: json delivery.json
	help - Показать эту справку
	`
	fmt.Println(helpText)
}

func (h *cliHandler) HandleAcceptOrder(args []string) error {
	if len(args) != 3 {
		return fmt.Errorf("ожидается 3 аргумента: ID, RecipientID, Expiry")
	}

	id := args[0]
	recipientID := args[1]
	expiryStr := args[2]

	expiry, err := time.Parse("2006-01-02", expiryStr)
	if err != nil {
		return fmt.Errorf("неверный формат даты: %v", err)
	}

	expiry = expiry.Add(24 * time.Hour) //заказ будет считаться просроченным с 00-00 следующего дня

	order := domain.Order{
		ID:          id,
		RecipientID: recipientID,
		Expiry:      expiry.UTC(),
	}

	return h.storageService.AcceptOrder(order)
}

func (h *cliHandler) HandleReturnOrder(args []string) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("ожидается 1 аргумент: ID")
	}

	id := args[0]

	return h.storageService.ReturnOrder(id)
}

func (h *cliHandler) HandleIssueOrders(userArgs, idsArgs []string) (string, []string, error) {
	if len(userArgs) != 1 {
		return "", nil, fmt.Errorf("ожидается 1 аргумент: ID")
	}

	if len(idsArgs) == 0 {
		return "", nil, fmt.Errorf("список ID заказов не может быть пустым")
	}

	id := userArgs[0]

	return h.storageService.IssueOrders(id, idsArgs)
}

func (h *cliHandler) HandleRefundOrders(userArgs, idsArgs []string) (string, []string, error) {
	if len(userArgs) != 1 {
		return "", nil, fmt.Errorf("ожидается 1 аргумент: ID")
	}

	if len(idsArgs) == 0 {
		return "", nil, fmt.Errorf("список ID заказов не может быть пустым")
	}

	id := userArgs[0]

	return h.storageService.RefundOrders(id, idsArgs)
}

func (h *cliHandler) HandleGetUserOrders(userArgs, nArgs, storedArgs []string) (string, error) {
	if len(userArgs) != 1 {
		return "", fmt.Errorf("ожидается 1 аргумент: ID")
	}

	if len(nArgs) > 1 {
		return "", fmt.Errorf("ожидается 0 или 1 аргумент: N")
	}

	if len(storedArgs) > 1 {
		return "", fmt.Errorf("ожидается 0 или 1 аргумент: stored")
	}

	userID := userArgs[0]

	limit := -1
	if len(nArgs) == 1 {
		_, err := fmt.Sscanf(nArgs[0], "%d", &limit)
		if err != nil {
			return "", fmt.Errorf("неверный формат числа: %v", err)
		}
	}

	status := ""
	if len(storedArgs) == 1 && storedArgs[0] == "yes" {
		status = string(domain.StatusStored)
	}

	orders, err := h.storageService.GetUserOrders(userID, limit, status, 0)
	if err != nil {
		return "", err
	}

	result := ""
	for _, order := range orders {
		result += fmt.Sprintf("ID: %s, Status: %s, Expiry: %s\n", order.ID, order.Status, order.Expiry)
	}

	return result, nil
}

func (h *cliHandler) HandleGetRefundedOrders(limit, offset int) ([]domain.Order, error) {
	return h.storageService.GetRefundedOrders(limit, offset)
}

func (h *cliHandler) HandleGetOrderHistory() ([]domain.Order, error) {
	return h.storageService.GetOrderHistory()
}

type OrderFromJSON struct {
	ID          string `json:"id"`
	RecipientID string `json:"recipient_id"`
	Expiry      string `json:"expiry"`
}

func (h *cliHandler) HandleAcceptOrdersFromJSONFile(fileName string) error {
	file, err := os.ReadFile(fileName)
	if err != nil {
		return fmt.Errorf("ошибка при чтении файла: %v", err)
	}

	var orders []OrderFromJSON
	if err := json.Unmarshal(file, &orders); err != nil {
		return fmt.Errorf("ошибка при парсинге JSON: %v", err)
	}

	for _, order := range orders {
		args := []string{order.ID, order.RecipientID, order.Expiry}
		if err := h.HandleAcceptOrder(args); err != nil {
			return fmt.Errorf("ошибка при обработке заказа %s: %v", order.ID, err)
		}
	}

	return nil
}

func formatOrder(order domain.Order) string {
	return fmt.Sprintf("ID: %s, Status: %s, Expiry: %s, Last Updated: %s", order.ID, order.Status, order.Expiry, order.UpdatedAt)
}
