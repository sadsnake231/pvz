package delivery

import (
	"fmt"
	"strconv"
	"strings"
	"github.com/rivo/tview"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/service"
)

type CLIHandler struct {
	service service.StorageService
}

func NewCLIHandler(service service.StorageService) *CLIHandler {
	return &CLIHandler{service: service}
}

func (h *CLIHandler) HandleCommand(input string) error {
	args := strings.Fields(input)
	if len(args) == 0 {
		return fmt.Errorf("введите команду")
	}

	cmd := args[0]
	args = args[1:]

	handlers := map[string]func([]string) error{
		"accept":       h.handleAccept,
		"return":       h.handleReturn,
		"issue/refund": h.handleIssueRefund,
		"list":         h.handleList,
		"refunded":     h.handleRefunded,
		"history":      h.handleHistory,
		"json":         h.handleJSON,
		"help":         h.handleHelp,
	}

	handler, ok := handlers[cmd]
	if !ok {
		return fmt.Errorf("неизвестная команда: %s", cmd)
	}

	return handler(args)
}

func (h *CLIHandler) handleAccept(args []string) error {
	res, err := h.service.AcceptOrder(args)
	if err != nil {
		return err
	}
	fmt.Println(res)
	return nil
}

func (h *CLIHandler) handleReturn(args []string) error {
	res, err := h.service.ReturnOrder(args)
	if err != nil {
		return err
	}
	fmt.Println(res)
	return nil
}

func (h *CLIHandler) handleIssueRefund(args []string) error {
	res, err := h.service.IssueRefundOrders(args)
	if err != nil {
		fmt.Println(res + "\n" + err.Error())
		return nil
	}
	fmt.Println(res + "\n")
	return nil
}

func (h *CLIHandler) handleList(args []string) error {
	orders, err := h.service.GetUserOrders(args)
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

func (h *CLIHandler) handleJSON(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("укажите имя файла")
	}
	res, err := h.service.AcceptOrdersFromJSONFile(args[0])
	if err != nil {
		return err
	}
	fmt.Println(res)
	return nil
}

func (h *CLIHandler) handleHelp(args []string) error {
	helpText, err := h.service.Help()
	if err != nil {
		return err
	}
	fmt.Println(helpText)
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
		order.FinalPrice,
		order.Weight,
		order.Packaging,
	)
}
