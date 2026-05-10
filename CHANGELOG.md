# Changelog

All notable changes to this project are documented here. The format
follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) loosely
and the project follows [Semantic Versioning](https://semver.org/).

## [v0.2.1] — 2026-05-10

No library changes since v0.2.0 — the `eas/` and `wbxml/` public API
and behavior are unchanged. This is a test- and infrastructure-hardening
release.

### Testing

- Unit-test coverage 87.9% → 99.3% across `eas/` and `wbxml/`. Every
  documented error path (state-store failures, malformed server
  responses, protocol-status branches, parser edge cases) now has a
  regression test. Remaining uncovered code is genuinely defensive
  stdlib failure paths (`xml.Marshal`, `gzip.Buffer` write/close,
  `http.NewRequestWithContext` with hard-coded inputs, `pkcs7`
  internal errors, `net.DefaultResolver` fallback).
- `wbxml` exercises every header-truncation, `mb_u_int32` overflow,
  and unexpected-token path in encode/decode.
- New shared test helpers (`errStateStore`, `errSentinel`,
  `countingState`) for state-store failure injection across packages.

### testenv

- Z-Push BackendCombined IMAP pins to address the known
  SmartReply/SmartForward Status=120, FolderDelete HTTP 500, and
  Ping cold-boot Status=7 edges (#3): `IMAP_DEFAULT_CHARSET=UTF-8`,
  `IMAP_INLINE_FORWARD=true`, `IMAP_FOLDER_PREFIX=''`.
- `testenv/README.md` gains a "Known limitations" section that
  documents these edges and the workarounds the integration tests
  already encode.

### CI

- `gotestsum` wraps both unit and integration test runs and emits
  JUnit XML alongside the existing coverage profile; uploads feed
  Codecov Test Analytics (with caveats — see
  [codecov/feedback#637](https://github.com/codecov/feedback/discussions/637)).
- README embeds the Codecov coverage sunburst chart under a new
  "Coverage map" subsection.

## [v0.2.0] — 2026-05-10

### Fixed

- `parseEASTime` now handles iCalendar basic form (`YYYYMMDDTHHMMSSZ`),
  the wire format Z-Push BackendCalDAV / BackendCardDAV emit. Without
  this, every Z-Push-fronted CalDAV event arrived with a zero
  `StartTime`.
- Ping response parser handles the spec-correct
  `<Folder>id</Folder>` text form (MS-ASCMD §2.2.2.11.2). Previously
  `Status` came back as `2` but `ChangedFolders` was always empty
  against Z-Push.
- Per-item Sync helpers (`Create*`/`Update*`/`Delete*` for Event,
  Contact, Task, Note) now do the documented Status=3
  reset-and-replay, matching `SyncCalendar` and `ApplyEmailChanges`.
- `Settings.GetOof` returns a hard error on unparseable timestamps
  instead of silently returning a zero `time.Time`.

### Changed

- Default `Config.DeviceType` flipped from `"MCP"` to `"GoActiveSync"`.
  Only affects callers leaving it blank; explicit settings unaffected.

### Testing

- `testenv` now bundles Radicale alongside Dovecot + Postfix;
  integration suite covers Email, Calendar, Contacts, Tasks, and Ping
  notification.
- Unit-test coverage 73% → 88%.

### Docs

- `AGENTS.md` describes the project for AI coding agents.
- Library docs no longer reference unannounced consumers.

## [v0.1.0] — 2026-05-09

Initial release. Microsoft Exchange ActiveSync (EAS) client for Go.
Implements every command in MS-ASCMD across protocol versions
12.0–16.1, all major auth schemes (Basic, Bearer, NTLM, SPNEGO,
mTLS), and the WBXML codec the spec is built on.

Tested against Z-Push 2.7.6 + Dovecot via the bundled testenv stack.

[v0.2.1]: https://github.com/hstern/go-activesync/releases/tag/v0.2.1
[v0.2.0]: https://github.com/hstern/go-activesync/releases/tag/v0.2.0
[v0.1.0]: https://github.com/hstern/go-activesync/releases/tag/v0.1.0
