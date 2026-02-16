package reply

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/nats-io/nats.go"
)

/*
Status represents a semantic response status sent via headers.
*/
type Status string

const (

	// StatusSuccess indicates that the operation was completed successfully and no errors occurred.
	StatusSuccess Status = "success"

	// StatusError indicates that the operation encountered an error or was unsuccessful.
	StatusError Status = "error"

	// StatusFail indicates that the operation did not succeed but also did not encounter critical errors.
	StatusFail Status = "fail"

	// StatusPartial indicates that the operation was only partially completed or partially successful.
	StatusPartial Status = "partial"
)

const (

	// HeaderStatus represents the header key used to indicate the semantic response status in messages.
	HeaderStatus = "X-Status"

	// HeaderErrorCode represents the header key used to specify the error code in response messages.
	HeaderErrorCode = "X-Error-Code"

	// HeaderContentType represents the header key used to specify the media type of the content in HTTP messages.
	HeaderContentType = "Content-Type"
)

/*
CodedError represents an error with a semantic code.
*/
type CodedError interface {
	error
	Code() string
}

/*
JSON builds a JSON reply inferring status and error code automatically.
*/
func JSON(ctx context.Context, result any, handlerErr error) ([]byte, nats.Header, error) {
	h := nats.Header{}
	h.Set(HeaderContentType, "application/json")

	if handlerErr != nil {
		h.Set(HeaderStatus, string(StatusError))

		var ce CodedError
		if errors.As(handlerErr, &ce) {
			h.Set(HeaderErrorCode, ce.Code())
		}

		return []byte(handlerErr.Error()), h, nil
	}

	b, err := json.Marshal(result)
	if err != nil {
		return nil, nil, err
	}

	h.Set(HeaderStatus, string(StatusSuccess))
	return b, h, nil
}

/*
WithStatus builds a JSON reply with an explicit semantic status.
*/
func WithStatus(ctx context.Context, result any, status Status) ([]byte, nats.Header, error) {
	b, err := json.Marshal(result)
	if err != nil {
		return nil, nil, err
	}

	h := nats.Header{}
	h.Set(HeaderStatus, string(status))
	h.Set(HeaderContentType, "application/json")
	return b, h, nil
}

/*
WithError builds a plain text error reply.
*/
func WithError(err error) ([]byte, nats.Header) {
	h := nats.Header{}
	h.Set(HeaderStatus, string(StatusError))
	h.Set(HeaderContentType, "text/plain")
	return []byte(err.Error()), h
}
