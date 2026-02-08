package stream

import (
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

type config struct {
	name            string
	subjects        []string
	retention       jetstream.RetentionPolicy
	maxAge          time.Duration
	storage         jetstream.StorageType
	replicas        int
	duplicateWindow time.Duration
}
