// Package app is the composition root: it constructs every backend
// component, wires dependencies explicitly and exposes the Wails services.
// No other package should build long-lived objects on its own.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"localmail/internal/bridge"
	"localmail/internal/config"
	"localmail/internal/events"
	"localmail/internal/ingest"
	"localmail/internal/logging"
	"localmail/internal/mailbox"
	"localmail/internal/mailhttp"
	"localmail/internal/settings"
	"localmail/internal/smtpd"
	"localmail/internal/storage/sqlite"
)

// shutdownBudget bounds the whole backend shutdown so closing the window
// never hangs, even if a component misbehaves.
const shutdownBudget = 10 * time.Second

// ContentRoute is where message content (HTML previews, attachments) is
// mounted on the Wails asset server.
const ContentRoute = "/lm"

// Options are the process-level inputs resolved in main.
type Options struct {
	Paths   config.Paths
	Logging *logging.Logging
	Now     func() time.Time
}

// App owns the backend object graph.
type App struct {
	paths     config.Paths
	log       *slog.Logger
	lifecycle *Lifecycle
	bus       *events.Bus

	db       *sqlite.DB
	settings *settings.Service
	smtp     *smtpd.Server

	system  *bridge.SystemService
	smtpAPI *bridge.SMTPService
	mailAPI *bridge.MailService
	content *mailhttp.Handler
}

// New builds the backend. It opens the database (fast, and failing early
// lets main show a clear error); everything else starts in the lifecycle.
func New(ctx context.Context, opts Options) (*App, error) {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	root := opts.Logging.Logger
	log := logging.Component(root, "app")

	db, err := sqlite.Open(ctx, opts.Paths.Database, logging.Component(root, "storage"))
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	bus := &events.Bus{}
	st := settings.NewService(db, logging.Component(root, "settings"))
	ing := ingest.NewService(db, bus, logging.Component(root, "ingest"), now)
	server := smtpd.New(logging.Component(root, "smtp"), ing, bus)
	mb := mailbox.NewService(db, bus, logging.Component(root, "mailbox"))

	lc := NewLifecycle(log, now, 5*time.Second)
	a := &App{
		paths: opts.Paths, log: log, lifecycle: lc, bus: bus,
		db: db, settings: st, smtp: server,
	}

	// Components start in this order and stop in reverse.
	lc.Add(
		FuncComponent{
			ComponentName: "storage",
			OnStop:        func(context.Context) error { return db.Close() },
		},
		FuncComponent{
			ComponentName: "tempdir",
			OnStart: func(context.Context) error {
				opts.Paths.CleanTemp()
				return nil
			},
		},
		FuncComponent{
			ComponentName: "settings",
			OnStart:       st.Load,
		},
		FuncComponent{
			ComponentName: "smtp",
			OnStart: func(context.Context) error {
				cfg := st.SMTP()
				if !cfg.AutoStart {
					log.Info("smtp autostart disabled")
					return nil
				}
				// A busy port must not prevent the app from opening; the UI
				// shows the error and lets the user pick another port.
				if _, err := server.Start(bridge.SMTPConfigFrom(cfg)); err != nil {
					log.Warn("smtp autostart failed", "err", err)
				}
				return nil
			},
			OnStop: func(ctx context.Context) error {
				_, err := server.Stop(ctx)
				return err
			},
		},
	)

	a.system = bridge.NewSystemService(opts.Paths, opts.Logging.File, opts.Logging.Ring, lc.StartedAt)
	a.smtpAPI = bridge.NewSMTPService(server, st, logging.Component(root, "smtp-api"))
	a.mailAPI = bridge.NewMailService(mb, wailsSaver{})
	a.content = mailhttp.New(mb, logging.Component(root, "content"), ContentRoute)
	return a, nil
}

// AttachEvents connects backend events to the frontend. Call it once the
// Wails application exists.
func (a *App) AttachEvents(sink func(name string, data any)) {
	a.bus.Attach(sink)
}

// Services returns the Wails services in registration order. The lifecycle
// service comes first so the backend is up before any bound API is called,
// and Wails shuts services down in reverse order.
func (a *App) Services() []application.Service {
	return []application.Service{
		application.NewServiceWithOptions(&lifecycleService{app: a}, application.ServiceOptions{Name: "lifecycle"}),
		application.NewService(a.system),
		application.NewService(a.smtpAPI),
		application.NewService(a.mailAPI),
		application.NewServiceWithOptions(a.content, application.ServiceOptions{Name: "content", Route: ContentRoute}),
	}
}

// lifecycleService adapts the Lifecycle to Wails' service hooks. It exposes
// no methods to the frontend.
type lifecycleService struct{ app *App }

func (s *lifecycleService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	// Wails cancels ctx right before shutdown; components must not tie
	// their own lifetime to it, so pass a detached copy.
	return s.app.lifecycle.Start(context.WithoutCancel(ctx))
}

func (s *lifecycleService) ServiceShutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownBudget)
	defer cancel()
	return s.app.lifecycle.Stop(ctx)
}

// wailsSaver shows the native Windows "Save as" dialog.
type wailsSaver struct{}

func (wailsSaver) PromptSavePath(defaultName, filterName, pattern string) (string, error) {
	wa := application.Get()
	if wa == nil {
		return "", fmt.Errorf("application not running")
	}
	d := wa.Dialog.SaveFile().SetFilename(defaultName).AddFilter(filterName, pattern).CanCreateDirectories(true)
	if w := wa.Window.Current(); w != nil {
		d = d.AttachToWindow(w)
	}
	path, err := d.PromptForSingleSelection()
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "cancel") {
		return "", nil // user closed the dialog
	}
	return path, err
}
