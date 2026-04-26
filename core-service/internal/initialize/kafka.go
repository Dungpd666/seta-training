package initialize

import (
	"context"
	"encoding/json"

	"github.com/dungpd/seta/core-service/internal/team"
	"github.com/rs/zerolog/log"
	kafka "github.com/segmentio/kafka-go"
)

func StartUserEventConsumer(ctx context.Context, brokers []string, repo team.ProjectionRepository) {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       "user.events",
		GroupID:     "core-user-projection",
		StartOffset: kafka.FirstOffset,
	})

	go func() {
		defer r.Close()
		for {
			msg, err := r.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Error().Err(err).Msg("kafka read error")
				continue
			}
			handleUserEvent(ctx, msg.Value, repo)
		}
	}()
}

func handleUserEvent(ctx context.Context, data []byte, repo team.ProjectionRepository) {
	var event team.UserEvent
	if err := json.Unmarshal(data, &event); err != nil {
		log.Error().Err(err).Msg("invalid user event payload")
		return
	}
	switch event.Type {
	case team.EventUserCreated, team.EventUserUpdated:
		if err := repo.Upsert(ctx, team.UserProjection{
			UserID:   event.UserID,
			Username: event.UserName,
			Email:    event.Email,
			Role:     event.Role,
		}); err != nil {
			log.Error().Err(err).Str("user_id", event.UserID).Msg("upsert user projection failed")
		}
	case team.EventUserDeleted:
		if err := repo.SoftDelete(ctx, event.UserID); err != nil {
			log.Error().Err(err).Str("user_id", event.UserID).Msg("soft delete user projection failed")
		}
	}
}
