package delivery

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rivo/tview"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/service"
)

type CLIHandler struct {
	service  service.StorageService
	handlers map[string]command
}

type command struct {
	Handler     func([]string) error
	Description string
}

func NewCLIHandler(service service.StorageService) *CLIHandler {
	h := &CLIHandler{service: service}
	h.handlers = h.initHandlers()
	return h
}

func (h *CLIHandler) HandleCommand(input string) error {
	args := strings.Fields(input)
	if len(args) == 0 {
		return fmt.Errorf("введите команду")
	}

	cmd := args[0]
	args = args[1:]

	command, ok := h.handlers[cmd]
	if !ok {
		return fmt.Errorf("неизвестная команда: %s", cmd)
	}

	return command.Handler(args)
}

func (h *CLIHandler) handleAccept(args []string) error {
	if len(args) != 6 {
		return fmt.Errorf("ожидается 6 аргументов: ID, RecipientID, Expiry, BasePrice, Weight, Packaging")
	}

	expiry, err := time.Parse("2006-01-02", args[2])
	if err != nil {
		return fmt.Errorf("неверный формат даты: %v", err)
	}

	basePrice, err := strconv.ParseFloat(args[3], 64)
	if err != nil {
		return fmt.Errorf("неверный формат цены: %v", err)
	}

	weight, err := strconv.ParseFloat(args[4], 64)
	if err != nil {
		return fmt.Errorf("неверный формат веса: %v", err)
	}

	packaging := domain.PackagingType(args[5])

	order := domain.Order{
		ID:          args[0],
		RecipientID: args[1],
		Expiry:      expiry.Add(24 * time.Hour).UTC(),
		BasePrice:   basePrice,
		Weight:      weight,
		Packaging:   packaging,
		Status:      domain.StatusStored,
		UpdatedAt:   time.Now().UTC(),
	}

	res, err := h.service.AcceptOrder(order)
	if err != nil {
		return err
	}
	fmt.Println(res)
	return nil
}

func (h *CLIHandler) handleJSON(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("укажите имя файла")
	}

	file, err := os.ReadFile(args[0])
	if err != nil {
		return fmt.Errorf("ошибка при чтении файла: %v", err)
	}

	var orders []domain.Order
	if err := json.Unmarshal(file, &orders); err != nil {
		return fmt.Errorf("ошибка при парсинге JSON: %v", err)
	}

	res, err := h.service.AcceptOrdersFromJSON(orders)
	if err != nil {
		return err
	}
	fmt.Println(res)
	return nil
}

func (h *CLIHandler) handleReturn(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("ожидается 1 аргумент: ID")
	}

	res, err := h.service.ReturnOrder(args[0])
	if err != nil {
		return err
	}
	fmt.Println(res)
	return nil
}

func (h *CLIHandler) handleIssueRefund(args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("ожидаются min 3 аргумента: команда, id пользователя, id заказа")
	}

	commandType := args[0]
	userID := args[1]
	orderIDs := args[2:]

	var result string
	var err error

	switch commandType {
	case "issue":
		result, err = h.service.IssueOrders(userID, orderIDs)
	case "refund":
		result, err = h.service.RefundOrders(userID, orderIDs)
	default:
		return fmt.Errorf("неверная команда: %s", commandType)
	}

	if err != nil {
		fmt.Println(result + "\n" + err.Error())
		return nil
	}
	fmt.Println(result + "\n")
	return nil
}

func (h *CLIHandler) handleList(args []string) error {
	if len(args) == 0 || len(args) > 3 {
		return fmt.Errorf("ожидается от 1 до 3 аргументов")
	}

	userID := args[0]
	limit := -1
	showStored := false

	if len(args) > 1 {
		var err error
		limit, err = strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("неверный формат лимита: %v", err)
		}
	}

	if len(args) > 2 {
		showStored = (args[2] == "yes")
	}

	orders, err := h.service.GetUserOrders(userID, limit, showStored)
	if err != nil {
		return err
	}
	h.displayOrdersWithScroll(orders)
	return nil
}

func (h *CLIHandler) handleRefunded(args []string) error {
	limit := 10
	if len(args) > 0 {
		limit, _ = strconv.Atoi(args[0])
	}

	offset := 0
	for {
		orders, err := h.service.GetRefundedOrders(limit, offset)
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

		if shouldExit := h.askForContinue(); shouldExit {
			break
		}
		offset += limit
	}
	return nil
}

func (h *CLIHandler) askForContinue() bool {
	fmt.Print("Нажмите Enter для следующей страницы или 'q' для выхода: ")
	var input string
	fmt.Scanln(&input)
	return input == "q"
}

func (h *CLIHandler) handleHistory(args []string) error {
	orders, err := h.service.GetOrderHistory()
	if err != nil {
		return err
	}
	h.displayOrders(orders)
	return nil
}

func (h *CLIHandler) handleHelp(args []string) error {
	fmt.Println("Доступные команды:")
	for cmd, details := range h.handlers {
		fmt.Printf("- %s: %s\n", cmd, details.Description)
	}
	return nil
}

func (h *CLIHandler) displayOrdersWithScroll(orders []domain.Order) {
	app := tview.NewApplication()
	list := tview.NewList().ShowSecondaryText(false)
	for _, order := range orders {
		list.AddItem(formatOrder(order), "", 0, nil)
	}
	list.SetDoneFunc(func() { app.Stop() })
	if err := app.SetRoot(list, true).Run(); err != nil {
		fmt.Printf("Ошибка отображения: %v", err)
	}
}

func (h *CLIHandler) displayOrders(orders []domain.Order) {
	for _, order := range orders {
		fmt.Println(formatOrder(order))
	}
}

func formatOrder(order domain.Order) string {
	return fmt.Sprintf(
		"ID: %s, Status: %s, Expiry: %s, Updated: %s, Price: %.2f, Weight: %.2f, Packaging: %s",
		order.ID,
		order.Status,
		order.Expiry.Format("2006-01-02"),
		order.UpdatedAt.Format("2006-01-02 15:04:05"),
		order.BasePrice+order.PackagePrice,
		order.Weight,
		order.Packaging,
	)
}

func (h *CLIHandler) initHandlers() map[string]command {
	return map[string]command{
		"accept": {
			Handler:     h.handleAccept,
			Description: "Принять заказ: accept <ID> <RecipientID> <Expiry> <BasePrice> <Weight> <Packaging>",
		},
		"return": {
			Handler:     h.handleReturn,
			Description: "Вернуть заказ: return <ID>",
		},
		"issue/refund": {
			Handler:     h.handleIssueRefund,
			Description: "Выдать или вернуть заказы: issue/refund <UserID> <OrderID1> <OrderID2> ...",
		},
		"list": {
			Handler:     h.handleList,
			Description: "Список заказов пользователя: list <UserID> [n] [yes]",
		},
		"refunded": {
			Handler:     h.handleRefunded,
			Description: "Список возвращенных заказов: refunded [limit]",
		},
		"history": {
			Handler:     h.handleHistory,
			Description: "История заказов: history",
		},
		"json": {
			Handler:     h.handleJSON,
			Description: "Принять заказы из JSON-файла: json <filename>",
		},
		"help": {
			Handler:     h.handleHelp,
			Description: "Показать справку: help",
		},
	}
}
