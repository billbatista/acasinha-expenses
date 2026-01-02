package eventlogger

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID        uuid.UUID         `json:"id,omitempty"`
	Type      string            `json:"event_type,omitempty"`
	Data      any               `json:"event_data,omitempty"`
	Metadata  map[string]string `json:"event_metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

type EventOption func(*Event)

func WithType(eventType string) EventOption {
	return func(e *Event) {
		e.Type = eventType
	}
}

func WithData(data any) EventOption {
	return func(e *Event) {
		e.Data = data
	}
}

func WithMetadata(metadata map[string]string) EventOption {
	return func(e *Event) {
		e.Metadata = metadata
	}
}

func NewEvent(opts ...EventOption) Event {
	e := Event{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		Metadata:  make(map[string]string),
	}
	for _, opt := range opts {
		opt(&e)
	}
	return e
}

type EventLogger interface {
	Save(ctx context.Context, e Event) error
	GetByType(ctx context.Context, eventType string) ([]Event, error)
}
