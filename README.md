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
-   Concurrent multi-route broker orchestration

The library is designed around explicit configuration, clear separation
of concerns, and production-safe defaults.

------------------------------------------------------------------------

# Philosophy

The project follows these principles:

-   Explicit configuration over hidden behavior
-   Clear separation between Core NATS and JetStream concerns
-   Route-driven consumption model
-   Functional options pattern
-   Sentinel errors for validation
-   Context-aware shutdown
-   Fail-fast orchestration
-   Concurrency safety by design
-   Production-grade resilience

------------------------------------------------------------------------

# Installation

    go get github.com/silviolleite/loafer-natsx

Requirements:

-   Go 1.26+
-   NATS Server
-   JetStream enabled for persistence features

------------------------------------------------------------------------

# Architecture

The project is organized into focused packages:

-   conn → Connection management
-   stream → JetStream stream provisioning
-   producer → Core and JetStream producers
-   router → Route definitions
-   consumer → Message consumption engine
-   broker → Multi-route concurrent orchestration
-   logger → Logging abstraction

Each package has a single responsibility and avoids implicit behavior.

------------------------------------------------------------------------

# Broker

The broker package allows running multiple routes concurrently within a
single service process.

It provides:

-   Registration of multiple validated routes with handlers
-   Configurable worker concurrency
-   Coordinated startup of all routes
-   Fail-fast behavior (if one route fails, all are stopped)
-   Context propagation across all routes
-   Global cancellation control
-   Safe shutdown without partial execution states

The broker guarantees:

-   Concurrency safety
-   No goroutine leaks
-   No silent route failures
-   Coordinated lifecycle management
-   Deterministic shutdown behavior

This enables building services that consume multiple subjects safely
without risking inconsistent runtime states.

------------------------------------------------------------------------

# Examples

See the examples directory:

https://github.com/silviolleite/loafer-natsx/tree/main/examples

------------------------------------------------------------------------

# License

MIT