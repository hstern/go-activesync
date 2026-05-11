#!/bin/sh
# Copyright (C) 2026 Henry Stern
# SPDX-License-Identifier: MIT
#
# SOGo testenv entrypoint. Bootstraps the dovecot passdb, ensures
# state directories have correct ownership, and hands off to
# supervisord.
set -e

# dovecot passwd-file. PLAIN scheme, single user, kept in /etc/dovecot
# rather than baked into the image so the password is easy to override
# at run time via a bind-mount if needed.
mkdir -p /etc/dovecot
if [ ! -f /etc/dovecot/users ]; then
    cat > /etc/dovecot/users <<'EOF'
integration:{PLAIN}integration:1000:1000::/home/integration::userdb_mail=maildir:/home/integration/Maildir
EOF
    chmod 640 /etc/dovecot/users
fi

# Postfix needs runtime dirs.
mkdir -p /var/spool/postfix/pid /var/spool/postfix/private
postfix set-permissions || true

# nginx runtime dir + log files writable for the nginx user.
mkdir -p /var/log/nginx /var/lib/nginx/body
chown -R www-data:www-data /var/log/nginx /var/lib/nginx

# SOGo runtime dirs.
mkdir -p /var/run/sogo /var/log/sogo /var/spool/sogo
chown -R sogo:sogo /var/run/sogo /var/log/sogo /var/spool/sogo

# Clean up stale pid files left by an ungraceful previous shutdown.
rm -f /run/dovecot/*.pid /run/sogo/*.pid

exec /usr/bin/supervisord -n -c /etc/supervisor/supervisord.conf
