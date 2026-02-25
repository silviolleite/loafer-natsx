package typed

import (
	"context"
	"fmt"

	"github.com/silviolleite/loafer-natsx/producer"
)

// Requester is a type-safe wrapper around producer.Producer that encodes
// request messages of type T and decodes response messages of type R.
type Requester[T any, R any] struct {
	inner    *producer.Producer
	reqCodec Codec[T]
	resCodec Codec[R]
}

// NewRequester creates a typed Requester. It delegates to producer.New for
// validation and construction, then wraps the result with the given codecs.
func NewRequester[T any, R any](
	pub producer.Publisher,
	subject string,
	reqCodec Codec[T],
	resCodec Codec[R],
	opts ...producer.Option,
) (*Requester[T, R], error) {
	p, err := producer.New(pub, subject, opts...)
	if err != nil {
		return nil, fmt.Errorf("typed: new requester: %w", err)
	}

	return &Requester[T, R]{inner: p, reqCodec: reqCodec, resCodec: resCodec}, nil
}

// Request encodes msg using the request codec, sends a request to the
// configured subject, and decodes the response using the response codec.
func (r *Requester[T, R]) Request(ctx context.Context, msg T) (R, error) {
	var zero R

	data, err := r.reqCodec.Encode(msg)
	if err != nil {
		return zero, fmt.Errorf("typed: encode request: %w", err)
	}

	resp, err := r.inner.Request(ctx, data)
	if err != nil {
		return zero, err
	}

	result, err := r.resCodec.Decode(resp)
	if err != nil {
		return zero, fmt.Errorf("typed: decode response: %w", err)
	}

	return result, nil
}
