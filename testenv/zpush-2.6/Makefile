# Copyright (C) 2026 Henry Stern
# SPDX-License-Identifier: MIT

# Drives the integration test stack from the host.
#
# Usage:
#   make up         # docker compose build + start, wait for healthy
#   make test       # run integration tests against the running stack
#   make probe      # run cmd/eas-autoprobe against the running stack
#   make logs       # follow container logs
#   make shell      # interactive shell inside the container
#   make seed       # inject a test message into the mailbox via IMAP APPEND
#   make down       # stop + remove containers and volumes
#   make clean      # down + prune build cache for this image

# Use docker compose plugin if available; fall back to standalone docker-compose.
COMPOSE ?= $(shell command -v docker-compose >/dev/null 2>&1 && echo docker-compose || echo docker compose)

EAS_INTEGRATION_URL    := http://localhost:8583/Microsoft-Server-ActiveSync
EAS_INTEGRATION_USER   := integration
EAS_INTEGRATION_PASS   := integration
EAS_INTEGRATION_DEVICE := integration00000000000000000003

ENV := \
    EAS_INTEGRATION_URL=$(EAS_INTEGRATION_URL) \
    EAS_INTEGRATION_USER=$(EAS_INTEGRATION_USER) \
    EAS_INTEGRATION_PASS=$(EAS_INTEGRATION_PASS) \
    EAS_INTEGRATION_DEVICE=$(EAS_INTEGRATION_DEVICE)

.PHONY: up down test probe logs shell seed clean

up:
	$(COMPOSE) up --build -d
	@echo -n "waiting for healthy"
	@for i in $$(seq 1 60); do \
	  status=$$($(COMPOSE) ps --format json zpush-2-6 2>/dev/null | grep -o '"Health":"[a-z]*"' | head -1 | cut -d'"' -f4); \
	  if [ "$$status" = "healthy" ]; then echo " — ready"; exit 0; fi; \
	  echo -n "."; sleep 1; \
	done; \
	echo " — timed out; check 'make logs'"; exit 1

test:
	cd ../.. && $(ENV) go test -tags integration -v -count=1 ./eas

probe:
	cd ../.. && \
	    EAS_PROBE_USER=$(EAS_INTEGRATION_USER) \
	    EAS_PROBE_SERVER=$(EAS_INTEGRATION_URL) \
	    EAS_PROBE_PASSWORD=$(EAS_INTEGRATION_PASS) \
	    go run ./cmd/eas-autoprobe -device-id $(EAS_INTEGRATION_DEVICE)

logs:
	$(COMPOSE) logs -f

shell:
	$(COMPOSE) exec zpush-2-6 bash

seed:
	@echo "appending a fixture message to integration's INBOX"
	$(COMPOSE) exec -T zpush-2-6 bash -lc '\
	  msg="From: alice@asmcp.test\r\nTo: integration@asmcp.test\r\nSubject: hello from seed\r\n\r\nseeded body\r\n"; \
	  printf "$$msg" | curl -s --url "imap://127.0.0.1/INBOX" \
	    -u integration:integration -T - --upload-file -'

down:
	$(COMPOSE) down -v

clean: down
	docker image rm go-activesync-zpush-2.6 2>/dev/null || true
