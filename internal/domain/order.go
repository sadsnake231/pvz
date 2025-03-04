package domain

import (
	"errors"
	"fmt"
	"time"
)

type OrderStatus string

const (
	StatusStored   OrderStatus = "stored"
	StatusIssued   OrderStatus = "issued"
	StatusRefunded OrderStatus = "refunded"
)

type Order struct {
	ID           string        `json:"id"`
	RecipientID  string        `json:"recipient_id"`
	Expiry       time.Time     `json:"expiry"`
	Status       OrderStatus   `json:"status"`
	UpdatedAt    time.Time     `json:"UpdatedAt"`
	BasePrice    float64       `json:"base_price"`
	PackagePrice float64       `json:"package_price"`
	Weight       float64       `json:"weight"`
	Packaging    PackagingType `json:"packaging"`
}

type ProcessedOrders struct {
	UserID   string
	OrderIDs []string
	Error    error
}

var (
	ErrExpiredOrder    = errors.New("Срок хранения заказа уже прошел")
	ErrDuplicateOrder  = errors.New("Заказ с таким ID уже есть")
	ErrNotExpiredOrder = errors.New("У этого заказа еще не истек срок хранения")

	ErrNotFoundOrder       = errors.New("Заказа с таким ID не существует")
	ErrNotStoredOrder      = errors.New("Заказа нет на складе")
	ErrUserNoOrders        = errors.New("Введенные заказы не готовы к выдаче или возврату")
	ErrRefundPeriodExpired = errors.New("Прошло 48 суток с момента выдачи заказа")
)

type ErrUserDoesntOwnOrder struct {
	OrderID string
	UserID  string
}

func (e *ErrUserDoesntOwnOrder) Error() string {
	return fmt.Sprintf("Заказ %s не принадлежит пользователю %s", e.OrderID, e.UserID)
}
