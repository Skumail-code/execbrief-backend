package db

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDB struct {
	Client   *mongo.Client
	Database *mongo.Database
}

func Connect(uri, dbName string) (*MongoDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOpts := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db := &MongoDB{
		Client:   client,
		Database: client.Database(dbName),
	}

	if err := db.createIndexes(ctx); err != nil {
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	return db, nil
}

func (m *MongoDB) createIndexes(ctx context.Context) error {
	users := m.Database.Collection("users")
	_, err := users.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetName("users_email").SetUnique(true),
		},
	})
	if err != nil {
		return fmt.Errorf("users indexes: %w", err)
	}

	sessions := m.Database.Collection("sessions")
	_, err = sessions.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "token_hash", Value: 1}},
			Options: options.Index().SetName("sessions_token_hash").SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "expires_at", Value: 1}},
			Options: options.Index().SetName("sessions_expires_at").SetExpireAfterSeconds(0),
		},
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}},
			Options: options.Index().SetName("sessions_user_id"),
		},
	})
	if err != nil {
		return fmt.Errorf("sessions indexes: %w", err)
	}

	meetings := m.Database.Collection("meetings")
	_, err = meetings.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "created_at", Value: 1}},
			Options: options.Index().SetName("meetings_created_at"),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetName("meetings_status"),
		},
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("meetings_user_created"),
		},
	})
	if err != nil {
		return fmt.Errorf("meetings indexes: %w", err)
	}

	analysis := m.Database.Collection("meeting_analysis")
	_, err = analysis.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "meeting_id", Value: 1}},
			Options: options.Index().SetName("analysis_meeting_id").SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}},
			Options: options.Index().SetName("analysis_user_id"),
		},
	})
	if err != nil {
		return fmt.Errorf("meeting_analysis indexes: %w", err)
	}

	actionItems := m.Database.Collection("action_items")
	_, err = actionItems.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "meeting_id", Value: 1}},
			Options: options.Index().SetName("action_items_meeting_id"),
		},
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}},
			Options: options.Index().SetName("action_items_user_id"),
		},
	})
	if err != nil {
		return fmt.Errorf("action_items indexes: %w", err)
	}

	decisions := m.Database.Collection("decisions")
	_, err = decisions.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "meeting_id", Value: 1}},
			Options: options.Index().SetName("decisions_meeting_id"),
		},
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}},
			Options: options.Index().SetName("decisions_user_id"),
		},
	})
	if err != nil {
		return fmt.Errorf("decisions indexes: %w", err)
	}

	return nil
}

func (m *MongoDB) Disconnect(ctx context.Context) error {
	return m.Client.Disconnect(ctx)
}
