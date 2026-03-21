package typed_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loafernatsx "github.com/silviolleite/loafer-natsx"
	"github.com/silviolleite/loafer-natsx/producer"
	"github.com/silviolleite/loafer-natsx/reply"
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
	mr := &mockRequester{reqResp: &producer.Response{Data: []byte(`{"status":"processed","order_id":"1"}`)}}
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
	mr := &mockRequester{reqResp: &producer.Response{Data: []byte(`not json`)}}
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
	mr := &mockRequester{reqResp: &producer.Response{Data: []byte(`{"status":"ok"}`)}}
	r, err := typed.NewRequester[order, processedOrder](
		mr, "orders.process", typed.JSONCodec[order]{}, failCodec[processedOrder]{},
	)
	require.NoError(t, err)

	result, err := r.Request(context.Background(), order{ID: "1", Amount: 50})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode response")
	assert.Zero(t, result)
}

func TestRequester_Request_ReplyError(t *testing.T) {
	tests := []struct {
		resp        *producer.Response
		wantResult  processedOrder
		name        string
		wantStatus  reply.Status
		wantCode    string
		wantMessage string
		wantBody    []byte
		wantErr     bool
	}{
		{
			name: "X-Status error returns ReplyError with message from JSON body",
			resp: &producer.Response{
				Data:   []byte(`{"error":"something went wrong"}`),
				Header: nats.Header{reply.HeaderStatus: []string{string(reply.StatusError)}},
			},
			wantErr:     true,
			wantStatus:  reply.StatusError,
			wantMessage: "something went wrong",
			wantBody:    []byte(`{"error":"something went wrong"}`),
		},
		{
			name: "X-Status error with X-Error-Code returns ReplyError with Code and Message",
			resp: &producer.Response{
				Data: []byte(`{"error":"invalid input"}`),
				Header: nats.Header{
					reply.HeaderStatus:    []string{string(reply.StatusError)},
					reply.HeaderErrorCode: []string{"VALIDATION"},
				},
			},
			wantErr:     true,
			wantStatus:  reply.StatusError,
			wantCode:    "VALIDATION",
			wantMessage: "invalid input",
			wantBody:    []byte(`{"error":"invalid input"}`),
		},
		{
			name: "X-Status fail returns ReplyError with message",
			resp: &producer.Response{
				Data:   []byte(`{"error":"operation failed"}`),
				Header: nats.Header{reply.HeaderStatus: []string{string(reply.StatusFail)}},
			},
			wantErr:     true,
			wantStatus:  reply.StatusFail,
			wantMessage: "operation failed",
			wantBody:    []byte(`{"error":"operation failed"}`),
		},
		{
			name: "X-Status error with plain text body extracts message",
			resp: &producer.Response{
				Data:   []byte("plain error text"),
				Header: nats.Header{reply.HeaderStatus: []string{string(reply.StatusError)}},
			},
			wantErr:     true,
			wantStatus:  reply.StatusError,
			wantMessage: "plain error text",
			wantBody:    []byte("plain error text"),
		},
		{
			name: "X-Status error with empty body has no message",
			resp: &producer.Response{
				Data:   nil,
				Header: nats.Header{reply.HeaderStatus: []string{string(reply.StatusError)}},
			},
			wantErr:    true,
			wantStatus: reply.StatusError,
		},
		{
			name: "X-Status success decodes normally",
			resp: &producer.Response{
				Data:   []byte(`{"status":"done","order_id":"42"}`),
				Header: nats.Header{reply.HeaderStatus: []string{string(reply.StatusSuccess)}},
			},
			wantResult: processedOrder{Status: "done", OrderID: "42"},
		},
		{
			name: "X-Status partial decodes normally",
			resp: &producer.Response{
				Data:   []byte(`{"status":"partial","order_id":"99"}`),
				Header: nats.Header{reply.HeaderStatus: []string{string(reply.StatusPartial)}},
			},
			wantResult: processedOrder{Status: "partial", OrderID: "99"},
		},
		{
			name: "nil header decodes normally",
			resp: &producer.Response{
				Data:   []byte(`{"status":"ok","order_id":"7"}`),
				Header: nil,
			},
			wantResult: processedOrder{Status: "ok", OrderID: "7"},
		},
		{
			name: "empty header decodes normally",
			resp: &producer.Response{
				Data:   []byte(`{"status":"ok","order_id":"8"}`),
				Header: nats.Header{},
			},
			wantResult: processedOrder{Status: "ok", OrderID: "8"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mr := &mockRequester{reqResp: tt.resp}
			r, err := typed.NewRequester[order, processedOrder](
				mr, "orders.process", typed.JSONCodec[order]{}, typed.JSONCodec[processedOrder]{},
			)
			require.NoError(t, err)

			result, err := r.Request(context.Background(), order{ID: "1", Amount: 10})

			if tt.wantErr {
				require.Error(t, err)
				var replyErr *typed.ReplyError
				require.True(t, errors.As(err, &replyErr))
				assert.Equal(t, tt.wantStatus, replyErr.Status)
				assert.Equal(t, tt.wantCode, replyErr.Code)
				assert.Equal(t, tt.wantMessage, replyErr.Message)
				assert.Equal(t, tt.wantBody, replyErr.Body)
				assert.Zero(t, result)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantResult, result)
		})
	}
}

