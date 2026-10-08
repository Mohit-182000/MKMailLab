// Package events defines application event names and a late-bound emitter.
//
// Backend services publish through Emitter without knowing about Wails. The
// composition root attaches the Wails event manager once the application
// exists; until then events are dropped (the UI fetches state on load).
package events

import "sync"

// Event names delivered to the frontend.
const (
	SMTPStatus   = "smtp:status"   // payload: smtpd.Status
	MailReceived = "mail:received" // payload: domain.MessageSummary
	MailChanged  = "mail:changed"  // payload: mailbox.Change
)

// Emitter publishes an event with a payload.
type Emitter interface {
	Emit(name string, data any)
}

// Bus is an Emitter whose destination is attached after construction.
type Bus struct {
	mu   sync.RWMutex
	sink func(name string, data any)
}

// Attach sets the destination. Passing nil detaches it.
func (b *Bus) Attach(sink func(name string, data any)) {
	b.mu.Lock()
	b.sink = sink
	b.mu.Unlock()
}

// Emit forwards the event to the attached sink, if any.
func (b *Bus) Emit(name string, data any) {
	b.mu.RLock()
	sink := b.sink
	b.mu.RUnlock()
	if sink != nil {
		sink(name, data)
	}
}

// Recorder is an Emitter that records events; for tests.
type Recorder struct {
	mu     sync.Mutex
	Events []Recorded
}

// Recorded is one captured event.
type Recorded struct {
	Name string
	Data any
}

// Emit records the event.
func (r *Recorder) Emit(name string, data any) {
	r.mu.Lock()
	r.Events = append(r.Events, Recorded{name, data})
	r.mu.Unlock()
}

// Named returns recorded events with the given name.
func (r *Recorder) Named(name string) []Recorded {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []Recorded
	for _, e := range r.Events {
		if e.Name == name {
			out = append(out, e)
		}
	}
	return out
}
