package consumer

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	natstest "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"

	loafernatsx "github.com/silviolleite/loafer-natsx"
	"github.com/silviolleite/loafer-natsx/logger"
	"github.com/silviolleite/loafer-natsx/router"
)

// mockMsg is a hand-written test double for jetstream.Msg used in internal tests.
type mockMsg struct {
	metaErr        error
	ackErr         error
	nakErr         error
	nakWithDelayFn func(time.Duration) error
	headers        nats.Header
	meta           *jetstream.MsgMetadata
	subject        string
	data           []byte
}

func (m *mockMsg) Metadata() (*jetstream.MsgMetadata, error) { return m.meta, m.metaErr }
func (m *mockMsg) Data() []byte                              { return m.data }
func (m *mockMsg) Headers() nats.Header                      { return m.headers }
func (m *mockMsg) Subject() string                           { return m.subject }
func (m *mockMsg) Reply() string                             { return "" }
func (m *mockMsg) Ack() error                                { return m.ackErr }
func (m *mockMsg) DoubleAck(_ context.Context) error         { return nil }
func (m *mockMsg) Nak() error                                { return m.nakErr }
func (m *mockMsg) NakWithDelay(d time.Duration) error {
	if m.nakWithDelayFn != nil {
		return m.nakWithDelayFn(d)
	}
	return nil
}
func (m *mockMsg) InProgress() error             { return nil }
func (m *mockMsg) Term() error                   { return nil }
func (m *mockMsg) TermWithReason(_ string) error { return nil }

func startInternalServer() (*natsserver.Server, string) {
	opts := natstest.DefaultTestOptions
	opts.Port = -1
	s := natstest.RunServer(&opts)
	return s, s.ClientURL()
}

func newTestRoute(t *testing.T, routeType router.Type, subject string, opts ...router.Option) *router.Route {
	t.Helper()
	r, err := router.New(routeType, subject, opts...)
	assert.NoError(t, err)
	return r
}

// TestHandleJetStreamMessage_AckError verifies that an Ack error after a
// successful handler invocation is logged and does not panic.
func TestHandleJetStreamMessage_AckError(t *testing.T) {
	c := &Consumer{logger: logger.NopLogger{}}

	r := newTestRoute(t, router.TypeJetStream, "ack.error",
		router.WithStream("S"), router.WithDurable("D"))

	msg := &mockMsg{
		subject: "ack.error",
		data:    []byte("payload"),
		meta:    &jetstream.MsgMetadata{Stream: "S", NumDelivered: 1},
		ackErr:  errors.New("ack failed"),
	}

	handler := func(_ context.Context, _ []byte) (any, error) {
		return nil, nil
	}

	c.handleJetStreamMessage(context.Background(), r, handler, msg)
}

// TestHandleJetStreamError_PermanentFailure_AckError verifies that an Ack
// error on the permanent-failure path is logged without panicking.
func TestHandleJetStreamError_PermanentFailure_AckError(t *testing.T) {
	c := &Consumer{logger: logger.NopLogger{}}

	r := newTestRoute(t, router.TypeJetStream, "perm.ack.error",
		router.WithStream("S"), router.WithDurable("D"))

	msg := &mockMsg{
		subject: "perm.ack.error",
		ackErr:  errors.New("ack failed"),
	}

	meta := &jetstream.MsgMetadata{Stream: "S", NumDelivered: 1}

	c.handleJetStreamError(r, msg, meta, loafernatsx.ErrPermanentFailure)
}

// TestHandleJetStreamError_SendToDLQ_NilMeta verifies the branch where
// ErrSendToDLQ is returned, DLQ is enabled, but meta is nil — the consumer
// falls back to Ack to avoid infinite retries.
func TestHandleJetStreamError_SendToDLQ_NilMeta(t *testing.T) {
	c := &Consumer{logger: logger.NopLogger{}}

	r := newTestRoute(t, router.TypeJetStream, "dlq.nil.meta",
		router.WithStream("S"), router.WithDurable("D"),
		router.WithEnableDLQ())

	msg := &mockMsg{subject: "dlq.nil.meta"}

	c.handleJetStreamError(r, msg, nil, loafernatsx.ErrSendToDLQ)
}

