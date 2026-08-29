// Package middleware defines the Handler and Middleware types, the Chain
// combinator, and the built-in observability middlewares (Metrics and
// OpenTelemetry) used to add cross-cutting concerns to NATS message
// processing.
//
// Middlewares wrap a consumer handler with additional behavior while keeping
// the handler signature unchanged. They can be composed with Chain and wired
// into the broker either globally, through broker.WithGlobalMiddleware, or per
// route registration, through broker.NewRouteRegistration.
//
// The package ships two observability middlewares out of the box:
//
//   - Metrics instruments processing with Prometheus collectors labeled by
//     subject.
//   - OTel creates a distributed tracing span per message and continues any
//     trace propagated through the NATS message headers.
//
// Additional metrics or tracing backends can be integrated by implementing the
// Middleware type directly, so the package is not limited to Prometheus and
// OpenTelemetry.
package middleware
