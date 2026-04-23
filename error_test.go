package loafernatsx_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

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
		{loafernatsx.ErrRequestTimeout, "request timeout: consumer did not reply in time"},
		{loafernatsx.ErrPermanentFailure, "permanent failure: message acknowledged without retry"},
		{loafernatsx.ErrSendToDLQ, "send to dead letter queue: skip retries"},
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

func TestSentinelErrors_WrappedIs(t *testing.T) {
	wrapped := fmt.Errorf("context: %w", loafernatsx.ErrPermanentFailure)
	assert.True(t, errors.Is(wrapped, loafernatsx.ErrPermanentFailure))

	wrappedDLQ := fmt.Errorf("context: %w", loafernatsx.ErrSendToDLQ)
	assert.True(t, errors.Is(wrappedDLQ, loafernatsx.ErrSendToDLQ))
}

func TestNakWithDelayError_ErrorMessage(t *testing.T) {
	e := loafernatsx.NakWithDelayError{Delay: 30 * time.Second}
	assert.Equal(t, "nak with delay: redelivery in 30s", e.Error())
}

func TestNakWithDelayError_ZeroDelay(t *testing.T) {
	e := loafernatsx.NakWithDelayError{}
	assert.Equal(t, "nak with delay: redelivery in 0s", e.Error())
}

func TestNakWithDelayError_ErrorsAs(t *testing.T) {
	original := loafernatsx.NakWithDelayError{Delay: 5 * time.Minute}
	wrapped := fmt.Errorf("processing: %w", original)

	var target loafernatsx.NakWithDelayError
	assert.True(t, errors.As(wrapped, &target))
	assert.Equal(t, 5*time.Minute, target.Delay)
}

func TestNakWithDelayError_IsNotSentinel(t *testing.T) {
	e := loafernatsx.NakWithDelayError{Delay: time.Second}
	assert.False(t, errors.Is(e, loafernatsx.ErrPermanentFailure))
	assert.False(t, errors.Is(e, loafernatsx.ErrSendToDLQ))
}
