package core_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/gopherex/xlog/pkg/core"
	"github.com/gopherex/xlog/pkg/field"
)

func TestAtomicLevel(t *testing.T) {
	level := core.NewAtomicLevel(core.InfoLevel)
	if level.Enabled(core.DebugLevel) {
		t.Fatal("debug should be disabled")
	}
	level.Set(core.DebugLevel)
	if !level.Enabled(core.DebugLevel) {
		t.Fatal("debug should be enabled")
	}
}

func TestFilterCore(t *testing.T) {
	level := core.NewAtomicLevel(core.WarnLevel)
	base := &recordCore{enabled: true}
	c := core.NewFilterCore(base, level)

	if err := c.Write(core.Event{Level: core.InfoLevel, Message: "ignored"}); err != nil {
		t.Fatalf("write: %v", err)
	}
	if len(base.events) != 0 {
		t.Fatalf("events = %#v", base.events)
	}
}

func TestTeeCore(t *testing.T) {
	a := &recordCore{enabled: true}
	b := &recordCore{enabled: true}
	c := core.NewTeeCore(a, b)

	if err := c.Write(core.Event{Level: core.InfoLevel, Message: "fanout"}); err != nil {
		t.Fatalf("write: %v", err)
	}
	if len(a.events) != 1 || len(b.events) != 1 {
		t.Fatalf("counts = %d %d", len(a.events), len(b.events))
	}
}

func TestSamplerCore(t *testing.T) {
	base := &recordCore{enabled: true}
	c := core.NewSamplerCore(base, 2, 3)
	for i := 0; i < 8; i++ {
		if err := c.Write(core.Event{Level: core.InfoLevel, Message: "same"}); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	if len(base.events) != 4 {
		t.Fatalf("sampled count = %d", len(base.events))
	}
}

func TestAsyncCoreSync(t *testing.T) {
	base := &recordCore{enabled: true}
	c := core.NewAsyncCore(base, 4, core.AsyncBlock, nil)
	for i := 0; i < 3; i++ {
		if err := c.Write(core.Event{Level: core.InfoLevel, Message: "msg"}); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	if err := c.Sync(); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if len(base.events) != 3 {
		t.Fatalf("events = %d", len(base.events))
	}
}

func TestAsyncCoreDropNewest(t *testing.T) {
	base := &slowCore{recordCore: recordCore{enabled: true}}
	obs := &observer{}
	c := core.NewAsyncCore(base, 1, core.AsyncDropNewest, obs)

	drops := 0
	for _, msg := range []string{"1", "2", "3"} {
		if err := c.Write(core.Event{Level: core.InfoLevel, Message: msg}); err != nil {
			drops++
		}
	}
	if err := c.Sync(); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if drops == 0 || obs.drops == 0 {
		t.Fatalf("drops=%d observed=%d", drops, obs.drops)
	}
}

func TestHookCore(t *testing.T) {
	base := &recordCore{enabled: true, err: errors.New("write failed")}
	obs := &observer{}
	c := core.NewHookCore(base, obs)
	err := c.Write(core.Event{Level: core.InfoLevel, Message: "msg"})
	if err == nil {
		t.Fatal("expected error")
	}
	if obs.writes != 1 {
		t.Fatalf("writes = %d", obs.writes)
	}
}

type recordCore struct {
	mu      sync.Mutex
	events  []core.Event
	enabled bool
	err     error
}

func (c *recordCore) Enabled(core.Level) bool { return c.enabled }
func (c *recordCore) Write(event core.Event) error {
	c.mu.Lock()
	c.events = append(c.events, event)
	c.mu.Unlock()
	return c.err
}
func (c *recordCore) With(fields []field.Field) core.Core {
	return c
}
func (c *recordCore) Sync() error { return nil }

type slowCore struct {
	recordCore
}

func (c *slowCore) Write(event core.Event) error {
	time.Sleep(10 * time.Millisecond)
	return c.recordCore.Write(event)
}

type observer struct {
	mu     sync.Mutex
	writes int
	drops  int
}

func (o *observer) OnWrite(core.Event, error) { o.writes++ }
func (o *observer) OnDrop(core.Event)         { o.drops++ }