// TestHandleJetStreamError_SendToDLQ_NilMeta_AckError verifies the same nil-meta
// branch when the fallback Ack also fails — logged without panicking.
func TestHandleJetStreamError_SendToDLQ_NilMeta_AckError(t *testing.T) {
	c := &Consumer{logger: logger.NopLogger{}}

	r := newTestRoute(t, router.TypeJetStream, "dlq.nil.meta.ack.err",
		router.WithStream("S"), router.WithDurable("D"),
		router.WithEnableDLQ())

	msg := &mockMsg{
		subject: "dlq.nil.meta.ack.err",
		ackErr:  errors.New("ack failed"),
	}

	c.handleJetStreamError(r, msg, nil, loafernatsx.ErrSendToDLQ)
}

// TestHandleJetStreamError_SendToDLQ_DLQDisabled_AckError verifies the branch
// where ErrSendToDLQ is returned and DLQ is disabled — falls back to Ack.
// An Ack error is also logged without panicking.
func TestHandleJetStreamError_SendToDLQ_DLQDisabled_AckError(t *testing.T) {
	c := &Consumer{logger: logger.NopLogger{}}

	r := newTestRoute(t, router.TypeJetStream, "dlq.disabled.ack.err",
		router.WithStream("S"), router.WithDurable("D"))

	msg := &mockMsg{
		subject: "dlq.disabled.ack.err",
		ackErr:  errors.New("ack failed"),
	}

	meta := &jetstream.MsgMetadata{Stream: "S", NumDelivered: 1}

	c.handleJetStreamError(r, msg, meta, loafernatsx.ErrSendToDLQ)
}

// TestHandleJetStreamError_NakError verifies that a Nak error on the normal
// retry path is logged without panicking.
func TestHandleJetStreamError_NakError(t *testing.T) {
	c := &Consumer{logger: logger.NopLogger{}}

	r := newTestRoute(t, router.TypeJetStream, "nak.error",
		router.WithStream("S"), router.WithDurable("D"))

	msg := &mockMsg{
		subject: "nak.error",
		nakErr:  errors.New("nak failed"),
	}

	meta := &jetstream.MsgMetadata{Stream: "S", NumDelivered: 1}

	c.handleJetStreamError(r, msg, meta, errors.New("transient"))
}

// TestPublishToDLQ_PublishError_NakFallback verifies that when the DLQ publish
// fails (closed connection), the consumer naks the message and logs the error.
func TestPublishToDLQ_PublishError_NakFallback(t *testing.T) {
	s, url := startInternalServer()
	defer s.Shutdown()

	nc, err := nats.Connect(url)
	assert.NoError(t, err)
	nc.Close() // close so PublishMsg fails

	c := &Consumer{nc: nc, logger: logger.NopLogger{}}

	r := newTestRoute(t, router.TypeJetStream, "dlq.pub.err",
		router.WithStream("S"), router.WithDurable("D"),
		router.WithEnableDLQ())

	msg := &mockMsg{
		subject: "dlq.pub.err",
		data:    []byte("payload"),
	}

	meta := &jetstream.MsgMetadata{Stream: "S", NumDelivered: 1}

	c.publishToDLQ(r, msg, meta, errors.New("transient"))
}

// TestPublishToDLQ_AckError verifies that an Ack error after a successful DLQ
// publish is logged without panicking.
func TestPublishToDLQ_AckError(t *testing.T) {
	s, url := startInternalServer()
	defer s.Shutdown()

	nc, err := nats.Connect(url)
	assert.NoError(t, err)
	defer nc.Close()

	c := &Consumer{nc: nc, logger: logger.NopLogger{}}

	r := newTestRoute(t, router.TypeJetStream, "dlq.ack.err",
		router.WithStream("S"), router.WithDurable("D"),
		router.WithEnableDLQ())

	msg := &mockMsg{
		subject: "dlq.ack.err",
		data:    []byte("payload"),
		ackErr:  errors.New("ack failed after dlq"),
	}

	meta := &jetstream.MsgMetadata{Stream: "S", NumDelivered: 1}

	c.publishToDLQ(r, msg, meta, errors.New("transient"))
}

