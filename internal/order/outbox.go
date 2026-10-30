package order

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/segmentio/kafka-go"
	"time"
)

type OutboxDispatcher struct {
	DB     *sql.DB
	Writer *kafka.Writer
}

func (d *OutboxDispatcher) Run(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.publishBatch(ctx)
		}
	}
}

func (d *OutboxDispatcher) publishBatch(ctx context.Context) {
	tx, err := d.DB.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT id,event_type,payload FROM outbox_events WHERE published_at IS NULL ORDER BY created_at LIMIT 50 FOR UPDATE SKIP LOCKED`)
	if err != nil {
		return
	}
	type event struct {
		id      int64
		kind    string
		payload []byte
	}
	batch := make([]event, 0)
	for rows.Next() {
		var item event
		if rows.Scan(&item.id, &item.kind, &item.payload) == nil {
			batch = append(batch, item)
		}
	}
	rows.Close()
	for _, item := range batch {
		var payload map[string]interface{}
		if json.Unmarshal(item.payload, &payload) != nil {
			continue
		}
		body, _ := json.Marshal(payload)
		if err := d.Writer.WriteMessages(ctx, kafka.Message{Key: []byte(item.kind), Value: body}); err != nil {
			return
		}
		if _, err := tx.ExecContext(ctx, `UPDATE outbox_events SET published_at=now() WHERE id=$1`, item.id); err != nil {
			return
		}
	}
	if len(batch) > 0 {
		_ = tx.Commit()
	}
}
