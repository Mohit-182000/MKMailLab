package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Component is a long-lived backend part (database, SMTP listeners, retention
// worker, …) with an explicit start/stop lifecycle.
type Component interface {
	Name() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// Lifecycle starts components in registration order and stops them in
// reverse order. If a component fails to start, the ones already started are
// stopped before the error is returned, so a failed startup never leaks
// listeners or open database handles.
type Lifecycle struct {
	log         *slog.Logger
	now         func() time.Time
	stopTimeout time.Duration

	mu         sync.Mutex
	components []Component
	started    []Component
	startedAt  time.Time
	running    bool
}

// NewLifecycle creates an empty lifecycle. stopTimeout bounds each
// component's Stop call individually.
func NewLifecycle(log *slog.Logger, now func() time.Time, stopTimeout time.Duration) *Lifecycle {
	if now == nil {
		now = time.Now
	}
	if stopTimeout <= 0 {
		stopTimeout = 5 * time.Second
	}
	return &Lifecycle{log: log, now: now, stopTimeout: stopTimeout}
}

// Add registers components. It must be called before Start.
func (l *Lifecycle) Add(cs ...Component) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.components = append(l.components, cs...)
}

// Start starts every registered component in order.
func (l *Lifecycle) Start(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.running {
		return errors.New("lifecycle: already started")
	}

	for _, c := range l.components {
		begin := l.now()
		if err := c.Start(ctx); err != nil {
			l.log.Error("component failed to start", "name", c.Name(), "err", err)
			stopErr := l.stopLocked(context.WithoutCancel(ctx))
			return errors.Join(fmt.Errorf("start %s: %w", c.Name(), err), stopErr)
		}
		l.started = append(l.started, c)
		l.log.Debug("component started", "name", c.Name(), "took", l.now().Sub(begin))
	}
	l.startedAt = l.now()
	l.running = true
	l.log.Info("application started", "components", len(l.started))
	return nil
}

// Stop stops every started component in reverse order. It is safe to call
// more than once; subsequent calls are no-ops.
func (l *Lifecycle) Stop(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.started) == 0 {
		return nil
	}
	err := l.stopLocked(ctx)
	l.running = false
	l.log.Info("application stopped")
	return err
}

func (l *Lifecycle) stopLocked(ctx context.Context) error {
	var errs []error
	for i := len(l.started) - 1; i >= 0; i-- {
		c := l.started[i]
		cctx, cancel := context.WithTimeout(ctx, l.stopTimeout)
		err := safeStop(cctx, c)
		cancel()
		if err != nil {
			l.log.Error("component failed to stop", "name", c.Name(), "err", err)
			errs = append(errs, fmt.Errorf("stop %s: %w", c.Name(), err))
		} else {
			l.log.Debug("component stopped", "name", c.Name())
		}
	}
	l.started = nil
	return errors.Join(errs...)
}

// safeStop shields the shutdown sequence from a panicking component so the
// remaining components still get stopped.
func safeStop(ctx context.Context, c Component) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic during stop: %v", r)
		}
	}()
	return c.Stop(ctx)
}

// StartedAt returns when startup completed (zero if not running).
func (l *Lifecycle) StartedAt() time.Time {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.startedAt
}

// Running reports whether Start completed and Stop has not been called.
func (l *Lifecycle) Running() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.running
}

// FuncComponent adapts plain functions to the Component interface.
type FuncComponent struct {
	ComponentName string
	OnStart       func(ctx context.Context) error
	OnStop        func(ctx context.Context) error
}

func (f FuncComponent) Name() string { return f.ComponentName }

func (f FuncComponent) Start(ctx context.Context) error {
	if f.OnStart == nil {
		return nil
	}
	return f.OnStart(ctx)
}

func (f FuncComponent) Stop(ctx context.Context) error {
	if f.OnStop == nil {
		return nil
	}
	return f.OnStop(ctx)
}
