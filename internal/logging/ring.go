package logging

import (
	"context"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"
)

// Entry is a log record flattened for display in the diagnostics UI.
type Entry struct {
	Seq       uint64         `json:"seq"`
	Time      time.Time      `json:"time"`
	Level     string         `json:"level"`
	Component string         `json:"component,omitempty"`
	Message   string         `json:"message"`
	Attrs     map[string]any `json:"attrs,omitempty"`
}

// Ring keeps the most recent log entries in memory and fans them out to live
// subscribers. It is the data source for the diagnostics page's live tail.
type Ring struct {
	mu      sync.Mutex
	buf     []Entry
	next    int
	full    bool
	seq     uint64
	subs    map[int]chan Entry
	nextSub int
}

// NewRing creates a ring holding up to capacity entries.
func NewRing(capacity int) *Ring {
	if capacity < 1 {
		capacity = 1
	}
	return &Ring{buf: make([]Entry, capacity), subs: map[int]chan Entry{}}
}

func (r *Ring) add(e Entry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	e.Seq = r.seq
	r.buf[r.next] = e
	r.next = (r.next + 1) % len(r.buf)
	if r.next == 0 {
		r.full = true
	}
	for _, ch := range r.subs {
		select {
		case ch <- e:
		default: // slow subscriber: drop rather than block logging
		}
	}
}

// Snapshot returns the buffered entries, oldest first.
func (r *Ring) Snapshot() []Entry {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.full {
		return slices.Clone(r.buf[:r.next])
	}
	out := make([]Entry, 0, len(r.buf))
	out = append(out, r.buf[r.next:]...)
	return append(out, r.buf[:r.next]...)
}

// Subscribe returns a channel receiving every new entry and a cancel function
// that must be called to release it.
func (r *Ring) Subscribe(buffer int) (<-chan Entry, func()) {
	ch := make(chan Entry, buffer)
	r.mu.Lock()
	id := r.nextSub
	r.nextSub++
	r.subs[id] = ch
	r.mu.Unlock()

	var once sync.Once
	return ch, func() {
		once.Do(func() {
			r.mu.Lock()
			delete(r.subs, id)
			r.mu.Unlock()
			close(ch)
		})
	}
}

// Clear drops all buffered entries.
func (r *Ring) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	clear(r.buf)
	r.next, r.full = 0, false
}

// Handler returns a slog.Handler writing into the ring at or above level.
func (r *Ring) Handler(level slog.Leveler) slog.Handler {
	return &ringHandler{ring: r, level: level}
}

type ringHandler struct {
	ring   *Ring
	level  slog.Leveler
	attrs  []slog.Attr // pre-bound attrs, keys already group-qualified
	prefix string      // current group prefix, e.g. "smtp.session."
}

func (h *ringHandler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.level.Level()
}

func (h *ringHandler) Handle(_ context.Context, rec slog.Record) error {
	e := Entry{Time: rec.Time, Level: rec.Level.String(), Message: rec.Message}
	attrs := make(map[string]any, len(h.attrs)+rec.NumAttrs())
	for _, a := range h.attrs {
		putAttr(attrs, "", a, &e)
	}
	rec.Attrs(func(a slog.Attr) bool {
		putAttr(attrs, h.prefix, a, &e)
		return true
	})
	if len(attrs) > 0 {
		e.Attrs = attrs
	}
	h.ring.add(e)
	return nil
}

func putAttr(dst map[string]any, prefix string, a slog.Attr, e *Entry) {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return
	}
	if a.Value.Kind() == slog.KindGroup {
		p := prefix
		if a.Key != "" {
			p = prefix + a.Key + "."
		}
		for _, ga := range a.Value.Group() {
			putAttr(dst, p, ga, e)
		}
		return
	}
	a = redactAttr(nil, a)
	key := prefix + a.Key
	if key == ComponentKey {
		e.Component = a.Value.String()
		return
	}
	dst[key] = a.Value.Any()
}

func (h *ringHandler) WithAttrs(as []slog.Attr) slog.Handler {
	nh := *h
	nh.attrs = slices.Clone(h.attrs)
	for _, a := range as {
		if h.prefix != "" {
			a.Key = h.prefix + a.Key
		}
		nh.attrs = append(nh.attrs, a)
	}
	return &nh
}

func (h *ringHandler) WithGroup(name string) slog.Handler {
	if strings.TrimSpace(name) == "" {
		return h
	}
	nh := *h
	nh.prefix = h.prefix + name + "."
	return &nh
}
