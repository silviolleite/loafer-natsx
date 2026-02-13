package processor

import "github.com/nats-io/nats.go"

func propagateHeaders(from, to *nats.Msg) {
	if from.Header == nil {
		return
	}

	if to.Header == nil {
		to.Header = nats.Header{}
	}

	for k, values := range from.Header {
		switch k {
		case "X-Correlation-ID", "traceparent", "tracestate", "baggage":
			for _, v := range values {
				to.Header.Add(k, v)
			}
		}
	}
}
