package typed_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/silviolleite/loafer-natsx/typed"
)

type replyResult struct {
	Status string `json:"status"`
}

func TestWrapReply_Success(t *testing.T) {
	fn := typed.WrapReply(func(_ context.Context, result replyResult, handlerErr error) ([]byte, nats.Header, error) {
		assert.Nil(t, handlerErr)
		assert.Equal(t, "ok", result.Status)

		h := nats.Header{}
		h.Set("X-Status", "success")

		b, err := json.Marshal(result)
		return b, h, err
	})

	data, headers, err := fn(context.Background(), replyResult{Status: "ok"}, nil)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"status":"ok"`)
	assert.Equal(t, "success", headers.Get("X-Status"))
}

func TestWrapReply_HandlerError(t *testing.T) {
	handlerErr := errors.New("handler failed")

	fn := typed.WrapReply(func(_ context.Context, result replyResult, hErr error) ([]byte, nats.Header, error) {
		assert.Equal(t, handlerErr, hErr)
		assert.Zero(t, result)

		h := nats.Header{}
		h.Set("X-Status", "error")
		return []byte(hErr.Error()), h, nil
	})

	data, headers, err := fn(context.Background(), nil, handlerErr)
	require.NoError(t, err)
	assert.Equal(t, "handler failed", string(data))
	assert.Equal(t, "error", headers.Get("X-Status"))
}

func TestWrapReply_TypeMismatch(t *testing.T) {
	fn := typed.WrapReply(func(_ context.Context, _ replyResult, _ error) ([]byte, nats.Header, error) {
		t.Fatal("should not be called")
		return nil, nil, nil
	})

	data, headers, err := fn(context.Background(), "wrong type", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "typed: reply: expected")
	assert.Nil(t, data)
	assert.Nil(t, headers)
}

func TestWrapReply_NilResult(t *testing.T) {
	fn := typed.WrapReply(func(_ context.Context, result *replyResult, handlerErr error) ([]byte, nats.Header, error) {
		assert.Nil(t, result)
		assert.Nil(t, handlerErr)
		return []byte("null"), nil, nil
	})

	data, _, err := fn(context.Background(), (*replyResult)(nil), nil)
	require.NoError(t, err)
	assert.Equal(t, "null", string(data))
}

func TestWrapReply_InnerFuncError(t *testing.T) {
	fn := typed.WrapReply(func(_ context.Context, _ replyResult, _ error) ([]byte, nats.Header, error) {
		return nil, nil, errors.New("marshal failed")
	})

	data, headers, err := fn(context.Background(), replyResult{Status: "ok"}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "marshal failed")
	assert.Nil(t, data)
	assert.Nil(t, headers)
}
