package mods

import (
	"encoding/json"
	"fmt"
	"sync"
)

// HostError is a mods API call, or next(e), that threw in JavaScript.
type HostError struct {
	Method  string
	Message string
}

func (e *HostError) Error() string { return fmt.Sprintf("mods: %s: %s", e.Method, e.Message) }

// driver carries one Step to the shim and returns its answer.
type driver interface {
	step(Step) (Resume, error)
}

// Ctx is the handler's way to the rest of Claude Code. It is the Go form of
// the `$` argument and of next.
type Ctx struct {
	UI      UI
	Command Commands
	Tool    Tools
	FS      FS
	Store   Store
	HTTP    HTTP
	Process Process
	Model   Model
	Session Session

	app *App
	drv driver
	run string
	raw json.RawMessage

	mu  sync.Mutex
	cid int
}

func (a *App) newCtx(drv driver, run string, raw json.RawMessage) *Ctx {
	c := &Ctx{app: a, drv: drv, run: run, raw: raw}
	c.UI, c.Command, c.Tool = UI{c}, Commands{c}, Tools{c}
	c.FS, c.Store, c.HTTP = FS{c}, Store{c}, HTTP{c}
	c.Process, c.Model, c.Session = Process{c}, Model{c}, Session{c}
	return c
}

// Raw is the event as Claude Code passed it.
func (c *Ctx) Raw() json.RawMessage { return c.raw }

// Call invokes a mods API method and returns its JSON result. The typed
// helpers (c.FS.Read, c.UI.Toast, ...) are built on it; use it for a method
// that has none. The method must have been declared with App.Uses.
func (c *Ctx) Call(m Method, args ...any) (json.RawMessage, error) {
	if !c.app.uses[m] {
		return nil, fmt.Errorf("mods: %s is not declared; add it with App.Uses", m)
	}
	raw := make([]json.RawMessage, len(args))
	for i, a := range args {
		b, err := json.Marshal(a)
		if err != nil {
			return nil, fmt.Errorf("mods: %s argument %d: %w", m, i, err)
		}
		raw[i] = b
	}
	return c.roundtrip(Step{Op: OpCall, Method: string(m), Args: raw})
}

// Next calls next(e): the hooks after this one, then Claude Code's own
// behaviour. It returns the event's result as JSON. Pass a changed copy of the
// event to rewrite it.
func (c *Ctx) Next(e any) (json.RawMessage, error) {
	b, err := json.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("mods: next: %w", err)
	}
	return c.roundtrip(Step{Op: OpNext, Method: "next", Event: b})
}

// Pass is next(e) with the event unchanged.
func (c *Ctx) Pass() (json.RawMessage, error) { return c.Next(c.raw) }

func (c *Ctx) roundtrip(s Step) (json.RawMessage, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cid++
	s.CID, s.Run = c.cid, c.run
	r, err := c.drv.step(s)
	if err != nil {
		return nil, err
	}
	if !r.OK {
		return nil, &HostError{Method: s.Method, Message: r.Error}
	}
	return r.Value, nil
}

func decode[T any](raw json.RawMessage, err error) (T, error) {
	var v T
	if err != nil {
		return v, err
	}
	if len(raw) == 0 || string(raw) == "null" {
		return v, nil
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return v, fmt.Errorf("mods: decoding result: %w", err)
	}
	return v, nil
}
