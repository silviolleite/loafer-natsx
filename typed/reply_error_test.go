package typed_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/silviolleite/loafer-natsx/reply"
	"github.com/silviolleite/loafer-natsx/typed"
)

var _ error = (*typed.ReplyError)(nil)

func TestReplyError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *typed.ReplyError
		want string
	}{
		{
			name: "returns message when present",
			err:  &typed.ReplyError{Status: reply.StatusError, Message: "auth failed"},
			want: "auth failed",
		},
		{
			name: "returns message with code present",
			err:  &typed.ReplyError{Status: reply.StatusError, Code: "AUTH", Message: "auth failed"},
			want: "auth failed",
		},
		{
			name: "returns fallback when no message",
			err:  &typed.ReplyError{Status: reply.StatusError},
			want: "reply error",
		},
		{
			name: "returns fallback when message is empty with code",
			err:  &typed.ReplyError{Status: reply.StatusFail, Code: "TIMEOUT"},
			want: "reply error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.err.Error())
		})
	}
}