var errAuth = errors.New("authentication failed")
var errNotFound = errors.New("not found")

func TestRequester_Request_WithErrorDecoder(t *testing.T) {
	decoder := func(status reply.Status, code string, body []byte) error {
		switch code {
		case "AUTH":
			return errAuth
		case "NOT_FOUND":
			return errNotFound
		default:
			return fmt.Errorf("unknown error: code=%s body=%s", code, string(body))
		}
	}

	tests := []struct {
		wantErr error
		resp    *producer.Response
		name    string
	}{
		{
			name: "decoder maps AUTH code to domain error",
			resp: &producer.Response{
				Data: []byte(`{"error":"auth failed"}`),
				Header: nats.Header{
					reply.HeaderStatus:    []string{string(reply.StatusError)},
					reply.HeaderErrorCode: []string{"AUTH"},
				},
			},
			wantErr: errAuth,
		},
		{
			name: "decoder maps NOT_FOUND code to domain error",
			resp: &producer.Response{
				Data: []byte(`{"error":"item not found"}`),
				Header: nats.Header{
					reply.HeaderStatus:    []string{string(reply.StatusError)},
					reply.HeaderErrorCode: []string{"NOT_FOUND"},
				},
			},
			wantErr: errNotFound,
		},
		{
			name: "decoder handles unknown code",
			resp: &producer.Response{
				Data: []byte(`{"error":"something"}`),
				Header: nats.Header{
					reply.HeaderStatus:    []string{string(reply.StatusError)},
					reply.HeaderErrorCode: []string{"UNKNOWN"},
				},
			},
		},
		{
			name: "decoder receives fail status",
			resp: &producer.Response{
				Data: []byte(`{"error":"rate limited"}`),
				Header: nats.Header{
					reply.HeaderStatus:    []string{string(reply.StatusFail)},
					reply.HeaderErrorCode: []string{"AUTH"},
				},
			},
			wantErr: errAuth,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mr := &mockRequester{reqResp: tt.resp}
			r, err := typed.NewRequester[order, processedOrder](
				mr, "orders.process",
				typed.JSONCodec[order]{},
				typed.JSONCodec[processedOrder]{},
				typed.WithErrorDecoder(decoder),
			)
			require.NoError(t, err)

			result, err := r.Request(context.Background(), order{ID: "1", Amount: 10})
			require.Error(t, err)
			assert.Zero(t, result)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.Contains(t, err.Error(), "unknown error")
			}
		})
	}
}

func TestRequester_Request_WithErrorDecoder_SuccessNotAffected(t *testing.T) {
	decoder := func(_ reply.Status, _ string, _ []byte) error {
		return errAuth
	}

	mr := &mockRequester{reqResp: &producer.Response{
		Data:   []byte(`{"status":"done","order_id":"1"}`),
		Header: nats.Header{reply.HeaderStatus: []string{string(reply.StatusSuccess)}},
	}}

	r, err := typed.NewRequester[order, processedOrder](
		mr, "orders.process",
		typed.JSONCodec[order]{},
		typed.JSONCodec[processedOrder]{},
		typed.WithErrorDecoder(decoder),
	)
	require.NoError(t, err)

	result, err := r.Request(context.Background(), order{ID: "1", Amount: 10})
	require.NoError(t, err)
	assert.Equal(t, "done", result.Status)
	assert.Equal(t, "1", result.OrderID)
}

func TestRequester_Request_WithoutErrorDecoder_FallsBackToReplyError(t *testing.T) {
	mr := &mockRequester{reqResp: &producer.Response{
		Data: []byte(`{"error":"boom"}`),
		Header: nats.Header{
			reply.HeaderStatus:    []string{string(reply.StatusError)},
			reply.HeaderErrorCode: []string{"SOME_CODE"},
		},
	}}

	r, err := typed.NewRequester[order, processedOrder](
		mr, "orders.process",
		typed.JSONCodec[order]{},
		typed.JSONCodec[processedOrder]{},
	)
	require.NoError(t, err)

	result, err := r.Request(context.Background(), order{ID: "1", Amount: 10})
	require.Error(t, err)
	assert.Zero(t, result)

	var replyErr *typed.ReplyError
	require.True(t, errors.As(err, &replyErr))
	assert.Equal(t, reply.StatusError, replyErr.Status)
	assert.Equal(t, "SOME_CODE", replyErr.Code)
}
