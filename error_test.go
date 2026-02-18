package loafernatsx_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	loafernatsx "github.com/silviolleite/loafer-natsx"
)

func TestErr_ErrorMethod(t *testing.T) {
	e := loafernatsx.Err("custom error")
	assert.Equal(t, "custom error", e.Error())
}

func TestSentinelErrors_ErrorStrings(t *testing.T) {
	tests := []struct {
		err      error
		expected string
	}{
		{loafernatsx.ErrUnsupportedType, "unsupported router type"},
		{loafernatsx.ErrMissingURL, "connection URL is required"},
		{loafernatsx.ErrMissingSubject, "subject is required"},
		{loafernatsx.ErrMissingQueueGroup, "queue group is required for the router"},
		{loafernatsx.ErrMissingStream, "stream is required for jetstream router"},
		{loafernatsx.ErrMissingDurable, "durable name is required for jetstream router"},
		{loafernatsx.ErrNilRoute, "route cannot be nil"},
		{loafernatsx.ErrNilHandler, "handler cannot be nil"},
		{loafernatsx.ErrNoRoutes, "no routes provided"},
		{loafernatsx.ErrNilRouteRegistration, "route registration cannot be nil"},
		{loafernatsx.ErrRequestNotSupported, "request-reply routes are not supported for JetStream producers"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.err.Error())
	}
}

func TestSentinelErrors_Equality(t *testing.T) {
	assert.True(t, errors.Is(loafernatsx.ErrUnsupportedType, loafernatsx.ErrUnsupportedType))
	assert.False(t, errors.Is(loafernatsx.ErrUnsupportedType, loafernatsx.ErrMissingSubject))
}

func TestSentinelErrors_AsErrorInterface(t *testing.T) {
	var err error = loafernatsx.ErrMissingSubject
	assert.Equal(t, "subject is required", err.Error())
}
