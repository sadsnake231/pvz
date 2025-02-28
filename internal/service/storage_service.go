package service

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
	"gitlab.ozon.dev/sadsnake2311/homework/internal/repository"
)

type StorageService interface {
	AcceptOrder(args []string) (string, error)
	AcceptOrdersFromJSONFile(filename string) (string, error)
	ReturnOrder(args []string) (string, error)
	IssueRefundOrders(args []string) (string, error)
	GetUserOrders(args []string) ([]domain.Order, error)
	GetRefundedOrders(limit, offset int) ([]domain.Order, error)
	GetOrderHistory() ([]domain.Order, error)
	Help() (string, error)
}

type storageService struct {
	repo repository.OrderRepository
}

func NewStorageService(repo repository.OrderRepository) StorageService {
	return &storageService{repo: repo}
}

func (s *storageService) AcceptOrder(args []string) (string, error) {
	if len(args) != 6 {
		return "", fmt.Errorf("ожидается 6 аргументов: ID, RecipientID, Expiry, BasePrice, BaseWeight, Packaging")
	}
	expiry, err := time.Parse("2006-01-02", args[2])
	if err != nil {
		return "", fmt.Errorf("неверный формат даты: %v", err)
	}
	basePrice, err := strconv.ParseFloat(args[3], 64)
	if err != nil {
		return "", fmt.Errorf("неверный формат цены: %v", err)
	}
	weight, err := strconv.ParseFloat(args[4], 64)
	if err != nil {
		return "", fmt.Errorf("неверный формат веса: %v", err)
	}
	packaging := domain.PackagingType(args[5])
	order := domain.Order{
		ID:          	args[0],
		RecipientID: 	args[1],
		Expiry:      	expiry.Add(24 * time.Hour).UTC(),
		BasePrice:   	basePrice,
		Weight:		weight,
		Packaging:   	packaging,
		Status:      	domain.StatusStored,
		UpdatedAt:   	time.Now().UTC(),
	}
	if err := s.repo.AcceptOrder(order); err != nil {
		return "", err
	}
	return fmt.Sprintf("Заказ принят!"), nil

}

type OrderFromJSON struct {
	ID          string  `json:"id"`
	RecipientID string  `json:"recipient_id"`
	Expiry      string  `json:"expiry"`
	BasePrice   float64 `json:"base_price"`
	Weight      float64 `json:"weight"`
	Packaging   string  `json:"packaging"`
}

func (s *storageService) AcceptOrdersFromJSONFile(filename string) (string, error) {
	file, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("ошибка при чтении файла: %v", err)
	}

	if len(file) == 0 {
		return "", fmt.Errorf("файл пуст")
	}

	var orders []OrderFromJSON
	if err := json.Unmarshal(file, &orders); err != nil {
		return "", fmt.Errorf("ошибка при парсинге JSON: %v", err)
	}

	for _, order := range orders {
		basePrice := fmt.Sprintf("%.2f", order.BasePrice)
		weight := fmt.Sprintf("%.2f", order.Weight)

		if _, err := s.AcceptOrder([]string{
			order.ID,
			order.RecipientID,
			order.Expiry,
			basePrice,
			weight,
			order.Packaging,
		}); err != nil {
			return "", fmt.Errorf("ошибка при обработке заказа %s: %v", order.ID, err)
		}
	}

	return "Заказы через JSON приняты!", nil
}

func (s *storageService) ReturnOrder(args []string) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("ожидается 1 аргумент: ID")
	}

	id, err := s.repo.ReturnOrder(args[0])
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Заказ %s возвращен курьеру!", id), nil
}

func (s *storageService) IssueRefundOrders(args []string) (string, error) {
	if len(args) < 3 {
		return "", fmt.Errorf("ожидаются min 3 аргумента: команда, id пользователя, id заказа")
	}

	var id string
	var processedOrders []string
	var err error

	switch args[0] {
		case "issue":
			id, processedOrders, err = s.repo.IssueOrders(args[1], args[2:])
		case "refund":
			id, processedOrders, err = s.repo.RefundOrders(args[1], args[2:])
		default:
			return "", fmt.Errorf("неверная команда")
	}

	return fmt.Sprintf("id пользователя: %s\nУспешно обработанные заказы: %s", id, strings.Join(processedOrders, ", ")), err
}

func (s *storageService) GetUserOrders(args []string) ([]domain.Order, error) {
	if len(args) == 0 || len(args) > 3 {
		return nil, fmt.Errorf("ожидается от 1 до 3 аргументов")
	}

	userID := args[0]
	limit := -1
	status := ""

	if len(args) > 1 {
		if n, err := strconv.Atoi(args[1]); err == nil {
			limit = n
		}
	}

	if len(args) > 2 && args[2] == "yes" {
		status = string(domain.StatusStored)
	}

	return s.repo.GetUserOrders(userID, limit, status, 0)
}

func (s *storageService) GetRefundedOrders(limit, offset int) ([]domain.Order, error) {
	return s.repo.GetRefundedOrders(limit, offset)
}

func (s *storageService) GetOrderHistory() ([]domain.Order, error) {
	return s.repo.GetOrderHistory()
}

func (s *storageService) Help() (string, error) {
	return `Доступные команды:
	accept <ID> <RecipientID> <Expiry> - Принять заказ
	return <ID> - Вернуть заказ доставке
	issue/refund <UserID> <OrderID1> <OrderID2> ... - Выдать/вернуть заказы
	list <UserID> [n] [yes] - Список заказов
	refunded [limit] - Возвращенные заказы
	history - История заказов
	json <filename> - Загрузить из JSON
	help - Справка
	exit - Выход`, nil
}
