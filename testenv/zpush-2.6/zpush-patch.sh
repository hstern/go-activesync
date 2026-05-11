#!/bin/sh
# Copyright (C) 2026 Henry Stern
# SPDX-License-Identifier: MIT

# Patch Z-Push's stock config files in place to point at the in-container
# Dovecot + Postfix + Radicale via BackendCombined. Keeps all the upstream
# constants (SCRIPT_TIMEOUT etc.) intact rather than maintaining a parallel
# config.
#
# Folder routing under BackendCombined:
#   mail folders          -> BackendIMAP    ('i')
#   appointments/calendar -> BackendCalDAV  ('c')
#   contacts              -> BackendCardDAV ('d')
#   tasks/notes/journal   -> BackendIMAP    ('i'; degenerate but never errors)
set -e

CONFIG=/usr/share/z-push/config.php
IMAP_CONFIG=/usr/share/z-push/backend/imap/config.php
CALDAV_CONFIG=/usr/share/z-push/backend/caldav/config.php
CARDDAV_CONFIG=/usr/share/z-push/backend/carddav/config.php
COMBINED_CONFIG=/usr/share/z-push/backend/combined/config.php

# --- main config ----------------------------------------------------------

sed -i "s|define('TIMEZONE', '');|define('TIMEZONE', 'UTC');|" "$CONFIG"
sed -i "s|define('BACKEND_PROVIDER',[ ]*'[^']*');|define('BACKEND_PROVIDER', 'BackendCombined');|" "$CONFIG"

# --- IMAP backend (mail) --------------------------------------------------

sed -i "s|define('IMAP_SERVER',[ ]*'[^']*');|define('IMAP_SERVER', '127.0.0.1');|" "$IMAP_CONFIG"
sed -i "s|define('IMAP_PORT',[ ]*[0-9]*);|define('IMAP_PORT', 143);|" "$IMAP_CONFIG"
sed -i "s|define('IMAP_OPTIONS',[ ]*'[^']*');|define('IMAP_OPTIONS', '/notls/norsh');|" "$IMAP_CONFIG"
sed -i "s|define('IMAP_SMTP_METHOD',[ ]*'[^']*');|define('IMAP_SMTP_METHOD', 'smtp');|" "$IMAP_CONFIG"

sed -i "s|define('IMAP_FOLDER_CONFIGURED', false);|define('IMAP_FOLDER_CONFIGURED', true);|" "$IMAP_CONFIG"
sed -i "s|define('IMAP_FOLDER_INBOX',[ ]*'[^']*');|define('IMAP_FOLDER_INBOX', 'INBOX');|" "$IMAP_CONFIG"
sed -i "s|define('IMAP_FOLDER_SENT',[ ]*'[^']*');|define('IMAP_FOLDER_SENT', 'Sent');|" "$IMAP_CONFIG"
sed -i "s|define('IMAP_FOLDER_DRAFT',[ ]*'[^']*');|define('IMAP_FOLDER_DRAFT', 'Drafts');|" "$IMAP_CONFIG"
sed -i "s|define('IMAP_FOLDER_TRASH',[ ]*'[^']*');|define('IMAP_FOLDER_TRASH', 'Trash');|" "$IMAP_CONFIG"
sed -i "s|define('IMAP_FOLDER_SPAM',[ ]*'[^']*');|define('IMAP_FOLDER_SPAM', 'Junk');|" "$IMAP_CONFIG"
sed -i "s|define('IMAP_FOLDER_ARCHIVE',[ ]*'[^']*');|define('IMAP_FOLDER_ARCHIVE', 'Archive');|" "$IMAP_CONFIG"

# Force UTF-8 as the default charset for body re-encoding. Stock Z-Push
# leaves this empty, which means SmartReply/SmartForward fall back to
# the original message's Content-Type charset header — and plain-text
# loopback messages from postfix arrive without one. The empty fallback
# trips PHP imap_*() into latin1, then the reply-quote conversion fails
# and Z-Push surfaces it as Status=120 (CannotConvertContent).
# (Tracking: github.com/hstern/go-activesync issue #3)
sed -i "s|define('IMAP_DEFAULT_CHARSET',[ ]*'[^']*');|define('IMAP_DEFAULT_CHARSET', 'UTF-8');|" "$IMAP_CONFIG"

# Inline forward: include the original body text in SmartForward output.
# The default is true in modern Z-Push, but pin it explicitly so a
# future upstream default-flip can't silently regress SmartForward.
sed -i "s|define('IMAP_INLINE_FORWARD',[ ]*[a-z]*);|define('IMAP_INLINE_FORWARD', true);|" "$IMAP_CONFIG"

# Mailbox folder prefix: Dovecot's `namespace inbox { inbox = yes }`
# config places caller-created mailboxes at the top level (not under
# INBOX). Pin the prefix to empty so Z-Push doesn't search for
# "INBOX.x" when the IMAP path is actually "x" — that mismatch is what
# triggers HTTP 500 on FolderDelete.
sed -i "s|define('IMAP_FOLDER_PREFIX',[ ]*'[^']*');|define('IMAP_FOLDER_PREFIX', '');|" "$IMAP_CONFIG"

cat >> "$IMAP_CONFIG" <<'EOF'

// Integration test override: SMTP via local postfix, no auth.
$imap_smtp_params = ['host' => '127.0.0.1', 'port' => 25, 'auth' => false];
EOF

# --- CalDAV backend (calendar) -------------------------------------------

