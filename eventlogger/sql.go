package eventlogger

import (
	"context"
	"database/sql"
	"encoding/json"
)

type sqlEventLogger struct {
	db *sql.DB
}

func NewSqlEventLogger(db *sql.DB) *sqlEventLogger {
	return &sqlEventLogger{
		db: db,
	}
}

func (el *sqlEventLogger) Save(ctx context.Context, e Event) error {
	jsonData, err := json.Marshal(e.Data)
	if err != nil {
		return err
	}
	jsonMetadata, err := json.Marshal(e.Metadata)
	if err != nil {
		return err
	}
	statement := `INSERT INTO events (id, event_type, event_data, event_metadata, created_at) VALUES ($1, $2, $3, $4, $5)`
	_, err = el.db.ExecContext(ctx, statement, e.ID, e.Type, jsonData, jsonMetadata, e.CreatedAt)
	if err != nil {
		return err
	}

	return nil
}

func (el *sqlEventLogger) GetByType(ctx context.Context, eventType string) ([]Event, error) {
	query := `SELECT id, event_type, event_data, event_metadata, created_at FROM events WHERE event_type = $1`
	result, err := el.db.QueryContext(ctx, query, eventType)
	if err != nil {
		return nil, err
	}
	defer result.Close()

	events := make([]Event, 0)
	for result.Next() {
		var event Event
		var jsonMetadata []byte
		if err := result.Scan(&event.ID, &event.Type, &event.Data, &jsonMetadata, &event.CreatedAt); err != nil {
			return events, err
		}
		var metadata map[string]string
		err = json.Unmarshal(jsonMetadata, &metadata)
		event.Metadata = metadata

		events = append(events, event)
	}

	if err := result.Err(); err != nil {
		return events, err
	}

	return events, nil
}
