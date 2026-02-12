#!/bin/bash
# up-nats-server.sh
# run nats server with jetStream

set -e

docker run -p 4222:4222 -ti nats:latest -js -sd /data/stan
