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
	StoredAt     *time.Time    `json:"stored_at"`
	IssuedAt     *time.Time    `json:"issued_at"`
	RefundedAt   *time.Time    `json:"refunded_at"`
	BasePrice    float64       `json:"base_price"`
	PackagePrice float64       `json:"package_price"`
	Weight       float64       `json:"weight"`
	Packaging    PackagingType `json:"packaging"`
}

type ProcessedOrders struct {
	UserID   string
	OrderIDs []string
	Failed   []string
	Error    error
}

var (
	ErrExpiredOrder    = errors.New("срок хранения заказа уже прошел")
	ErrDuplicateOrder  = errors.New("заказ с таким ID уже есть")
	ErrNotExpiredOrder = errors.New("у этого заказа еще не истек срок хранения")

	ErrNotFoundOrder       = errors.New("заказа с таким ID не существует")
	ErrNotStoredOrder      = errors.New("заказа нет на складе")
	ErrNotIssuedOrder      = errors.New("заказ ещё не был выдан")
	ErrUserNoOrders        = errors.New("введенные заказы не готовы к выдаче или возврату")
	ErrUserNoActiveOrders  = errors.New("у пользователя нет активных заказов")
	ErrRefundPeriodExpired = errors.New("прошло 48 суток с момента выдачи заказа")

	ErrInvalidWeight = errors.New("слишком большой вес для этой упаковки")

	ErrWrongJSON = errors.New("тело запроса содержит ошибки")
	ErrDatabase  = errors.New("ошибка базы данных")
	ErrCache     = errors.New("ошибка кэша")
)

type ErrUserDoesntOwnOrder struct {
	OrderID string
	UserID  string
}

func (e *ErrUserDoesntOwnOrder) Error() string {
	return fmt.Sprintf("Заказ %s не принадлежит пользователю %s", e.OrderID, e.UserID)
}

func (o Order) Status() OrderStatus {
	var latestTime time.Time
	var status OrderStatus

	if o.RefundedAt != nil && o.RefundedAt.After(latestTime) {
		latestTime = *o.RefundedAt
		status = StatusRefunded
	}
	if o.IssuedAt != nil && o.IssuedAt.After(latestTime) {
		latestTime = *o.IssuedAt
		status = StatusIssued
	}
	if o.StoredAt != nil && o.StoredAt.After(latestTime) {
		status = StatusStored
	}

	return status
}

func (o Order) LastUpdated() time.Time {
	times := []time.Time{}
	if o.StoredAt != nil {
		times = append(times, *o.StoredAt)
	}
	if o.IssuedAt != nil {
		times = append(times, *o.IssuedAt)
	}
	if o.RefundedAt != nil {
		times = append(times, *o.RefundedAt)
	}

	if len(times) == 0 {
		return time.Time{}
	}

	maxTime := times[0]
	for _, t := range times[1:] {
		if t.After(maxTime) {
			maxTime = t
		}
	}
	return maxTime
}
