package typed

import (
	"encoding/json"

	"github.com/silviolleite/loafer-natsx/reply"
)

// ReplyError represents an error reported by the consumer handler via response headers.
type ReplyError struct {
	Status  reply.Status
	Code    string
	Message string
	Body    []byte
}

// Error returns the original error message from the consumer when available.
// Structured metadata (Status, Code) is accessible via the exported fields
// using errors.As.
func (e *ReplyError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "reply error"
}

// extractErrorMessage attempts to extract the "error" field from a JSON body.
// Returns empty string if the body is not JSON or doesn't contain an "error" field.
func extractErrorMessage(body []byte) string {
	var payload struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &payload) == nil && payload.Error != "" {
		return payload.Error
	}
	// Fall back to plain text body for non-JSON replies (e.g. reply.WithError).
	if len(body) > 0 {
		return string(body)
	}
	return ""
}
