package producer_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"

	loafernatsx "github.com/silviolleite/loafer-natsx"
	"github.com/silviolleite/loafer-natsx/logger"
	"github.com/silviolleite/loafer-natsx/producer"
)

type mockPublisher struct {
	err    error
	msg    *nats.Msg
	called bool
}

func (m *mockPublisher) Publish(ctx context.Context, msg *nats.Msg, opts producer.PublishOptions) error {
	m.called = true
	m.msg = msg
	return m.err
}

type mockRequester struct {
	mockPublisher
	reqErr   error
	response []byte
}

func (m *mockRequester) Request(ctx context.Context, subject string, data []byte) ([]byte, error) {
	if m.reqErr != nil {
		return nil, m.reqErr
	}
	return m.response, nil
}

func TestNew_MissingSubject(t *testing.T) {
	p, err := producer.New(&mockPublisher{}, "")
	assert.Nil(t, p)
	assert.ErrorIs(t, err, loafernatsx.ErrMissingSubject)
}

func TestNew_NilPublisher(t *testing.T) {
	p, err := producer.New(nil, "subject")
	assert.Nil(t, p)
	assert.ErrorIs(t, err, loafernatsx.ErrUnsupportedType)
}

func TestPublish_Success(t *testing.T) {
	mp := &mockPublisher{}

	p, err := producer.New(mp, "test.subject", producer.WithLogger(logger.NopLogger{}))
	assert.NoError(t, err)

	err = p.Publish(context.Background(), []byte("data"))
	assert.NoError(t, err)
	assert.True(t, mp.called)
	assert.Equal(t, "test.subject", mp.msg.Subject)
	assert.Equal(t, []byte("data"), mp.msg.Data)
}

func TestPublish_WithHeaders(t *testing.T) {
	mp := &mockPublisher{}

	p, _ := producer.New(mp, "test.subject")

	h := nats.Header{}
	h.Set("k", "v")

	err := p.Publish(
		context.Background(),
		[]byte("data"),
		producer.PublishWithHeaders(h),
	)

	assert.NoError(t, err)
	assert.Equal(t, "v", mp.msg.Header.Get("k"))
}

func TestPublish_Error(t *testing.T) {
	mp := &mockPublisher{err: errors.New("fail")}

	p, _ := producer.New(mp, "test.subject")

	err := p.Publish(context.Background(), []byte("data"))
	assert.Error(t, err)
}

func TestRequest_NotSupported(t *testing.T) {
	mp := &mockPublisher{}

	p, _ := producer.New(mp, "test.subject")

	resp, err := p.Request(context.Background(), []byte("data"))
	assert.Nil(t, resp)
	assert.ErrorIs(t, err, loafernatsx.ErrRequestNotSupported)
}

func TestRequest_Success(t *testing.T) {
	mr := &mockRequester{
		response: []byte("ok"),
	}

	p, _ := producer.New(mr, "test.subject")

	resp, err := p.Request(context.Background(), []byte("data"))
	assert.NoError(t, err)
	assert.Equal(t, []byte("ok"), resp)
}

func TestRequest_Error(t *testing.T) {
	mr := &mockRequester{
		reqErr: errors.New("fail"),
	}

	p, _ := producer.New(mr, "test.subject")

	resp, err := p.Request(context.Background(), []byte("data"))
	assert.Nil(t, resp)
	assert.Error(t, err)
}
