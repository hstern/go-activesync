# Integration test environment

A single-container Z-Push stack used by the `go-activesync` integration
tests. Bundles Z-Push 2.7 (configured for `BackendCombined`), Apache +
PHP, Dovecot (IMAP for mail), Postfix (local SMTP delivery), and
Radicale (CalDAV + CardDAV for calendar and contacts) so the tests can
exercise FolderSync, SyncEmail, SendMail loopback, Calendar CRUD,
Contacts CRUD, and friends without any external mail infrastructure.

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
| Z-Push    | `:8580/Microsoft-Server-ActiveSync` | Apache + PHP, BackendCombined → IMAP+CalDAV+CardDAV |
| Dovecot   | inside container, `localhost:143`   | PAM-backed system users  |
| Postfix   | inside container, `localhost:25`    | LMTP → Dovecot → Maildir |
| Radicale  | inside container, `localhost:5232`  | htpasswd auth, pre-seeded `/integration/calendar` and `/integration/addressbook` |
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
- **Calendar / Contacts events created via the Go client don't reappear
  on the same client's next Sync**: this is correct EAS behavior — the
  server doesn't echo a client's own adds back to it. The Calendar CRUD
  test verifies the round-trip with a *second* client (different
  DeviceID) so the bootstrap-then-Sync dance returns the new item.
- **`Radicale: 5xx` in z-push logs**: usually a permissions glitch on
  `/var/lib/radicale/collections`. `make logs` will show the trace.
  `entrypoint.sh` re-`chown`s on every boot, so a `make down && make up`
  cycle is the fastest recovery.

## Known limitations

The Z-Push 2.7 BackendCombined stack has a few rough edges that the
go-activesync integration tests and the downstream activesync-mcp Tier 3
e2e suite work around. Tracked in [issue #3](https://github.com/hstern/go-activesync/issues/3).

- **Ping returns Status=7 (FolderHierarchyOutOfDate) on the very first
  call after FolderSync.** Z-Push needs at least one prior SyncEmail
  per folder before it has a baseline to detect changes against. The
  integration tests bootstrap each folder with `SyncEmail{WindowSize: 1,
  NoBootstrap: true}` before starting the Ping watcher
  (see `TestIntegration_Ping_NotifiesOnNewEmail`). Production callers
  that hit Status=7 should re-do FolderSync and re-subscribe; this is
  spec-compliant recovery behavior, not a Z-Push bug we can patch.

- **SmartReply/SmartForward returns Status=120 (CannotConvertContent)
  on plain-text-only originals.** `zpush-patch.sh` now forces
  `IMAP_DEFAULT_CHARSET = 'UTF-8'` and pins `IMAP_INLINE_FORWARD = true`
  — those address the most common cause (the empty fallback charset
  triggers a latin1 round-trip that fails on UTF-8 bodies). Reply
  composition against MIME-multipart originals still occasionally hits
  120 due to BackendIMAP's quoted-printable splicing; a downstream
  retry with HTML body is the recommended workaround.

- **FolderDelete on caller-created top-level folders returns HTTP 500.**
  Dovecot's `namespace inbox { inbox = yes }` puts caller-created
  mailboxes at the top level rather than under `INBOX.`; Z-Push
  BackendIMAP previously assumed an `INBOX.` prefix when resolving the
  IMAP path on delete. `zpush-patch.sh` now pins `IMAP_FOLDER_PREFIX = ''`
  to match. If the 500 persists, callers can move the operation under
  an existing folder (e.g. `parent_id = <Inbox ID>`).

If you change anything in `zpush-patch.sh` to address these, rebuild
the image (`make down && make up`) before running `make test` — the
patches are baked in at build time.

## Files

- `Dockerfile.zpush` — image definition
- `docker-compose.yml` — the one-service stack
- `entrypoint.sh` — container init (perms + supervisord)
- `supervisord.conf` — runs apache + dovecot + postfix + radicale
- `apache-zpush.conf` — Apache vhost serving Z-Push
- `dovecot.conf` — IMAP + LMTP config, system-user passdb
- `postfix-main.cf` — local LMTP delivery only
- `radicale.conf` — Radicale CalDAV/CardDAV server config (htpasswd auth)
- `zpush-patch.sh` — sed-patches stock Z-Push configs to BackendCombined
  with IMAP+CalDAV+CardDAV routing
- `Makefile` — `up`/`down`/`test`/`logs`/`shell`/`seed`/`clean`
