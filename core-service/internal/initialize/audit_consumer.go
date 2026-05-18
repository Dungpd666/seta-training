package initialize

import (
	"context"
	"encoding/json"
	"time"

	"github.com/dungpd/seta/core-service/internal/asset"
	"github.com/dungpd/seta/core-service/internal/db"
	"github.com/dungpd/seta/core-service/internal/team"
	"github.com/rs/zerolog/log"
	kafka "github.com/segmentio/kafka-go"
)

const groupAudit = "core-audit"

func StartAuditConsumer(ctx context.Context, brokers []string, q *db.Queries) {
	for _, topic := range []string{team.TopicTeamActivity, asset.TopicAssetChanges} {
		go consumeAuditTopic(ctx, brokers, topic, q)
	}
}

func consumeAuditTopic(ctx context.Context, brokers []string, topic string, q *db.Queries) {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       topic,
		GroupID:     groupAudit,
		StartOffset: kafka.FirstOffset,
	})

	defer r.Close()
	for {
		msg, err := r.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Error().Err(err).Str("topic", topic).Msg("audit consumer read error")
			time.Sleep(500 * time.Millisecond)
			continue
		}

		var envelope struct {
			Event string `json:"event"`
		}
		if err := json.Unmarshal(msg.Value, &envelope); err != nil {
			log.Error().Err(err).Str("topic", topic).Msg("invalid audit event payload")
			continue
		}

		if err := q.InsertAuditLog(ctx, db.InsertAuditLogParams{
			EventType: envelope.Event,
			Payload:   msg.Value,
		}); err != nil {
			log.Error().Err(err).Str("event", envelope.Event).Msg("failed to insert audit log")
		}
	}
}
