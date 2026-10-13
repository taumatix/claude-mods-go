// Package mods lets a Go program act as a Claude Code mod.
//
// A mod's hooks module is JavaScript that Claude Code loads and calls. This
// package is the Go half of a bridge: the modgo tool generates a small
// JavaScript shim that forwards every event to your program, and your handlers
// answer through the same protocol, calling mods API methods ($.fs.read, ...)
// and next(e) by asking the shim to do it.
//
//	app := mods.New("tally").Uses(mods.CommandRegister)
//	app.OnSessionStart(func(c *mods.Ctx, e mods.SessionStartEvent) (any, error) {
//		if err := c.Command.Register(mods.CommandSpec{Name: "tally", Description: "Show the count"}); err != nil {
//			return nil, err
//		}
//		return c.Pass()
//	})
//	app.OnCommand("tally", func(c *mods.Ctx, e mods.CommandRunEvent) (any, error) {
//		return mods.Text("hello"), nil
//	})
//	func main() { app.Run() }
package mods

import (
	"encoding/json"
	"fmt"
	"sort"
)

// Handler handles one event. e is the event exactly as Claude Code passed it
// to the hook. The value it returns is what the hook returns: use c.Next or
// c.Pass to let Claude Code carry on, or return an answer of your own.
type Handler func(c *Ctx, e json.RawMessage) (any, error)

type registration struct {
	hook Hook
	h    Handler
}

// App is a mod: its hooks and the mods API methods it may call.
type App struct {
	name, version, description string
	regs                       []registration
	uses                       map[Method]bool
}

// New starts a mod. name becomes the plugin name; Claude Code allows letters,
// digits and hyphens, and refuses a name that begins with "claude-".
func New(name string) *App {
	return &App{name: name, uses: map[Method]bool{}}
}

// Version sets the plugin version written to plugin.json.
func (a *App) Version(v string) *App { a.version = v; return a }

// Describe sets the plugin description written to plugin.json.
func (a *App) Describe(d string) *App { a.description = d; return a }

// Uses declares the mods API methods the handlers call. The generated shim
// bridges exactly these, so `claude plugin validate` and an administrator's
// plugin.register policy see the mod's real reach. A call to a method that was
// not declared fails with an error naming it.
func (a *App) Uses(methods ...Method) *App {
	for _, m := range methods {
		a.uses[m] = true
	}
	return a
}

// Option narrows a hook with a matcher, as the second argument to on() does.
type Option func(*Hook)

// Where adds a matcher field: Where("tool", "Bash") hooks only Bash calls.
func Where(key, value string) Option {
	return func(h *Hook) {
		if h.Match == nil {
			h.Match = map[string]string{}
		}
		h.Match[key] = value
	}
}

// On registers a handler for any event by name. The typed OnXxx methods are
// easier for the events they cover.
func (a *App) On(event string, h Handler, opts ...Option) *App {
	hook := Hook{Event: event}
	for _, o := range opts {
		o(&hook)
	}
	a.regs = append(a.regs, registration{hook: hook, h: h})
	return a
}

// Manifest returns what the generator needs. It reports an error for a method
// that cannot be bridged.
func (a *App) Manifest() (Manifest, error) {
	m := Manifest{Name: a.name, Version: a.version, Description: a.description, Hooks: []Hook{}, Uses: []Method{}}
	for _, r := range a.regs {
		m.Hooks = append(m.Hooks, r.hook)
	}
	for u := range a.uses {
		if !IsKnown(u) {
			return Manifest{}, fmt.Errorf("mods: %q cannot be bridged (see mods.KnownMethods)", u)
		}
		m.Uses = append(m.Uses, u)
	}
	sort.Slice(m.Uses, func(i, j int) bool { return m.Uses[i] < m.Uses[j] })
	if len(m.Hooks) == 0 {
		return Manifest{}, fmt.Errorf("mods: %s registers no hooks", a.name)
	}
	return m, nil
}
