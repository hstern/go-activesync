#!/bin/sh
# Copyright (C) 2026 Henry Stern
# SPDX-License-Identifier: MIT

# Patch Z-Push's stock config files in place to point at the in-container
# Dovecot + Postfix + use BackendIMAP. Keeps all the upstream constants
# (SCRIPT_TIMEOUT etc.) intact rather than maintaining a parallel config.
set -e

CONFIG=/usr/share/z-push/config.php
IMAP_CONFIG=/usr/share/z-push/backend/imap/config.php

# Main config: choose BackendIMAP, point state + log at writable dirs.
sed -i "s|define('TIMEZONE', '');|define('TIMEZONE', 'UTC');|" "$CONFIG"
sed -i "s|define('STATE_MACHINE', 'FILE');|define('STATE_MACHINE', 'FILE');|" "$CONFIG"
sed -i "s|define('STATE_DIR', '/var/lib/z-push/');|define('STATE_DIR', '/var/lib/z-push/');|" "$CONFIG"
sed -i "s|define('LOGFILEDIR', '/var/log/z-push/');|define('LOGFILEDIR', '/var/log/z-push/');|" "$CONFIG"
sed -i "s|define('BACKEND_PROVIDER', '');|define('BACKEND_PROVIDER', 'BackendIMAP');|" "$CONFIG"

# Some Z-Push releases ship a different default for these — replace
# more aggressively in case the stock value isn't exactly the empty form.
sed -i "s|define('BACKEND_PROVIDER',[ ]*'[^']*');|define('BACKEND_PROVIDER', 'BackendIMAP');|" "$CONFIG"

# IMAP backend: localhost / no TLS / SMTP via PHP mail() through postfix.
sed -i "s|define('IMAP_SERVER',[ ]*'[^']*');|define('IMAP_SERVER', '127.0.0.1');|" "$IMAP_CONFIG"
sed -i "s|define('IMAP_PORT',[ ]*[0-9]*);|define('IMAP_PORT', 143);|" "$IMAP_CONFIG"
sed -i "s|define('IMAP_OPTIONS',[ ]*'[^']*');|define('IMAP_OPTIONS', '/notls/norsh');|" "$IMAP_CONFIG"

# Use SMTP via postfix on localhost:25.
sed -i "s|define('IMAP_SMTP_METHOD',[ ]*'[^']*');|define('IMAP_SMTP_METHOD', 'smtp');|" "$IMAP_CONFIG"

# Tell BackendIMAP that the folder names are explicitly configured;
# dovecot creates Sent/Drafts/Trash/Junk via auto-subscribe.
sed -i "s|define('IMAP_FOLDER_CONFIGURED', false);|define('IMAP_FOLDER_CONFIGURED', true);|" "$IMAP_CONFIG"
sed -i "s|define('IMAP_FOLDER_INBOX',[ ]*'[^']*');|define('IMAP_FOLDER_INBOX', 'INBOX');|" "$IMAP_CONFIG"
sed -i "s|define('IMAP_FOLDER_SENT',[ ]*'[^']*');|define('IMAP_FOLDER_SENT', 'Sent');|" "$IMAP_CONFIG"
sed -i "s|define('IMAP_FOLDER_DRAFT',[ ]*'[^']*');|define('IMAP_FOLDER_DRAFT', 'Drafts');|" "$IMAP_CONFIG"
sed -i "s|define('IMAP_FOLDER_TRASH',[ ]*'[^']*');|define('IMAP_FOLDER_TRASH', 'Trash');|" "$IMAP_CONFIG"
sed -i "s|define('IMAP_FOLDER_SPAM',[ ]*'[^']*');|define('IMAP_FOLDER_SPAM', 'Junk');|" "$IMAP_CONFIG"
sed -i "s|define('IMAP_FOLDER_ARCHIVE',[ ]*'[^']*');|define('IMAP_FOLDER_ARCHIVE', 'Archive');|" "$IMAP_CONFIG"

# SMTP host override (some z-push versions read from a global array).
cat >> "$IMAP_CONFIG" <<'EOF'

// Integration test override.
$imap_smtp_params = ['host' => '127.0.0.1', 'port' => 25, 'auth' => false];
EOF

echo "z-push config patched"
