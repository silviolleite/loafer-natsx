package middleware

import (
	"context"

	"github.com/silviolleite/loafer-natsx/consumer"
)

// unknownSubject is the subject label value used when the incoming context
// carries no message Metadata, so metrics and spans always have a stable,
// non-empty subject dimension.
const unknownSubject = "unknown"

// subjectFromContext returns the message subject attached to the context by the
// consumer, falling back to unknownSubject when no Metadata is present or the
// subject is empty.
func subjectFromContext(ctx context.Context) string {
	if md, ok := consumer.MetadataFromContext(ctx); ok && md != nil && md.Subject != "" {
		return md.Subject
	}

	return unknownSubject
}
