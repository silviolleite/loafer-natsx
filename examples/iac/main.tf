terraform {
  required_providers {
    jetstream = {
      source  = "nats-io/jetstream"
      version = "~> 0.3.0"
    }
  }
}

provider "jetstream" {
  // servers = "nats://localhost:4222" to use manual nats server
  servers = "nats://nats:4222" // run inside docker container
}

##################################################
# MAIN STREAM
##################################################

resource "jetstream_stream" "orders" {
  name = "ORDERS"

  subjects = [
    "orders.created",
    "orders.cancelled",
    "orders.failed",
    "orders.replay",
    "orders.dedup"
  ]

  retention        = "limits"
  storage          = "file"
  max_age          = 604800   # 7 days
  duplicate_window = 60      # 1 minute
  replicas         = 1
}

##################################################
# DLQ STREAM
##################################################

resource "jetstream_stream" "orders_dlq" {
  name = "ORDERS_DLQ"

  subjects = [
    "dlq.orders.failed"
  ]

  retention = "limits"
  storage   = "file"
  max_age   = 2592000  # 30 days
  replicas  = 1
}