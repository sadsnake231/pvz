package domain

import "time"

type EventType string

const (
	EventStatusChange EventType = "status_change"
	EventAPIRequest   EventType = "api_request"
	EventAPIResponse  EventType = "api_response"
)

type Event struct {
	Type EventType
	Data any
	Time time.Time
}

func NewEvent(t EventType, data any) Event {
	return Event{
		Type: t,
		Data: data,
		Time: time.Now().UTC(),
	}
}
