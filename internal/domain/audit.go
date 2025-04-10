package domain

import "time"

type EventType string
type TaskStatus string

const (
	EventStatusChange EventType = "status_change"
	EventAPIRequest   EventType = "api_request"
	EventAPIResponse  EventType = "api_response"
)

const (
	StatusCreated        TaskStatus = "CREATED"
	StatusProcessing     TaskStatus = "PROCESSING"
	StatusFailed         TaskStatus = "FAILED"
	StatusNoAttemptsLeft TaskStatus = "NO_ATTEMPTS_LEFT"
	StatusFinished       TaskStatus = "FINISHED"
)

type Event struct {
	Type EventType
	Data any
	Time time.Time
}

type AuditTask struct {
	ID            int
	AuditLog      []byte
	Status        TaskStatus
	AttemptNumber int
	CreatedAt     time.Time
	UpdatedAt     time.Time
	FinishedAt    time.Time
	NextRetry     time.Time
}

func NewEvent(t EventType, data any) Event {
	return Event{
		Type: t,
		Data: data,
		Time: time.Now().UTC(),
	}
}
