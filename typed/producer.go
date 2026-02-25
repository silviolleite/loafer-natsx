package typed

import (
	"context"
	"fmt"

	"github.com/silviolleite/loafer-natsx/producer"
)

// Producer is a type-safe wrapper around producer.Producer that encodes
// messages of type T before publishing.
type Producer[T any] struct {
	inner *producer.Producer
	codec Codec[T]
}

// NewProducer creates a typed Producer. It delegates to producer.New for
// validation and construction, then wraps the result with the given codec.
func NewProducer[T any](
	pub producer.Publisher,
	subject string,
	codec Codec[T],
	opts ...producer.Option,
) (*Producer[T], error) {
	p, err := producer.New(pub, subject, opts...)
	if err != nil {
		return nil, fmt.Errorf("typed: new producer: %w", err)
	}

	return &Producer[T]{inner: p, codec: codec}, nil
}

// Publish encodes msg using the codec and publishes the resulting bytes.
func (p *Producer[T]) Publish(ctx context.Context, msg T, opts ...producer.PublishOption) error {
	data, err := p.codec.Encode(msg)
	if err != nil {
		return fmt.Errorf("typed: encode: %w", err)
	}

	return p.inner.Publish(ctx, data, opts...)
}
