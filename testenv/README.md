# testenv

Docker stacks for integration-testing the `eas` client against
EAS-compatible mail servers. One subdir per stack; pick the one
your interop question is about, or add a new one when the existing
stacks don't cover what you need.

## Available stacks

| Subdir | Server stack | What it covers |
|--------|--------------|----------------|
| [`zpush/`](zpush/) | Z-Push 2.7.6 + Dovecot + Postfix + Radicale, routed via BackendCombined / BackendIMAP / BackendCalDAV / BackendCardDAV | The default. Email / Calendar / Contacts / Tasks across the most common OSS EAS deployment. |
| [`zpush-2.6/`](zpush-2.6/) | Same as above but pinned to Z-Push 2.6.4 with a PHP-8 compatibility patch | Regression coverage against deployments still on the 2.6.x line. Surfaces a Provision-handler issue specific to 2.6 on modern PHP. |

More stacks are planned — see `RELEASE-PLAN.md` for the priority
order (SOGo, Grommunio, Z-Push + BackendKopano, ...).

## Quick start

```sh
cd testenv
make up           # bring up the default stack (zpush)
make probe        # run cmd/eas-autoprobe against it
make test         # run the integration test suite against it
make down
```

Or directly:

```sh
cd testenv/zpush
make up && make test && make down
```

The umbrella `testenv/Makefile` accepts `STACK=<name>` to select a
different stack once more land. See each stack's `README.md` for
its own credentials, ports, and known limitations.
