#!/bin/sh
# Copyright (C) 2026 Henry Stern
# SPDX-License-Identifier: MIT

# Container entrypoint: refresh permissions on z-push state/log dirs
# and hand off to supervisord.
set -e

mkdir -p /var/log/z-push /var/lib/z-push
chown -R www-data:www-data /var/log/z-push /var/lib/z-push

# Radicale needs write access on its collections + a place for cache.
# Bind-mounting an empty volume on /var/lib/radicale would clobber the
# pre-seeded calendar/addressbook, so we don't volume-mount it; just
# refresh ownership in case of host-uid drift.
mkdir -p /var/lib/radicale/collections /var/lib/radicale/cache
chown -R radicale:radicale /var/lib/radicale

# Create the apache /var/run dir that some systems clean on boot.
mkdir -p /var/run/apache2 /var/log/apache2
chown -R www-data:www-data /var/run/apache2 /var/log/apache2

# Clean up stale pid files left by an ungraceful previous shutdown.
# `docker-compose restart` doesn't wipe /run, so without this dovecot
# refuses to start on second boot ("already running with PID …").
rm -f /run/dovecot/*.pid /run/apache2/apache2.pid

# Refresh postfix's config-derived files (chroot dir, etc.).
postfix set-permissions || true

exec /usr/bin/supervisord -n -c /etc/supervisor/supervisord.conf
