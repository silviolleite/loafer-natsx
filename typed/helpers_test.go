package typed_test

import (
	"context"
	"errors"

	"github.com/nats-io/nats.go"

	"github.com/silviolleite/loafer-natsx/producer"
)

// mockPublisher implements producer.Publisher for testing.
type mockPublisher struct {
	err    error
	msg    *nats.Msg
	called bool
}

func (m *mockPublisher) Publish(_ context.Context, msg *nats.Msg, _ producer.PublishOptions) error {
	m.called = true
	m.msg = msg
	return m.err
}

// mockRequester implements both producer.Publisher and producer.Requester.
type mockRequester struct {
	reqErr error
	mockPublisher
	reqResp []byte
}

func (m *mockRequester) Request(_ context.Context, subject string, data []byte) ([]byte, error) {
	return m.reqResp, m.reqErr
}

// order is a test message type.
type order struct {
	ID     string  `json:"id"`
	Amount float64 `json:"amount"`
}

// failCodec always fails on Encode and Decode.
type failCodec[T any] struct{}

func (failCodec[T]) Encode(_ T) ([]byte, error) { return nil, errors.New("encode boom") }
func (failCodec[T]) Decode(_ []byte) (T, error) {
	var zero T
	return zero, errors.New("decode boom")
}
