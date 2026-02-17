# loafer-natsx

[![Go
Version](https://img.shields.io/badge/go-1.26+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![CI](https://github.com/silviolleite/loafer-natsx/actions/workflows/ci.yml/badge.svg)](https://github.com/silviolleite/loafer-natsx/actions)

A structured, Go library for working with NATS and
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
-   producer → Core and JetStream producers
-   router → Route definitions
-   consumer → Message consumption engine
-   broker → Multi-route concurrent orchestration
-   logger → Logging abstraction

## High-Level Architecture Diagram

                        ┌─────────────────────┐
                        │     Application     │
                        └──────────┬──────────┘
                                   │
                          ┌────────▼────────┐
                          │      Broker     │
                          │  (Orchestrator) │
                          └────────┬────────┘
                                   │
            ┌──────────────────────┼──────────────────────┐
            │                      │                      │
     ┌──────▼──────┐       ┌───────▼───────┐       ┌──────▼──────┐
     │   Router    │       │   Router      │       │   Router     │
     │ (Route A)   │       │ (Route B)     │       │ (Route N)    │
     └──────┬──────┘       └───────┬───────┘       └──────┬──────┘
            │                      │                      │
     ┌──────▼──────┐       ┌───────▼───────┐       ┌──────▼──────┐
     │  Consumer   │       │   Consumer    │       │   Consumer   │
     │ (Workers)   │       │  (Workers)    │       │  (Workers)   │
     └──────┬──────┘       └───────┬───────┘       └──────┬──────┘
            │                      │                      │
            └──────────────┬───────┴──────────────┬───────┘
                           │                      │
                     ┌─────▼─────┐          ┌─────▼─────┐
                     │   NATS    │          │ JetStream │
                     │  (Core)   │          │ Persistence│
                     └───────────┘          └───────────┘

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

# Dead Letter Queue (DLQ)

When enabled for JetStream routes:

-   Messages exceeding MaxDeliver are published to `dlq.<subject>`
-   Headers include:
    -   X-Error
    -   X-Retry-Count

------------------------------------------------------------------------

# Deduplication

Deduplication occurs during publish when a MsgID is provided.

If another message with the same MsgID is published within the stream's
duplicate window:

-   The message is not stored again
-   The server acknowledges the original sequence
-   ack.Duplicate is set to true

------------------------------------------------------------------------

# Graceful Shutdown

All consumers and brokers respect context.Context.

When the context is canceled:

-   Core subscriptions are drained
-   JetStream consumers are stopped
-   Broker cancels all routes
-   Connections can be gracefully drained

------------------------------------------------------------------------

# Examples

See the examples directory:

https://github.com/silviolleite/loafer-natsx/tree/main/examples

------------------------------------------------------------------------

# License

MIT