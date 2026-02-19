package reply_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/silviolleite/loafer-natsx/reply"
)

type codedErr struct {
	msg  string
	code string
}

func (e codedErr) Error() string { return e.msg }
func (e codedErr) Code() string  { return e.code }

func TestJSON_Success(t *testing.T) {
	data := map[string]any{
		"id": 1,
	}

	b, h, err := reply.JSON(context.Background(), data, nil)

	assert.NoError(t, err)
	assert.Equal(t, "application/json", h.Get(reply.HeaderContentType))
	assert.Equal(t, string(reply.StatusSuccess), h.Get(reply.HeaderStatus))
	assert.Contains(t, string(b), `"id":1`)
}

func TestJSON_ErrorWithoutCode(t *testing.T) {
	handlerErr := errors.New("boom")

	b, h, err := reply.JSON(context.Background(), nil, handlerErr)

	assert.NoError(t, err)
	assert.Equal(t, "application/json", h.Get(reply.HeaderContentType))
	assert.Equal(t, string(reply.StatusError), h.Get(reply.HeaderStatus))
	assert.Empty(t, h.Get(reply.HeaderErrorCode))
	assert.Equal(t, "boom", string(b))
}

func TestJSON_ErrorWithCode(t *testing.T) {
	handlerErr := codedErr{
		msg:  "failure",
		code: "E123",
	}

	b, h, err := reply.JSON(context.Background(), nil, handlerErr)

	assert.NoError(t, err)
	assert.Equal(t, string(reply.StatusError), h.Get(reply.HeaderStatus))
	assert.Equal(t, "E123", h.Get(reply.HeaderErrorCode))
	assert.Equal(t, "failure", string(b))
}

func TestJSON_MarshalError(t *testing.T) {
	ch := make(chan int)

	b, h, err := reply.JSON(context.Background(), ch, nil)

	assert.Error(t, err)
	assert.Nil(t, b)
	assert.Nil(t, h)
}

func TestWithStatus(t *testing.T) {
	data := map[string]string{"ok": "true"}

	b, h, err := reply.WithStatus(
		context.Background(),
		data,
		reply.StatusPartial,
	)

	assert.NoError(t, err)
	assert.Equal(t, string(reply.StatusPartial), h.Get(reply.HeaderStatus))
	assert.Equal(t, "application/json", h.Get(reply.HeaderContentType))
	assert.Contains(t, string(b), `"ok":"true"`)
}

func TestWithStatus_MarshalError(t *testing.T) {
	ch := make(chan int)

	b, h, err := reply.WithStatus(
		context.Background(),
		ch,
		reply.StatusSuccess,
	)

	assert.Error(t, err)
	assert.Nil(t, b)
	assert.Nil(t, h)
}

func TestWithError(t *testing.T) {
	err := errors.New("fatal")

	b, h := reply.WithError(err)

	assert.Equal(t, string(reply.StatusError), h.Get(reply.HeaderStatus))
	assert.Equal(t, "text/plain", h.Get(reply.HeaderContentType))
	assert.Equal(t, "fatal", string(b))
}
