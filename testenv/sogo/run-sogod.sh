#!/bin/sh
# Copyright (C) 2026 Henry Stern
# SPDX-License-Identifier: MIT
#
# Wrapper for supervisord. SOGo's sogod forks into a watchdog +
# worker pool and the parent exits, which supervisord interprets as
# a crash. Run sogod, then exec `tail --pid` so supervisord keeps a
# live process to track for as long as the daemon is up.

set -e

PIDFILE=/run/sogo/sogo.pid

mkdir -p /run/sogo
chown sogo:sogo /run/sogo 2>/dev/null || true
rm -f "$PIDFILE"

/usr/sbin/sogod -WOLogFile - -WOPidFile "$PIDFILE"

# Give the daemon a moment to write its pid.
for i in 1 2 3 4 5 6 7 8 9 10; do
    [ -f "$PIDFILE" ] && break
    sleep 1
done

PID=$(cat "$PIDFILE" 2>/dev/null || true)
if [ -z "$PID" ] || ! kill -0 "$PID" 2>/dev/null; then
    echo "sogod failed to start (no valid pid in $PIDFILE)" >&2
    exit 1
fi

echo "sogod running with pid $PID; tracking via tail --pid"
exec tail --pid="$PID" -f /dev/null
