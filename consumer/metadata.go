package consumer

import (
	"context"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// metadataKey is an unexported context key type to avoid collisions with other packages.
type metadataKey struct{}

// Metadata carries message-level information propagated through context.Context
// to the handler, allowing access to subject, headers, reply subject, and
// JetStream-specific metadata without coupling the handler signature to any
// specific NATS message type.
//
// The Metadata is subscription-mode aware:
//   - For Core NATS (PubSub, Queue, RequestReply) only Subject, Headers
//     and Reply are populated.
//   - For JetStream, all fields including Stream, Sequence, NumDelivered
//     and Timestamp are populated when available.
type Metadata struct {
	Timestamp    time.Time
	Headers      nats.Header
	Subject      string
	Reply        string
	Stream       string
	Consumer     string
	Sequence     uint64
	NumDelivered uint64
}

// WithMetadata returns a new context carrying the given Metadata value.
// The pointer is stored as-is; callers should not mutate it after the call.
func WithMetadata(ctx context.Context, md *Metadata) context.Context {
	return context.WithValue(ctx, metadataKey{}, md)
}

// MetadataFromContext extracts the Metadata previously attached to the context.
// The returned boolean is false when no Metadata is present.
func MetadataFromContext(ctx context.Context) (*Metadata, bool) {
	if ctx == nil {
		return nil, false
	}
	md, ok := ctx.Value(metadataKey{}).(*Metadata)
	return md, ok
}

// metadataFromCoreMsg builds a Metadata from a Core NATS message.
func metadataFromCoreMsg(msg *nats.Msg) *Metadata {
	return &Metadata{
		Subject: msg.Subject,
		Reply:   msg.Reply,
		Headers: msg.Header,
	}
}

// metadataFromJetStreamMsg builds a Metadata from a JetStream message.
// The JetStream-specific fields fall back to zero values when metadata is
// unavailable (e.g. on non-flow messages).
func metadataFromJetStreamMsg(msg jetstream.Msg) *Metadata {
	md := &Metadata{
		Subject: msg.Subject(),
		Reply:   msg.Reply(),
		Headers: msg.Headers(),
	}

	if meta, err := msg.Metadata(); err == nil && meta != nil {
		md.Stream = meta.Stream
		md.Consumer = meta.Consumer
		md.Sequence = meta.Sequence.Stream
		md.NumDelivered = meta.NumDelivered
		md.Timestamp = meta.Timestamp
	}

	return md
}
