package typed_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loafernatsx "github.com/silviolleite/loafer-natsx"
	"github.com/silviolleite/loafer-natsx/typed"
)

type processedOrder struct {
	Status  string `json:"status"`
	OrderID string `json:"order_id"`
}

func TestNewRequester_EmptySubject(t *testing.T) {
	r, err := typed.NewRequester[order, processedOrder](
		&mockRequester{}, "", typed.JSONCodec[order]{}, typed.JSONCodec[processedOrder]{},
	)
	assert.Nil(t, r)
	assert.ErrorIs(t, err, loafernatsx.ErrMissingSubject)
}

func TestNewRequester_NilPublisher(t *testing.T) {
	r, err := typed.NewRequester[order, processedOrder](
		nil, "orders.process", typed.JSONCodec[order]{}, typed.JSONCodec[processedOrder]{},
	)
	assert.Nil(t, r)
	assert.ErrorIs(t, err, loafernatsx.ErrUnsupportedType)
}

func TestNewRequester_Success(t *testing.T) {
	r, err := typed.NewRequester[order, processedOrder](
		&mockRequester{}, "orders.process", typed.JSONCodec[order]{}, typed.JSONCodec[processedOrder]{},
	)
	require.NoError(t, err)
	assert.NotNil(t, r)
}

func TestRequester_Request_Success(t *testing.T) {
	mr := &mockRequester{reqResp: []byte(`{"status":"processed","order_id":"1"}`)}
	r, err := typed.NewRequester[order, processedOrder](
		mr, "orders.process", typed.JSONCodec[order]{}, typed.JSONCodec[processedOrder]{},
	)
	require.NoError(t, err)

	result, err := r.Request(context.Background(), order{ID: "1", Amount: 50})
	require.NoError(t, err)
	assert.Equal(t, "processed", result.Status)
	assert.Equal(t, "1", result.OrderID)
}

func TestRequester_Request_EncodeError(t *testing.T) {
	mr := &mockRequester{}
	r, err := typed.NewRequester[order, processedOrder](
		mr, "orders.process", failCodec[order]{}, typed.JSONCodec[processedOrder]{},
	)
	require.NoError(t, err)

	result, err := r.Request(context.Background(), order{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "encode request")
	assert.Zero(t, result)
}

func TestRequester_Request_DecodeError(t *testing.T) {
	mr := &mockRequester{reqResp: []byte(`not json`)}
	r, err := typed.NewRequester[order, processedOrder](
		mr, "orders.process", typed.JSONCodec[order]{}, typed.JSONCodec[processedOrder]{},
	)
	require.NoError(t, err)

	result, err := r.Request(context.Background(), order{ID: "1", Amount: 50})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode response")
	assert.Zero(t, result)
}

func TestRequester_Request_NotSupported(t *testing.T) {
	mp := &mockPublisher{}
	r, err := typed.NewRequester[order, processedOrder](
		mp, "orders.process", typed.JSONCodec[order]{}, typed.JSONCodec[processedOrder]{},
	)
	require.NoError(t, err)

	result, err := r.Request(context.Background(), order{ID: "1", Amount: 50})
	assert.ErrorIs(t, err, loafernatsx.ErrRequestNotSupported)
	assert.Zero(t, result)
}

func TestRequester_Request_RequesterError(t *testing.T) {
	mr := &mockRequester{reqErr: errors.New("timeout")}
	r, err := typed.NewRequester[order, processedOrder](
		mr, "orders.process", typed.JSONCodec[order]{}, typed.JSONCodec[processedOrder]{},
	)
	require.NoError(t, err)

	result, err := r.Request(context.Background(), order{ID: "1", Amount: 50})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
	assert.Zero(t, result)
}

func TestRequester_Request_DecodeResponseCodecError(t *testing.T) {
	mr := &mockRequester{reqResp: []byte(`{"status":"ok"}`)}
	r, err := typed.NewRequester[order, processedOrder](
		mr, "orders.process", typed.JSONCodec[order]{}, failCodec[processedOrder]{},
	)
	require.NoError(t, err)

	result, err := r.Request(context.Background(), order{ID: "1", Amount: 50})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode response")
	assert.Zero(t, result)
}
