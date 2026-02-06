package reply

import (
	"encoding/json"
	"errors"

	"github.com/nats-io/nats.go"
)

/*
Status represents a semantic response status sent via headers.
*/
type Status string

const (
	StatusSuccess Status = "success"
	StatusError   Status = "error"
	StatusFail    Status = "fail"
	StatusPartial Status = "partial"
)

const (
	HeaderStatus      = "X-Status"
	HeaderErrorCode   = "X-Error-Code"
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
func JSON(result any, handlerErr error) ([]byte, nats.Header, error) {
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
func WithStatus(result any, status Status) ([]byte, nats.Header, error) {
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
Error builds a plain text error reply.
*/
func Error(err error) ([]byte, nats.Header) {
	h := nats.Header{}
	h.Set(HeaderStatus, string(StatusError))
	h.Set(HeaderContentType, "text/plain")
	return []byte(err.Error()), h
}
