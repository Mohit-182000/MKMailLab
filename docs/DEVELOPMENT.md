# Development

## Prerequisites

| Tool        | Version              | Notes |
|-------------|----------------------|-------|
| Go          | 1.26+ (tested 1.27.1)| `go version` |
| Wails CLI   | v3.0.0-beta.28       | `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.28` |
| Node.js     | 24.x (24.15+ recommended) | Only for building the frontend; not needed by end users |
| WebView2    | any evergreen        | Preinstalled on Windows 11 |
| NSIS        | 3.x                  | Only for building the installer (Phase 14) |

No C compiler is needed: the build is `CGO_ENABLED=0` (the SQLite driver is
pure Go).

Verify with `wails3 doctor`.

> The Go installer adds `C:\Program Files\Go\bin` and `%USERPROFILE%\go\bin`
> to PATH, but terminals opened before installation won't see them — open a
> new terminal (or restart VS Code) after installing Go.

## Common tasks

Run from the repository root. `wails3 task <name>` runs tasks from
`Taskfile.yml`.

| Command                     | What it does |
|-----------------------------|--------------|
| `wails3 dev`                | Hot-reloading dev build (Go rebuilds on save; Vite HMR for the UI) |
| `wails3 build`              | Production `bin\mkmaillab.exe` |
| `wails3 task test`          | All Go + frontend tests |
| `wails3 task test:go`       | Go tests only (`go test ./internal/...`) |
| `wails3 task test:frontend` | Vitest unit tests |
| `wails3 task lint`          | gofmt, go vet, vue-tsc |
| `wails3 task bindings`      | Regenerate TypeScript bindings after changing Go service signatures |
| `wails3 task common:generate:icons` | Rebuild `build/windows/icon.ico` from `build/appicon.png` |

Frontend-only (inside `frontend/`): `npm run typecheck`, `npm test`,
`npm run test:watch`.

## Dev vs production builds

| | Dev (`wails3 dev`) | Production (`wails3 build`) |
|-|-|-|
| Build tag | none → `brand.Dev = true` | `production` → `brand.Dev = false` |
| Data dir | `%LOCALAPPDATA%\MKMailLab-Dev` | `%LOCALAPPDATA%\MKMailLab` |
| Single-instance ID | `dev.mkmaillab.app.dev` | `dev.mkmaillab.app` |
| Log level | debug, also to stderr | info, file only |
| DevTools | enabled (F12) | disabled |

A dev build can therefore run beside an installed copy without touching its
mailbox (SMTP port conflicts are handled in Phase 4).

## Project layout

```
main.go                 thin entry point: flags, paths, logging, Wails app
internal/
  app/                  composition root + component lifecycle
  brand/                product identity (rename here) + build mode
  bridge/               Wails-bound API surface (thin DTO layer)
  config/               data-directory resolution (AppData / portable)
  logging/              slog fan-out: rotating file, ring buffer, redaction
  platform/             OS integration (Windows impl + inert fallbacks)
frontend/
  bindings/             GENERATED from Go — do not edit
  src/
    services/backend.ts the only gateway to Go (swappable for tests)
    stores/             Pinia stores
    views/ components/ layouts/ composables/ types/ utils/ styles/
build/                  Wails build config, Windows manifest, NSIS
docs/
```

## Conventions

- **Backend calls go through `services/backend.ts`.** Stores receive a
  `Backend`; components never import from `bindings/`. Tests call
  `setBackend(fake)`.
- **Business logic lives in Go services**, not in `bridge/` and not in Vue
  components.
- **Every long-lived Go object is constructed in `internal/app`.** Components
  with resources implement `app.Component` (`Start`/`Stop`) and are registered
  with the lifecycle in dependency order.
- **Log with a component tag:** `logging.Component(log, "smtp")`. Never log
  passwords or message bodies; keys containing `password`, `token`, `secret`,
  etc. are redacted automatically as a safety net, not as a licence.
- **Contexts:** accept `context.Context` as the first parameter for anything
  that does I/O or may block.

## Troubleshooting the toolchain

- **`wails3: command not found`** — open a new terminal so PATH includes
  `%USERPROFILE%\go\bin`.
- **"Looks like npm isn't installed" during `wails3 build`** — this message
  appears when a *parallel* build step fails and cancels the npm check. Scroll
  up for the first `ERROR`.
- **`EBADENGINE` warnings for `abbrev`/`nopt`** — harmless; they come from a
  CLI helper inside `@vue/test-utils`. Upgrading Node to 24.15+ silences them.
- **Data from a previous run interferes** — delete
  `%LOCALAPPDATA%\MKMailLab-Dev` (dev) or run with `--data-dir`.