sed -i "s|define('CALDAV_PROTOCOL',[ ]*'[^']*');|define('CALDAV_PROTOCOL', 'http');|" "$CALDAV_CONFIG"
sed -i "s|define('CALDAV_SERVER',[ ]*'[^']*');|define('CALDAV_SERVER', '127.0.0.1');|" "$CALDAV_CONFIG"
sed -i "s|define('CALDAV_PORT',[ ]*'[^']*');|define('CALDAV_PORT', '5232');|" "$CALDAV_CONFIG"
sed -i "s|define('CALDAV_PATH',[ ]*'[^']*');|define('CALDAV_PATH', '/%u/');|" "$CALDAV_CONFIG"
sed -i "s|define('CALDAV_PERSONAL',[ ]*'[^']*');|define('CALDAV_PERSONAL', 'calendar');|" "$CALDAV_CONFIG"
sed -i "s|define('CALDAV_SUPPORTS_SYNC',[ ]*[a-z]*);|define('CALDAV_SUPPORTS_SYNC', false);|" "$CALDAV_CONFIG"

# --- CardDAV backend (contacts) ------------------------------------------

sed -i "s|define('CARDDAV_PROTOCOL',[ ]*'[^']*');|define('CARDDAV_PROTOCOL', 'http');|" "$CARDDAV_CONFIG"
sed -i "s|define('CARDDAV_SERVER',[ ]*'[^']*');|define('CARDDAV_SERVER', '127.0.0.1');|" "$CARDDAV_CONFIG"
sed -i "s|define('CARDDAV_PORT',[ ]*'[^']*');|define('CARDDAV_PORT', '5232');|" "$CARDDAV_CONFIG"
sed -i "s|define('CARDDAV_PATH',[ ]*'[^']*');|define('CARDDAV_PATH', '/%u/');|" "$CARDDAV_CONFIG"
sed -i "s|define('CARDDAV_DEFAULT_PATH',[ ]*'[^']*');|define('CARDDAV_DEFAULT_PATH', '/%u/addressbook/');|" "$CARDDAV_CONFIG"
sed -i "s|define('CARDDAV_GAL_PATH',[ ]*'[^']*');|define('CARDDAV_GAL_PATH', '');|" "$CARDDAV_CONFIG"
sed -i "s|define('CARDDAV_SUPPORTS_SYNC',[ ]*[a-z]*);|define('CARDDAV_SUPPORTS_SYNC', false);|" "$CARDDAV_CONFIG"
sed -i "s|define('CARDDAV_SUPPORTS_FN_SEARCH',[ ]*[a-z]*);|define('CARDDAV_SUPPORTS_FN_SEARCH', false);|" "$CARDDAV_CONFIG"
sed -i "s|define('CARDDAV_URL_VCARD_EXTENSION',[ ]*'[^']*');|define('CARDDAV_URL_VCARD_EXTENSION', '.vcf');|" "$CARDDAV_CONFIG"

# --- Combined backend (router) -------------------------------------------
#
# Stock combined/config.php targets the legacy Kopano stack (BackendKopano
# 'z' for non-mail). Replace it wholesale with our IMAP + CalDAV + CardDAV
# routing. The replacement still defines the same `BackendCombinedConfig`
# class with `GetBackendCombinedConfig()` that combined.php calls.

cat > "$COMBINED_CONFIG" <<'EOF'
<?php
// Integration-test override: BackendIMAP + BackendCalDAV + BackendCardDAV.
class BackendCombinedConfig {
    public static function GetBackendCombinedConfig() {
        return array(
            'backends' => array(
                'i' => array('name' => 'BackendIMAP'),
                'c' => array('name' => 'BackendCalDAV'),
                'd' => array('name' => 'BackendCardDAV'),
            ),
            'delimiter' => '/',
            'folderbackend' => array(
                SYNC_FOLDER_TYPE_INBOX            => 'i',
                SYNC_FOLDER_TYPE_DRAFTS           => 'i',
                SYNC_FOLDER_TYPE_WASTEBASKET      => 'i',
                SYNC_FOLDER_TYPE_SENTMAIL         => 'i',
                SYNC_FOLDER_TYPE_OUTBOX           => 'i',
                SYNC_FOLDER_TYPE_OTHER            => 'i',
                SYNC_FOLDER_TYPE_USER_MAIL        => 'i',
                SYNC_FOLDER_TYPE_APPOINTMENT      => 'c',
                SYNC_FOLDER_TYPE_USER_APPOINTMENT => 'c',
                SYNC_FOLDER_TYPE_CONTACT          => 'd',
                SYNC_FOLDER_TYPE_USER_CONTACT     => 'd',
                // Tasks: BackendCalDAV exposes a Tasks ('T'-prefixed)
                // companion folder for every calendar collection — those
                // hold the VTODO components alongside the VEVENTs.
                SYNC_FOLDER_TYPE_TASK             => 'c',
                SYNC_FOLDER_TYPE_USER_TASK        => 'c',
                // Notes: no clean CalDAV/CardDAV mapping (EAS Notes is an
                // Outlook concept). Route to mail so the backend doesn't
                // error; integration tests for notes skip cleanly.
                SYNC_FOLDER_TYPE_NOTE             => 'i',
                SYNC_FOLDER_TYPE_USER_NOTE        => 'i',
                SYNC_FOLDER_TYPE_JOURNAL          => 'i',
                SYNC_FOLDER_TYPE_USER_JOURNAL     => 'i',
                SYNC_FOLDER_TYPE_UNKNOWN          => 'i',
            ),
            'rootcreatefolderbackend' => 'i',
        );
    }
}
EOF

echo "z-push config patched (BackendCombined: IMAP+CalDAV+CardDAV)"
