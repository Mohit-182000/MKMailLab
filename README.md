# LocalMail

A local SMTP server and inbox for developers. Point your application's mail
settings at `127.0.0.1:1025`, send an email, and inspect it instantly —
rendered HTML, plain text, headers, raw MIME and attachments — without any
email ever leaving your machine.

LocalMail is a native Windows desktop application (Go + Wails + Vue 3). It
needs no PHP, Node.js, Docker or database server on the user's machine.

> **Status:** early development — Phase 1 (foundation) complete.
> See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the full plan.

## Quick configuration (once SMTP lands in Phase 4)

| Setting  | Value       |
|----------|-------------|
| Host     | `127.0.0.1` |
| Port     | `1025`      |
| Username | *(empty)*   |
| Password | *(empty)*   |
| TLS      | none        |

Laravel `.env`:

```env
MAIL_MAILER=smtp
MAIL_HOST=127.0.0.1
MAIL_PORT=1025
MAIL_USERNAME=null
MAIL_PASSWORD=null
MAIL_ENCRYPTION=null
```

## Where LocalMail keeps data

| Mode                                   | Location                          |
|----------------------------------------|-----------------------------------|
| Installed                              | `%LOCALAPPDATA%\LocalMail\`       |
| Development build (`wails3 dev`)       | `%LOCALAPPDATA%\LocalMail-Dev\`   |
| Portable (`localmail.portable` beside the exe) | `.\data\` next to the exe |
| Custom                                 | `localmail.exe --data-dir <path>` |

Inside: `localmail.db` (SQLite), `logs\localmail.log` (rotated JSON logs),
`certs\` (TLS), `tmp\` (cleared at startup).

## Command-line flags

| Flag                 | Purpose                                   |
|----------------------|-------------------------------------------|
| `--minimized`        | Start hidden (used by "Start with Windows") |
| `--data-dir <path>`  | Override the data directory               |
| `--log-level <lvl>`  | `debug`, `info`, `warn` or `error`        |

## Documentation

- [ARCHITECTURE.md](docs/ARCHITECTURE.md) — design, components, data flow
- [DEVELOPMENT.md](docs/DEVELOPMENT.md) — toolchain, running, testing, building

Further guides (DATABASE, SMTP, SECURITY, BUILD-WINDOWS, TROUBLESHOOTING) are
written in the phase that implements each area.
