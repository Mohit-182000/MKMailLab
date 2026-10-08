package app

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"localmail/internal/logging"
)

type recorder struct{ events []string }

func (r *recorder) comp(name string, startErr error) Component {
	return FuncComponent{
		ComponentName: name,
		OnStart: func(context.Context) error {
			r.events = append(r.events, "start:"+name)
			return startErr
		},
		OnStop: func(context.Context) error {
			r.events = append(r.events, "stop:"+name)
			return nil
		},
	}
}

func TestLifecycleOrder(t *testing.T) {
	rec := &recorder{}
	lc := NewLifecycle(logging.Discard(), nil, time.Second)
	lc.Add(rec.comp("db", nil), rec.comp("ingest", nil), rec.comp("smtp", nil))

	if err := lc.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !lc.Running() || lc.StartedAt().IsZero() {
		t.Fatal("lifecycle should be running")
	}
	if err := lc.Start(context.Background()); err == nil {
		t.Fatal("second Start should fail")
	}
	if err := lc.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := lc.Stop(context.Background()); err != nil {
		t.Fatal("second Stop should be a no-op")
	}

	want := []string{"start:db", "start:ingest", "start:smtp", "stop:smtp", "stop:ingest", "stop:db"}
	if !reflect.DeepEqual(rec.events, want) {
		t.Fatalf("events = %v, want %v", rec.events, want)
	}
}

func TestLifecycleRollsBackOnStartFailure(t *testing.T) {
	rec := &recorder{}
	boom := errors.New("port in use")
	lc := NewLifecycle(logging.Discard(), nil, time.Second)
	lc.Add(rec.comp("db", nil), rec.comp("smtp", boom), rec.comp("never", nil))

	err := lc.Start(context.Background())
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want wrapping %v", err, boom)
	}
	want := []string{"start:db", "start:smtp", "stop:db"}
	if !reflect.DeepEqual(rec.events, want) {
		t.Fatalf("events = %v, want %v", rec.events, want)
	}
	if lc.Running() {
		t.Fatal("lifecycle must not be running after failed start")
	}
}

func TestLifecycleStopContinuesAfterPanicAndTimeout(t *testing.T) {
	var stopped []string
	lc := NewLifecycle(logging.Discard(), nil, 50*time.Millisecond)
	lc.Add(
		FuncComponent{ComponentName: "first", OnStop: func(context.Context) error {
			stopped = append(stopped, "first")
			return nil
		}},
		FuncComponent{ComponentName: "slow", OnStop: func(ctx context.Context) error {
			<-ctx.Done() // respects the per-component timeout
			return ctx.Err()
		}},
		FuncComponent{ComponentName: "panicky", OnStop: func(context.Context) error { panic("bad") }},
	)
	if err := lc.Start(context.Background()); err != nil {
		t.Fatal(err)
	}

	err := lc.Stop(context.Background())
	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected joined error including deadline, got %v", err)
	}
	if len(stopped) != 1 {
		t.Fatal("first component was not stopped after earlier failures")
	}
}
