// Package app is the composition root: it constructs every backend
// component, wires dependencies explicitly and exposes the Wails services.
// No other package should build long-lived objects on its own.
package app

import (
	"context"
	"log/slog"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"localmail/internal/bridge"
	"localmail/internal/config"
	"localmail/internal/logging"
)

// shutdownBudget bounds the whole backend shutdown so closing the window
// never hangs, even if a component misbehaves.
const shutdownBudget = 10 * time.Second

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

	system *bridge.SystemService
}

// New builds the backend. It performs no I/O beyond construction; resources
// are acquired in Start via the lifecycle.
func New(opts Options) *App {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	log := logging.Component(opts.Logging.Logger, "app")
	lc := NewLifecycle(log, now, 5*time.Second)

	a := &App{
		paths:     opts.Paths,
		log:       log,
		lifecycle: lc,
	}

	// Components are started in this order and stopped in reverse.
	// Later phases add: storage → ingest → smtp → retention.
	lc.Add(FuncComponent{
		ComponentName: "tempdir",
		OnStart: func(context.Context) error {
			opts.Paths.CleanTemp()
			return nil
		},
	})

	a.system = bridge.NewSystemService(opts.Paths, opts.Logging.File, opts.Logging.Ring, lc.StartedAt)
	return a
}

// Services returns the Wails services in registration order. The lifecycle
// service comes first so the backend is up before any bound API is called,
// and Wails shuts services down in reverse order.
func (a *App) Services() []application.Service {
	return []application.Service{
		application.NewServiceWithOptions(&lifecycleService{app: a}, application.ServiceOptions{Name: "lifecycle"}),
		application.NewService(a.system),
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
