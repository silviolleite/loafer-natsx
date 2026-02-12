package stream

import (
	"context"

	"github.com/nats-io/nats.go/jetstream"

	loafernastx "github.com/silviolleite/loafer-natsx"
)

const (
	defaultReplicas = 1
)

// Ensure validates and creates or updates a JetStream stream with the given name and configuration options.
func Ensure(ctx context.Context, js jetstream.JetStream, name string, opts ...Option) error {
	if name == "" {
		return loafernastx.ErrMissingName
	}

	cfg := config{
		name:      name,
		retention: jetstream.LimitsPolicy,
		storage:   jetstream.FileStorage,
		replicas:  defaultReplicas,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	if len(cfg.subjects) == 0 {
		return loafernastx.ErrMissingSubjects
	}

	_, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:       cfg.name,
		Subjects:   cfg.subjects,
		Retention:  cfg.retention,
		MaxAge:     cfg.maxAge,
		Storage:    cfg.storage,
		Replicas:   cfg.replicas,
		Duplicates: cfg.duplicateWindow,
	})

	return err
}
