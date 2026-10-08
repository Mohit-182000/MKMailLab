# MKMailLab

A local SMTP server and inbox for developers. Point your application's mail
settings at `127.0.0.1:1025`, send an email, and inspect it instantly —
rendered HTML, plain text, headers, raw MIME and attachments — without any
email ever leaving your machine.

MKMailLab is a native Windows desktop application (Go + Wails + Vue 3). It
needs no PHP, Node.js, Docker or database server on the user's machine.

> **Status:** usable MVP — SMTP capture, inbox, preview, configuration and a
> Windows installer. See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the
> roadmap.

## Install

1. Build the installer (or use one already built): `wails3 task package` →
   `bin\MKMailLab-Setup-0.1.0.exe`
2. Run it. It installs per-user into `%LOCALAPPDATA%\Programs\MKMailLab`
   (no admin rights), adds Start Menu and desktop shortcuts and an entry in
   *Settings → Apps* for uninstalling.
3. Open **MKMailLab**. The SMTP server starts automatically on
   `127.0.0.1:1025`.

Silent install: `MKMailLab-Setup-0.1.0.exe /S`. Uninstalling keeps your
captured emails in `%LOCALAPPDATA%\MKMailLab`.

## Using it

- **Header switch.** Connect or disconnect (start or stop) the SMTP server.
  The pill shows status and address; click it to open Configuration.
- **Inbox tab.** Every captured email, newest first, with live updates.
  - **Search and filter:** search by subject, sender or recipient, or show
    only unread mail.
  - **Inspect:** Preview (sandboxed, with a mobile-width toggle), Plain
    text, Headers, Raw source and Attachments.
  - **Save:** save the email as `.eml` or save individual attachments.
  - **Keyboard:** `↑/↓` move, `Del` delete, `S` star, `U` mark unread,
    `R` refresh, `Ctrl+K` search, `Esc` close.
- **Configuration tab.**
  - **Connection details:** host, port, credentials and encryption, each
    with a copy button.
  - **Framework snippets:** Laravel, Node.js, Python, .NET, PHP and Django.
  - **Server settings:** listen address, port, auth mode, size and
    connection limits, autostart.
  - **Test email:** send one to verify the whole setup.

## Quick configuration

| Setting  | Value       |
|----------|-------------|
| Host     | `127.0.0.1` |
| Port     | `1025`      |
| Username | *(empty, or anything; any credentials are accepted by default)* |
| Password | *(empty, or anything)* |
| TLS      | none        |

Authentication can be set to **Accept any** (default), **Disabled** or
**Required** (specific username and password) in Configuration.

Laravel `.env`:

```env
MAIL_MAILER=smtp
MAIL_HOST=127.0.0.1
MAIL_PORT=1025
MAIL_USERNAME=null
MAIL_PASSWORD=null
MAIL_ENCRYPTION=null
```

## Where MKMailLab keeps data

| Mode                                   | Location                          |
|----------------------------------------|-----------------------------------|
| Installed                              | `%LOCALAPPDATA%\MKMailLab\`       |
| Development build (`wails3 dev`)       | `%LOCALAPPDATA%\MKMailLab-Dev\`   |
| Portable (`mkmaillab.portable` beside the exe) | `.\data\` next to the exe |
| Custom                                 | `mkmaillab.exe --data-dir <path>` |

Inside: `mkmaillab.db` (SQLite), `logs\mkmaillab.log` (rotated JSON logs),
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
