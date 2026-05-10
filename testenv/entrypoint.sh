#!/bin/sh
# Copyright (C) 2026 Henry Stern
# SPDX-License-Identifier: MIT

# Container entrypoint: refresh permissions on z-push state/log dirs
# and hand off to supervisord.
set -e

mkdir -p /var/log/z-push /var/lib/z-push
chown -R www-data:www-data /var/log/z-push /var/lib/z-push

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
