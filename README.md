# loafer-natsx

A structured, production-ready Go library for working with NATS and
JetStream.

`loafer-natsx` provides a clean abstraction layer for:

-   Core NATS publishing
-   JetStream publishing with deduplication
-   Route-based message consumption
-   Durable consumers
-   Retry and redelivery handling
-   Dead Letter Queue (DLQ)
-   Request--Reply patterns
-   Historical replay
-   Graceful shutdown handling

The library is designed around explicit configuration, clear separation
of concerns, and production-safe defaults.

------------------------------------------------------------------------

# Philosophy

The project follows these principles:

-   Explicit configuration over hidden behavior
-   Separation between Core NATS and JetStream concerns
-   Route-driven consumption model
-   Functional options pattern
-   Sentinel errors for validation
-   Context-aware shutdown
-   Production-grade resilience

------------------------------------------------------------------------

# Installation

``` bash
go get github.com/silviolleite/loafer-natsx
```

Requirements:

-   Go 1.26+
-   NATS Server
-   JetStream enabled for persistence features

------------------------------------------------------------------------

# Architecture

The project is organized into focused packages:

-   **conn** → Connection management
-   **stream** → JetStream stream provisioning
-   **producer** → Core and JetStream producers
-   **router** → Route definitions
-   **consumer** → Message consumption engine
-   **logger** → Logging abstraction

Each package has a single responsibility and avoids implicit behavior.

------------------------------------------------------------------------

# Connection Management

The `conn` package wraps NATS connection setup with safe defaults,
reconnect configuration, and structured options.

Connections are designed to be long-lived and support graceful shutdown
using `Drain()`.

------------------------------------------------------------------------

# Producer

Two distinct producer types are provided:

-   Core NATS producer
-   JetStream producer

JetStream publishing supports:

-   Message ID--based deduplication
-   Publish headers
-   Stream acknowledgements
-   Duplicate detection via `ack.Duplicate`

Deduplication is controlled by the stream's `DuplicateWindow`
configuration.

------------------------------------------------------------------------

# Router

Routes define how messages should be consumed.

Supported route types:

-   Pub/Sub
-   Queue Group
-   Request--Reply
-   JetStream Durable

JetStream routes support:

-   Durable consumers
-   AckWait configuration
-   MaxDeliver retry limits
-   Delivery policies
-   DLQ enablement
-   Handler timeout configuration

Routes are validated during creation and enforce required fields.

------------------------------------------------------------------------

# Consumer

The `consumer` package executes routes and handlers.

Features include:

-   Automatic subscription management
-   Context-aware shutdown
-   Core subscription draining
-   JetStream consumer stopping
-   Explicit Ack/Nak handling
-   DLQ publishing when max delivery is exceeded

Retry behavior for JetStream is controlled by:

-   `AckWait`
-   `MaxDeliver`

------------------------------------------------------------------------

# Dead Letter Queue (DLQ)

When enabled for JetStream routes:

-   Messages exceeding `MaxDeliver` are published to `dlq.<subject>`
-   Headers include:
    -   `X-Error`
    -   `X-Retry-Count`

Original messages are acknowledged after DLQ publishing to prevent
further retries.

------------------------------------------------------------------------

# Retry & Redelivery

JetStream consumers automatically retry failed messages.

Behavior is controlled by:

-   Ack wait duration
-   Maximum delivery attempts
-   Explicit negative acknowledgements

The consumer ensures proper redelivery flow and clean state transitions.

------------------------------------------------------------------------

# Historical Replay

Delivery policies allow replaying existing messages in a stream.

Supported modes:

-   Deliver all historical messages
-   Deliver only new messages
-   Deliver last message

Replay behavior is controlled at the route level.

------------------------------------------------------------------------

# Deduplication

Deduplication occurs during publish when a `MsgID` is provided.

If another message with the same `MsgID` is published within the
stream's duplicate window:

-   The message is not stored again
-   The server acknowledges the original sequence
-   `ack.Duplicate` is set to true

Deduplication is enforced at the stream level.

------------------------------------------------------------------------

# Graceful Shutdown

All consumers respect `context.Context`.

When the context is canceled:

-   Core subscriptions are drained
-   JetStream consumers are stopped
-   Connections can be gracefully drained

This ensures no message loss during shutdown.

------------------------------------------------------------------------

# Stream Provisioning

Streams must be provisioned explicitly using the `stream` package.

Provisioning is separate from runtime message publishing.

Streams support:

-   Subject configuration
-   Retention policy
-   Storage type
-   Duplicate window
-   Message age limits

------------------------------------------------------------------------

# Examples

See the `examples/` directory for full working examples:

-   Pub/Sub
-   Queue Group
-   Request--Reply
-   JetStream Durable Consumer
-   Retry and Redelivery
-   Dead Letter Queue (DLQ)
-   Historical Replay
-   Deduplication with MsgID

------------------------------------------------------------------------

# Production Recommendations

-   Always use `Drain()` instead of `Close()` during shutdown
-   Configure `DuplicateWindow` when using deduplication
-   Use durable consumers for persistent processing
-   Set `AckWait` according to handler execution time
-   Enable DLQ for resilient failure handling
-   Implement idempotent handlers when possible

------------------------------------------------------------------------

# Design Goals

-   No implicit stream creation
-   No hidden runtime behavior
-   Clear separation of Core and JetStream concerns
-   Safe-by-default configuration
-   Predictable retry semantics
-   Production-focused architecture

------------------------------------------------------------------------

# License

MIT