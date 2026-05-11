# testenv/grommunio — Grommunio + gromox + grommunio-sync **(WIP)**

> **Status (2026-05-11):** scaffolding only. The Dockerfile installs
> the apt packages successfully; runtime configuration (gromox HTTP,
> mysql_adaptor, grommunio-sync, MariaDB init, supervisord, nginx
> reverse proxy, healthcheck wiring) is not yet written. Picks up
> from priority 2 in `RELEASE-PLAN.md`.

Single-container EAS test target for Grommunio — the OSS Exchange
replacement that succeeded Kopano. Compared to `testenv/zpush` and
`testenv/sogo`, this exercises native GAL and the highest EAS
protocol versions (12.0 through 16.1 — `Find`, modern calendar
features, ItemOperations extensions). Without this stack, library
coverage of the 16.x code paths is unit-tests-only.

## Quick reference

| | testenv/zpush | testenv/sogo | testenv/grommunio |
|---|---|---|---|
| EAS frontend | Z-Push 2.7.6 (stock) | sogo-activesync (Z-Push fork) | grommunio-sync (Z-Push fork) |
| Mail / store backend | Dovecot via BackendCombined/IMAP | Dovecot via SOGo | gromox MAPI store + MariaDB |
| GAL | unsupported (BackendIMAP gap) | SOGo SQL user-source | gromox native |
| Highest advertised EAS version | 14.0 (Z-Push default) | depends on sogo-activesync | up to 16.1 |
| Port | 8580 | 8581 | 8582 |

## What's left

The Dockerfile through the apt-install stage builds and exits cleanly
because the `CMD` is a sleep loop placeholder. To get to a healthy
EAS endpoint, the following needs to land:

1. **MariaDB initialisation.** Bootstrap the `grommunio` database +
   user at build time (same pattern as `testenv/sogo`'s postgres
   block). The schema comes from `grommunio-setup` normally; we'll
   need to extract the SQL or use `grommunio-cli` non-interactively
   to populate it. Open question: can `grommunio-cli user create
   integration@asmcp.test` run idempotently at build time, or does
   it need a running gromox? If the latter, init has to move to the
   entrypoint after MariaDB + gromox are up.

2. **gromox config files.** At minimum:
   - `http.cfg` (the `/Microsoft-Server-ActiveSync` listener)
   - `mysql_adaptor.cfg` (DB connection string + auth)
   - `imap.cfg` (so SOGo-style IMAP-side delivery works)
   - `exmdb_provider.cfg` (storage backend wiring)

3. **grommunio-sync config.** `/etc/grommunio-sync/config.ini`
   pointing at the local gromox HTTP socket. This is the EAS
   frontend; gromox's HTTP handles the raw protocol.

4. **supervisord program list.** mariadb / redis / gromox-http /
   gromox-event / grommunio-sync (or whatever the EAS launcher is)
   / postfix / nginx. Priorities matter — mariadb + redis before
   gromox, gromox before grommunio-sync.

5. **nginx reverse proxy.** `/Microsoft-Server-ActiveSync` →
   gromox HTTP. Autodiscover endpoint at
   `/Autodiscover/Autodiscover.xml` → wherever grommunio handles it
   (TBD).

6. **Healthcheck.** Same OPTIONS-with-creds shape as the zpush /
   sogo stacks; doesn't go green until everything above is wired.

7. **Test user provisioning.** Create `integration@asmcp.test` with
   password `integration` and a default Inbox. Either via
   `grommunio-cli` or by direct SQL inserts depending on what works
   non-interactively.

## Quick start (once it's done — placeholder)

```sh
make up        # build + start, wait for healthy (~60-120s first time)
make probe     # cmd/eas-autoprobe against the stack
make test      # go test -tags integration ./eas
make down      # stop + wipe volumes
```

Until then, `make up` will exit cleanly and report the container as
running but unhealthy (the placeholder `sleep infinity` keeps the
container alive without serving any actual EAS traffic).

## References

- Grommunio community apt repo:
  <https://download.grommunio.com/community/Debian_12/>
- Grommunio admin docs:
  <https://docs.grommunio.com/admin/installation.html>
- gromox upstream: <https://github.com/grommunio/gromox>
- grommunio-sync (Z-Push fork): <https://github.com/grommunio/grommunio-sync>
