# testenv/sogo — SOGo + sogo-activesync **(WIP)**

> **Status (2026-05-11):** the image builds, but the stack isn't fully
> healthy yet. PostgreSQL, Dovecot, memcached, and nginx start cleanly;
> sogod and postfix don't yet stay up under supervisord. See the
> "Known issues" section at the bottom for current state and a debug
> hand-off.

Single-container EAS test target mirroring the deployment shape of
`mail.stern.ca`: SOGo as the groupware front, sogo-activesync (Inverse's
Z-Push fork) as the EAS endpoint, all behind nginx.

## Quick start

```sh
make up         # build + start, wait for healthy (~30-60s first time)
make probe      # cmd/eas-autoprobe against the running stack
make test       # go test -tags integration ./eas
make down       # stop and wipe volumes
```

The container exposes the EAS endpoint at:

```
http://localhost:8581/Microsoft-Server-ActiveSync
```

Test credentials: `integration` / `integration` (email
`integration@asmcp.test`).

## What's inside

| Component | Role | Listen |
|---|---|---|
| **nginx** | reverse proxy; fronts SOGo + the EAS endpoint | 80 (mapped to host 8581) |
| **SOGo** (`sogod`) | groupware: web UI + EAS via internal proxy | 127.0.0.1:20000 |
| **sogo-activesync** | Z-Push fork bundled with SOGo; handles the EAS protocol | served via SOGo |
| **PostgreSQL 15** | SOGo profile / folder / session store | 127.0.0.1:5432 |
| **Dovecot** | IMAP backend SOGo talks to for mail | 127.0.0.1:143 |
| **Postfix** | local delivery (so SendMail loopback works) | 127.0.0.1:25 |
| **memcached** | SOGo session cache | 127.0.0.1:11211 |

The PostgreSQL database `sogo` and a single test user
`integration / integration` are seeded at image-build time, so the
container comes up ready to serve requests without entrypoint
bootstrapping.

## Known interop notes

This stack reproduces the SOGo-specific behaviors the library has
to handle gracefully. The most consequential one:

- **Autodiscover mobilesync schema rejection.** SOGo's autodiscover
  module implements only the Outlook request schema. POSTing the EAS
  `mobilesync` schema gets HTTP 400 with `<ErrorCode>601</ErrorCode>
  "Not supported xmlns"`. v1.1.0 added a well-known fallback that
  resolves the EAS endpoint via OPTIONS probe; this testenv exercises
  that path end-to-end.

See [`eas/README.md`](../../eas/README.md#server-specific-notes) for
the full list of server-specific behaviors the library handles.

## Iteration tips

- `make logs` follows supervisord output — useful when SOGo or
  nginx is misbehaving.
- `make shell` drops into a bash inside the container. Useful for
  `psql -U sogo sogo`, `dovecot reload`, etc.
- A `make down && make up` rebuilds the image and wipes volumes
  (so the postgres state restarts clean). Skip `down` to keep the
  volumes across rebuilds.

## Comparison with `testenv/zpush/`

| | testenv/zpush | testenv/sogo |
|---|---|---|
| EAS frontend | stock Z-Push 2.7.6 | sogo-activesync (Z-Push fork) |
| Mail backend | Dovecot via BackendCombined/BackendIMAP | Dovecot via SOGo |
| Calendar/Contacts | Radicale via BackendCalDAV/CardDAV | SOGo native (PostgreSQL) |
| Autodiscover surface | Z-Push's responder (mobilesync OK) | SOGo's responder (mobilesync rejected) |
| Port | 8580 | 8581 |

## Known issues (WIP debugging notes)

As of 2026-05-11 the stack starts but neither sogod nor postfix
stays up under supervisord. Everything else (postgres, dovecot,
memcached, nginx) runs cleanly. Open items:

1. **sogod can't bind its WOWatchDog socket.** Log:
   ```
   sogod: [WARN] <WOWatchDog> listening socket: attempt N failed
   sogod: [ERROR] <WOWatchDog> unable to listen on specified port,
          check that no other process is already using it
   ```
   Nothing visible on `ss -ltnp` is holding port 20000. The
   watchdog socket may be a unix-domain socket whose path isn't
   writable; `/run/sogo/` is `0750 sogo:sogo` but the actual path
   sogod is trying to bind is undiscovered. Next debug step:
   `strace -f -e trace=bind` against sogod (needs strace package
   added to the Dockerfile).

2. **postfix daemonises despite `master -d`.** `master -d` exits
   immediately on debian:12 here even though that's documented as
   the no-detach form. Postfix DOES end up listening on 25
   (visible in `ss -ltnp`), but supervisord respawns the wrapper
   in a tight loop. Likely needs a wrapper similar to
   `run-sogod.sh`, or run via `postfix start` (which daemonises)
   plus a `tail --pid` tracker.

3. **Healthcheck never goes green.** Direct consequence of (1).
   Until sogod binds, the OPTIONS probe against
   `/Microsoft-Server-ActiveSync` returns nothing useful and the
   container stays unhealthy.

Suggested order to resume: fix (1) first — get sogod bound and
listening. (3) clears on its own once that lands. (2) is annoying
but doesn't block protocol testing since postfix isn't on the EAS
path (it's only there for `SendMail` loopback, which isn't on the
critical path for the autodiscover/mobilesync interop test that
motivated this stack in the first place).
