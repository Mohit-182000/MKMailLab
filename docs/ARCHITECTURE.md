# Architecture

LocalMail is a single Windows process: a Go backend and a Vue 3 UI rendered
by WebView2 through Wails v3. There is no internal HTTP server. The UI calls
Go through Wails bindings and receives Wails events. Email HTML and
attachments are served by an in-process asset handler.

## Layers

```
┌──────────────────────────── localmail.exe ────────────────────────────┐
│ WebView2 · Vue 3                                                       │
│   views → Pinia stores → services/backend.ts → bindings / events       │
│ ───────────────────────────────────────────────────────────────────── │
│ bridge/      Wails-bound API structs (validate, map DTOs; no logic)    │
│ services/    mailbox · projects · search · settings · export ·         │
│              retention · diagnostics · testmail · sampledata           │
│ ingest/      raw → mime.Parse → route → repo.Save (tx) → bus.Publish   │
│ smtp/        ListenerManager → listeners (main + per-project ports)    │
│ events/      typed in-process bus → Wails bridge, tray, notifier       │
│ storage/     SQLite: single writer + reader pool, migrations, repos    │
│ assethandler/ sandboxed email HTML, cid: images, attachment streaming  │
│ platform/    tray, toast, autostart, DPAPI, shell-open, dialogs        │
│ app/         composition root + lifecycle                              │
│ logging/ config/ brand/                                                │
└────────────────────────────────────────────────────────────────────────┘
```

**Rules**

1. `internal/app` constructs every long-lived object and wires dependencies
   explicitly. There is no global mutable state.
2. Interfaces are declared by their consumers (e.g. `ingest` declares the
   `MessageStore` it needs), which keeps packages decoupled and easy to fake.
3. Bridge types are thin. Business logic lives in services and is unit-tested
   without Wails.
4. The event bus is the extension point. Webhooks, forwarding, rules and a
   REST API can subscribe to it or call services without changing the core.

## Key technology decisions

| Area | Choice | Reason |
|---|---|---|
| Desktop | Wails v3 (beta.28) | Native tray, notifications, single-instance, close-to-tray, service lifecycle, and `http.Handler` services mounted on the asset server |
| SQLite | `modernc.org/sqlite` (pure Go) | No CGO or C toolchain; FTS5 support |
| SMTP | `emersion/go-smtp` | Battle-tested protocol handling (pipelining, dot-stuffing, AUTH, STARTTLS, BDAT) |
| MIME | `jhillyerd/enmime` + `x/text` | Lenient with malformed mail; collects errors instead of failing; legacy charsets |
| HTML safety | `x/net/html` rewrite + CSP + sandboxed iframe | Keeps rendering fidelity while blocking scripts |
| Logging | `log/slog` | Stdlib; fan-out to file, ring buffer, DB |
| UI | Vue 3, TS, Pinia, Tailwind v4, reka-ui, @tanstack/vue-virtual, CodeMirror 6 (lazy) | |

Raw messages are stored **inside SQLite** (an `email_raw` table), not as
loose `.eml` files. This keeps inserts atomic, leaves no orphaned files, makes
retention a single `DELETE` and backup a single file copy. Remote images in
previews load by default (the mail comes from your own apps), and a setting
and per-message toggle can block them.

## Lifecycle

```
main ─► config.ResolveFromEnvironment ─► paths.EnsureDirs
     ─► logging.New (file + ring [+ stderr in dev])
     ─► app.New  (constructs graph, registers components)
     ─► application.New(Services: [lifecycle, bridge services…])
     ─► Run()
           ServiceStartup(lifecycle) → components Start in order
           …
           ServiceShutdown(lifecycle) → components Stop in reverse (10 s budget)
```

- If a component fails to start, the components already started are stopped
  before the error surfaces. No listener or database handle leaks.
- Each component's `Stop` has its own timeout. A panicking `Stop` is
  recovered so later components still shut down.
- Wails cancels its startup context just before shutdown. Components receive
  a detached context and are stopped explicitly instead.
- Errors that happen before the UI exists (unwritable data directory, log
  file) show a native message box. A top-level panic is logged with its stack
  and reported the same way.

Planned component order (start → stop reversed):
`tempdir → storage → settings → event bus → ingest → smtp listeners → retention → notifier/tray`.

## Data flow: receiving an email (Phases 4–8)

```
SMTP client ─► listener (limits, timeouts, AUTH)
            ─► DATA (size-capped stream)
            ─► ingest queue (bounded; back-pressure → 451)
            ─► mime.Parse (panic-safe; errors recorded, raw always kept)
            ─► route to project (listener port / X-LocalMail-Project / AUTH user)
            ─► single SQLite transaction (emails, bodies, raw, recipients,
               headers, attachments, FTS)
            ─► "250 OK: queued as <id>"   (only after commit)
            ─► bus.Publish(EmailReceived)
            ─► Wails bridge (200 ms coalescing) ─► Vue store inserts row
```

## Data directory

Resolved by `internal/config`:

1. `--data-dir` override
2. Portable mode, enabled by `localmail.portable` next to the exe → `.\data`
3. `%LOCALAPPDATA%\LocalMail` (`LocalMail-Dev` for dev builds)

## Logging

`internal/logging` creates one `slog.Logger` that fans out to:

- `logs\localmail.log`: JSON lines, rotated at 10 MiB, 5 backups kept. A
  record is never split across files.
- an in-memory ring of 2,000 entries with live subscribers, which feeds the
  Diagnostics page
- stderr as text, in dev builds only
- *(Phase 3+)* a DB sink for warnings, errors and lifecycle events

Attribute keys that look sensitive (`password`, `token`, `secret`,
`authorization`, `cookie`, …) are redacted in every sink. The Wails framework
logger is filtered to WARN and above.

The level is a `slog.LevelVar`, so it can be changed at runtime from settings.

## Frontend architecture

- `services/backend.ts` defines the `Backend` interface, the only gateway to
  Go. The Wails implementation is loaded lazily, and tests inject fakes with
  `setBackend()`.
- Pinia stores own the state. Components render state and dispatch actions.
- Generated DTO types are re-exported from `src/types` under stable names.
- *(Phase 2)* Every sidebar folder (Inbox, Unread, Starred, project, saved
  search) is the same mailbox view with a preset query. View state lives in
  the URL.

## Rebranding

Edit `internal/brand/brand.go` and `build/config.yml`, then run
`wails3 task common:update:build-assets` and replace `build/appicon.png`.
