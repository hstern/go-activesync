# Copyright (C) 2026 Henry Stern
# SPDX-License-Identifier: MIT

# testenv/ umbrella Makefile. Each EAS-compatible stack lives in its
# own subdir (one Dockerfile + docker-compose.yml + Makefile per
# stack). Targets here delegate to the chosen stack's Makefile.
#
# Stacks currently available:
#
#   zpush     Z-Push 2.7.6 + Dovecot + Postfix + Radicale (default)
#   sogo      SOGo + sogo-activesync + Dovecot + Postfix + PostgreSQL
#
# Usage:
#
#   make up                 # bring up the default stack (zpush)
#   make probe              # cmd/eas-autoprobe against the default stack
#   make test               # go test -tags integration against the default stack
#   make down               # tear down
#
# Use STACK= to select an alternate stack once more land:
#
#   make up STACK=sogo
#   make probe STACK=grommunio

STACK ?= zpush

.PHONY: up down test probe logs shell seed clean
up down test probe logs shell seed clean:
	$(MAKE) -C $(STACK) $@
