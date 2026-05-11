# testenv/zpush-2.6 — Z-Push 2.6 + Dovecot + Postfix + Radicale

Mirror of [`testenv/zpush/`](../zpush/) with the **Z-Push version
pinned to 2.6.4** instead of 2.7.6. Same Dovecot, Postfix, Radicale,
test user, and configuration files; the only thing that changes is
the upstream source tarball.

Catches regressions against downstream deployments still on the
2.6.x line, where defaults around charset handling, FolderType
numbering edges, and SmartReply / SmartForward behave subtly
differently from 2.7.

## Quick start

```sh
make up         # build + start, wait for healthy
make probe      # cmd/eas-autoprobe against the running stack
make test       # go test -tags integration ./eas
make down       # tear down + wipe volumes
```

EAS endpoint: `http://localhost:8583/Microsoft-Server-ActiveSync`
(8580 is the 2.7.6 stack — both can run side-by-side).

Test credentials: `integration` / `integration` (email
`integration@asmcp.test`).

## Versus testenv/zpush

| | testenv/zpush | testenv/zpush-2.6 |
|---|---|---|
| Z-Push version | 2.7.6 (current stable) | 2.6.4 (last 2.6.x release) |
| Port | 8580 | 8583 |
| Container name | `go-activesync-zpush` | `go-activesync-zpush-2.6` |
| Everything else | identical | identical |

For Dovecot config, Radicale, the Z-Push patch script, and the
testenv-wide credential conventions see
[`testenv/zpush/README.md`](../zpush/README.md). This stack
deliberately stays a thin variant so the two versions remain
diff-able when 2.6-specific behaviour surfaces in tests.

## Known issues against this stack

Probing this stack with `cmd/eas-autoprobe` against a freshly-built
container returns **10 of 12 OK**, surfacing two real findings:

1. **`Provision` returns HTTP 500** — Z-Push 2.6.4's Provision
   handler throws `WBXMLException: Unknown error in
   Provisioning->Handle()` on PHP 8. The library is asking for the
   standard policy-key handshake; 2.6 fails internally. FolderSync
   and the rest succeed because 2.6 doesn't enforce the policy key
   as strictly as 2.7 — the spec mandates 449 retry on
   policy-violated commands, but this stack just lets them through.
   Worth filing upstream if anyone still cares about the 2.6 line.
2. **`ResolveRecipients` returns Status 5 (ServerError)** — same as
   the 2.7 stack; the BackendIMAP backend doesn't implement the GAL
   lookup verb. Already documented in
   `eas/README.md` "Server-specific notes".

The Dockerfile also carries one PHP-8 compatibility sed-patch
against `ipcsharedmemoryprovider.php:45`, where Z-Push 2.6's
`sprintf("%s", $semaphore)` debug-log call breaks because PHP 8's
`sem_get()` returns a `SysvSemaphore` object instead of a resource.
Without that patch the first request returns HTTP 500 unconditionally.
