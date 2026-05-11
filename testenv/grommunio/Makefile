# Copyright (C) 2026 Henry Stern
# SPDX-License-Identifier: MIT

# *** STATUS: WIP — see README.md for what's left. ***
# Right now this Makefile only builds the image. The runtime targets
# (up/probe/test) won't produce a healthy EAS endpoint until the
# stack configuration lands.

COMPOSE ?= $(shell command -v docker-compose >/dev/null 2>&1 && echo docker-compose || echo docker compose)

EAS_INTEGRATION_URL    := http://localhost:8582/Microsoft-Server-ActiveSync
EAS_INTEGRATION_USER   := integration
EAS_INTEGRATION_PASS   := integration
EAS_INTEGRATION_DEVICE := integration00000000000000000002

ENV := \
    EAS_INTEGRATION_URL=$(EAS_INTEGRATION_URL) \
    EAS_INTEGRATION_USER=$(EAS_INTEGRATION_USER) \
    EAS_INTEGRATION_PASS=$(EAS_INTEGRATION_PASS) \
    EAS_INTEGRATION_DEVICE=$(EAS_INTEGRATION_DEVICE)

.PHONY: up down test probe logs shell clean

up:
	$(COMPOSE) up --build -d
	@echo -n "waiting for healthy"
	@for i in $$(seq 1 180); do \
	  status=$$($(COMPOSE) ps --format json grommunio 2>/dev/null | grep -o '"Health":"[a-z]*"' | head -1 | cut -d'"' -f4); \
	  if [ "$$status" = "healthy" ]; then echo " — ready"; exit 0; fi; \
	  echo -n "."; sleep 2; \
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
	$(COMPOSE) exec grommunio bash

down:
	$(COMPOSE) down -v

clean: down
	docker image rm go-activesync-grommunio 2>/dev/null || true
