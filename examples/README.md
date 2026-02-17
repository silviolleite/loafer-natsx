# Examples

This directory contains runnable examples demonstrating how to use **loafer-natsx** with different messaging patterns:

- Core NATS (Pub/Sub, Queue, Request-Reply)
- JetStream durable consumers
- DLQ handling
- Deduplication
- Replay
- Broker with multiple routes

Infrastructure is provisioned automatically using **Docker Compose + Terraform**.

---

# Prerequisites

Make sure you have installed:

- Go (1.26+)
- Docker & Docker Compose
- (Optional) NATS CLI for debugging

---

# Running with Docker Compose (Recommended)

From the `examples` directory:

```bash
docker compose up
```

This will:

1. Start a NATS Server with JetStream enabled
2. Enable full debug logging (`-DV`)
3. Wait until the server is healthy
4. Run `terraform init`
5. Run `terraform apply -auto-approve`
6. Provision all JetStream Streams automatically

To stop everything:

```bash
docker compose down -v
```

---

# Infrastructure Details

Terraform configuration is located in:

```
./examples/iac
```

Streams are provisioned declaratively via Terraform.

Consumers are created dynamically by the application at startup.

---

# Running the Examples

After Docker Compose finishes successfully, open a new terminal.

Navigate to the desired example folder and execute:

```bash
go run .
```

Example:

```bash
cd broker
go run .
```

Each example is self-contained and demonstrates a specific messaging pattern.

---

# Development Workflow

Typical workflow:

```bash
docker compose up

# In separate terminals:
cd <example-folder>
go run .
```

---

# Notes

- Streams are managed via Terraform.
- Core NATS subjects do not require infrastructure provisioning.
- JetStream streams must exist before running JetStream-based examples.
- Debug logging is enabled for development purposes.
