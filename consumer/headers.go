package consumer

import "github.com/nats-io/nats.go"

const (

	// HeaderErrorKey is the name of the header used to store error messages in message metadata.
	HeaderErrorKey = "X-Error"

	// HeaderRetryCountKey is the name of the header used to specify the count of retry attempts for a request.
	HeaderRetryCountKey = "X-Retry-Count"

	// HeaderCorrelationIDKey is the name of the header used to store a unique identifier for tracing a request across services.
	HeaderCorrelationIDKey = "X-Correlation-ID"

	// HeaderTraceParentKey is the name of the header used to propagate trace information in a distributed tracing system.
	HeaderTraceParentKey = "traceparent"

	// HeaderTraceStateKey is the name of the header used to propagate vendor-specific trace information in a distributed trace.
	HeaderTraceStateKey = "tracestate"

	// HeaderBaggageKey is the name of the header used to propagate optional application-specific context in a request.
	HeaderBaggageKey = "baggage"
)

func propagateHeaders(from, to *nats.Msg) {
	if from.Header == nil {
		return
	}

	if to.Header == nil {
		to.Header = nats.Header{}
	}

	for k, values := range from.Header {
		switch k {
		case HeaderCorrelationIDKey, HeaderTraceParentKey, HeaderTraceStateKey, HeaderBaggageKey:
			for _, v := range values {
				to.Header.Add(k, v)
			}
		}
	}
}
