package typed

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"

	"github.com/silviolleite/loafer-natsx/producer"
	"github.com/silviolleite/loafer-natsx/reply"
)

// ErrorDecoder is a function that converts a reply error into a domain-specific
// error. It receives the semantic status, the error code from X-Error-Code
// header, and the raw response body. The returned error is what Request will
// return to the caller, enabling patterns like errors.Is(err, myDomainError).
type ErrorDecoder func(status reply.Status, code string, body []byte) error

// RequesterOption configures optional behavior for a Requester.
type RequesterOption func(*requesterConfig)

type requesterConfig struct {
	errDecoder ErrorDecoder
}

// WithErrorDecoder sets a custom error decoder that translates reply errors
// into domain-specific errors. When set, the decoder is called instead of
// returning a generic *ReplyError. This allows callers to use errors.Is and
// errors.As with their own error types.
func WithErrorDecoder(dec ErrorDecoder) RequesterOption {
	return func(c *requesterConfig) {
		c.errDecoder = dec
	}
}

// Requester is a type-safe wrapper around producer.Producer that encodes
// request messages of type T and decodes response messages of type R.
type Requester[T any, R any] struct {
	inner      *producer.Producer
	reqCodec   Codec[T]
	resCodec   Codec[R]
	errDecoder ErrorDecoder
}

// NewRequester creates a typed Requester. It delegates to producer.New for
// validation and construction, then wraps the result with the given codecs.
// Optional RequesterOption values can be passed after the producer options.
func NewRequester[T any, R any](
	pub producer.Publisher,
	subject string,
	reqCodec Codec[T],
	resCodec Codec[R],
	opts ...any,
) (*Requester[T, R], error) {
	var (
		prodOpts []producer.Option
		cfg      requesterConfig
	)

	for _, o := range opts {
		switch v := o.(type) {
		case producer.Option:
			prodOpts = append(prodOpts, v)
		case RequesterOption:
			v(&cfg)
		}
	}

	p, err := producer.New(pub, subject, prodOpts...)
	if err != nil {
		return nil, fmt.Errorf("typed: new requester: %w", err)
	}

	return &Requester[T, R]{
		inner:      p,
		reqCodec:   reqCodec,
		resCodec:   resCodec,
		errDecoder: cfg.errDecoder,
	}, nil
}

// Request encodes msg using the request codec, sends a request to the
// configured subject, and decodes the response using the response codec.
// When an error status is detected and an ErrorDecoder is configured, the
// decoder is called to produce the returned error. Otherwise a *ReplyError
// is returned.
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

	if isErrorStatus(resp.Header) {
		return zero, r.decodeError(resp)
	}

	result, err := r.resCodec.Decode(resp.Data)
	if err != nil {
		return zero, fmt.Errorf("typed: decode response: %w", err)
	}

	return result, nil
}

func (r *Requester[T, R]) decodeError(resp *producer.Response) error {
	re := newReplyError(resp)
	if r.errDecoder != nil {
		return r.errDecoder(re.Status, re.Code, re.Body)
	}
	return re
}

func isErrorStatus(h nats.Header) bool {
	if h == nil {
		return false
	}
	s := reply.Status(h.Get(reply.HeaderStatus))
	return s == reply.StatusError || s == reply.StatusFail
}

func newReplyError(resp *producer.Response) *ReplyError {
	return &ReplyError{
		Status:  reply.Status(resp.Header.Get(reply.HeaderStatus)),
		Code:    resp.Header.Get(reply.HeaderErrorCode),
		Message: extractErrorMessage(resp.Data),
		Body:    resp.Data,
	}
}
