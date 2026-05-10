# Integration test environment

A single-container Z-Push stack used by the `go-activesync` integration
tests. Bundles Z-Push 2.7, Apache + PHP, Dovecot (IMAP), and Postfix
(local SMTP delivery) so the tests can exercise FolderSync, SyncEmail,
SendMail loopback, and friends without any external mail
infrastructure.

The container is local-only — never expose it to a real network.

## Quick start

```sh
make up      # build the image and start the stack (waits for healthy)
make test    # run the Go integration tests against it
make down    # tear everything down
```

`make test` sets the env vars the test file expects:

```
EAS_INTEGRATION_URL    http://localhost:8580/Microsoft-Server-ActiveSync
EAS_INTEGRATION_USER   integration
EAS_INTEGRATION_PASS   integration
EAS_INTEGRATION_DEVICE integration00000000000000000000
```

You can also export those yourself and run the tests directly:

```sh
go test -tags integration -v ./eas
```

## What the stack provides

| Component | Where           | Notes                                  |
|-----------|-----------------|----------------------------------------|
| Z-Push    | `:8580/Microsoft-Server-ActiveSync` | Apache + PHP, BackendIMAP |
| Dovecot   | inside container, `localhost:143`   | PAM-backed system users  |
| Postfix   | inside container, `localhost:25`    | LMTP → Dovecot → Maildir |
| Test user | `integration` / `integration`        | Maildir at `/home/integration/Maildir` |

`make seed` injects a fixture message into the inbox via IMAP APPEND
(useful when iterating on the SyncEmail tests).

## Pointing the tests at a different server

The integration tests are not Docker-specific — set the env vars to any
EAS endpoint (a Z-Push, SOGo, or even Exchange Online box you control)
and the same suite runs against it. Use `EAS_INTEGRATION_INSECURE=1`
for self-signed certs and `EAS_INTEGRATION_VERBOSE=1` for slog debug.

## Troubleshooting

- **Build hangs on apt fetch**: the upstream Z-Push repo
  (`repo.z-hub.io`) is occasionally slow. Re-run `make up`.
- **`make up` reports "timed out"**: `make logs` to see what's
  refusing to start. Apache misconfig is the usual culprit; the vhost
  in `apache-zpush.conf` is the place to look.
- **SendMail tests don't see the loopback message**: postfix may still
  be queueing on first boot. Wait a few seconds and re-run, or
  `make shell` and `mailq` to inspect.
- **No Calendar / Contacts folders in FolderSync**: this stack ships
  IMAP only (BackendIMAP). The corresponding tests in
  `integration_test.go` skip cleanly. To add CalDAV/CardDAV, switch
  Z-Push to BackendCombined and add a Radicale (or similar) container
  — left as an exercise; the IMAP-only path covers the common case.

## Files

- `Dockerfile.zpush` — image definition
- `docker-compose.yml` — the one-service stack
- `entrypoint.sh` — container init (perms + supervisord)
- `supervisord.conf` — runs apache + dovecot + postfix
- `apache-zpush.conf` — Apache vhost serving Z-Push
- `dovecot.conf` — IMAP + LMTP config, system-user passdb
- `postfix-main.cf` — local LMTP delivery only
- `zpush-config.php` — Z-Push main config (BackendIMAP)
- `zpush-imap.php` — IMAP backend pointed at localhost
- `Makefile` — `up`/`down`/`test`/`logs`/`shell`/`seed`/`clean`