// TestHandleRequestReplyMessage_PublishError verifies that a PublishMsg error
// on the reply path is logged without panicking.
func TestHandleRequestReplyMessage_PublishError(t *testing.T) {
	s, url := startInternalServer()
	defer s.Shutdown()

	nc, err := nats.Connect(url)
	assert.NoError(t, err)
	nc.Close() // close so PublishMsg fails

	c := &Consumer{nc: nc, logger: logger.NopLogger{}}

	reply := func(_ context.Context, _ any, _ error) ([]byte, nats.Header, error) {
		return []byte("reply"), nil, nil
	}

	r := newTestRoute(t, router.TypeRequestReply, "rr.pub.err",
		router.WithQueueGroup("workers"),
		router.WithReply(reply))

	msg := &nats.Msg{
		Subject: "rr.pub.err",
		Reply:   "_INBOX.test",
		Data:    []byte("data"),
	}

	handler := func(_ context.Context, _ []byte) (any, error) {
		return "result", nil
	}

	c.handleRequestReplyMessage(context.Background(), r, handler, msg)
}

// TestHandleJetStreamError_NakWithDelay_Direct verifies that a NakWithDelayError
// triggers NakWithDelay on the message with the correct duration.
func TestHandleJetStreamError_NakWithDelay_Direct(t *testing.T) {
	c := &Consumer{logger: logger.NopLogger{}}

	r := newTestRoute(t, router.TypeJetStream, "nak.delay",
		router.WithStream("S"), router.WithDurable("D"))

	var capturedDelay time.Duration
	msg := &mockMsg{
		subject:        "nak.delay",
		nakWithDelayFn: func(d time.Duration) error { capturedDelay = d; return nil },
	}

	meta := &jetstream.MsgMetadata{Stream: "S", NumDelivered: 1}

	c.handleJetStreamError(r, msg, meta, loafernatsx.NakWithDelayError{Delay: 5 * time.Second})

	assert.Equal(t, 5*time.Second, capturedDelay)
}

// TestHandleJetStreamError_NakWithDelay_Wrapped verifies that a NakWithDelayError
// wrapped inside another error is still unwrapped and handled correctly.
func TestHandleJetStreamError_NakWithDelay_Wrapped(t *testing.T) {
	c := &Consumer{logger: logger.NopLogger{}}

	r := newTestRoute(t, router.TypeJetStream, "nak.delay.wrapped",
		router.WithStream("S"), router.WithDurable("D"))

	var capturedDelay time.Duration
	msg := &mockMsg{
		subject:        "nak.delay.wrapped",
		nakWithDelayFn: func(d time.Duration) error { capturedDelay = d; return nil },
	}

	meta := &jetstream.MsgMetadata{Stream: "S", NumDelivered: 1}

	wrapped := fmt.Errorf("precondition: %w", loafernatsx.NakWithDelayError{Delay: 30 * time.Second})
	c.handleJetStreamError(r, msg, meta, wrapped)

	assert.Equal(t, 30*time.Second, capturedDelay)
}

// TestHandleJetStreamError_NakWithDelay_NakError verifies that a NakWithDelay
// infrastructure error is logged without panicking.
func TestHandleJetStreamError_NakWithDelay_NakError(t *testing.T) {
	c := &Consumer{logger: logger.NopLogger{}}

	r := newTestRoute(t, router.TypeJetStream, "nak.delay.err",
		router.WithStream("S"), router.WithDurable("D"))

	msg := &mockMsg{
		subject:        "nak.delay.err",
		nakWithDelayFn: func(_ time.Duration) error { return errors.New("nak-with-delay failed") },
	}

	meta := &jetstream.MsgMetadata{Stream: "S", NumDelivered: 1}

	c.handleJetStreamError(r, msg, meta, loafernatsx.NakWithDelayError{Delay: time.Second})
}
