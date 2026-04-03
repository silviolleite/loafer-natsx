package producer_test

import (
	"context"
	"errors"
	"testing"
	"time"

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

func (m *mockPublisher) Publish(_ context.Context, msg *nats.Msg, _ producer.PublishOptions) (*producer.PublishResult, error) {
	m.called = true
	m.msg = msg
	if m.err != nil {
		return nil, m.err
	}
	return &producer.PublishResult{}, nil
}

type mockRequester struct {
	reqErr      error
	capturedCtx context.Context
	response    *producer.Response
	mockPublisher
	blockUntilCtxDone bool
}

func (m *mockRequester) Request(ctx context.Context, _ string, _ []byte) (*producer.Response, error) {
	m.capturedCtx = ctx
	if m.blockUntilCtxDone {
		<-ctx.Done()
		return nil, ctx.Err()
	}
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

	result, err := p.Publish(context.Background(), []byte("data"))
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, mp.called)
	assert.Equal(t, "test.subject", mp.msg.Subject)
	assert.Equal(t, []byte("data"), mp.msg.Data)
}

func TestPublish_WithHeaders(t *testing.T) {
	mp := &mockPublisher{}

	p, _ := producer.New(mp, "test.subject")

	h := nats.Header{}
	h.Set("k", "v")

	_, err := p.Publish(
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

	result, err := p.Publish(context.Background(), []byte("data"))
	assert.Nil(t, result)
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
		response: &producer.Response{Data: []byte("ok")},
	}

	p, _ := producer.New(mr, "test.subject")

	resp, err := p.Request(context.Background(), []byte("data"))
	assert.NoError(t, err)
	assert.Equal(t, []byte("ok"), resp.Data)
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

func TestRequest_WithRequestTimeout(t *testing.T) {
	t.Run("applies default 10s deadline when no option is set", func(t *testing.T) {
		mr := &mockRequester{
			response: &producer.Response{Data: []byte("ok")},
		}

		p, err := producer.New(mr, "test.subject")
		assert.NoError(t, err)

		_, err = p.Request(context.Background(), []byte("data"))
		assert.NoError(t, err)

		deadline, ok := mr.capturedCtx.Deadline()
		assert.True(t, ok)
		assert.WithinDuration(t, time.Now().Add(10*time.Second), deadline, 1*time.Second)
	})

	t.Run("applies custom deadline to context", func(t *testing.T) {
		mr := &mockRequester{
			response: &producer.Response{Data: []byte("ok")},
		}

		p, err := producer.New(mr, "test.subject", producer.WithRequestTimeout(5*time.Second))
		assert.NoError(t, err)

		_, err = p.Request(context.Background(), []byte("data"))
		assert.NoError(t, err)

		deadline, ok := mr.capturedCtx.Deadline()
		assert.True(t, ok)
		assert.WithinDuration(t, time.Now().Add(5*time.Second), deadline, 1*time.Second)
	})

	t.Run("respects shorter caller deadline", func(t *testing.T) {
		mr := &mockRequester{
			response: &producer.Response{Data: []byte("ok")},
		}

		p, err := producer.New(mr, "test.subject", producer.WithRequestTimeout(10*time.Second))
		assert.NoError(t, err)

		callerTimeout := 2 * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), callerTimeout)
		defer cancel()

		_, err = p.Request(ctx, []byte("data"))
		assert.NoError(t, err)

		deadline, ok := mr.capturedCtx.Deadline()
		assert.True(t, ok)
		assert.WithinDuration(t, time.Now().Add(callerTimeout), deadline, 1*time.Second)
	})

	t.Run("no deadline when timeout explicitly disabled", func(t *testing.T) {
		mr := &mockRequester{
			response: &producer.Response{Data: []byte("ok")},
		}

		p, err := producer.New(mr, "test.subject", producer.WithoutRequestTimeout())
		assert.NoError(t, err)

		_, err = p.Request(context.Background(), []byte("data"))
		assert.NoError(t, err)

		_, ok := mr.capturedCtx.Deadline()
		assert.False(t, ok)
	})

	t.Run("returns ErrRequestTimeout when deadline exceeded", func(t *testing.T) {
		mr := &mockRequester{
			blockUntilCtxDone: true,
		}

		p, err := producer.New(mr, "test.subject", producer.WithRequestTimeout(10*time.Millisecond))
		assert.NoError(t, err)

		resp, err := p.Request(context.Background(), []byte("data"))
		assert.Nil(t, resp)
		assert.ErrorIs(t, err, loafernatsx.ErrRequestTimeout)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("does not wrap non-timeout errors with ErrRequestTimeout", func(t *testing.T) {
		mr := &mockRequester{
			reqErr: errors.New("connection closed"),
		}

		p, err := producer.New(mr, "test.subject", producer.WithoutRequestTimeout())
		assert.NoError(t, err)

		resp, err := p.Request(context.Background(), []byte("data"))
		assert.Nil(t, resp)
		assert.Error(t, err)
		assert.False(t, errors.Is(err, loafernatsx.ErrRequestTimeout))
	})
}
