// Tally counts tool calls and reports the count with /tally.
package main

import (
	"fmt"

	"github.com/taumatix/claude-mods-go/mods"
)

func main() {
	app := mods.New("tally").
		Describe("Counts tool calls (a Go mod)").
		Uses(mods.CommandRegister, mods.StoreGet, mods.StoreSet)

	app.OnSessionStart(func(c *mods.Ctx, e mods.SessionStartEvent) (any, error) {
		if err := c.Command.Register(mods.CommandSpec{Name: "tally", Description: "Show how many tool calls ran"}); err != nil {
			return nil, err
		}
		return c.Pass()
	})

	app.OnToolCall("", func(c *mods.Ctx, e mods.ToolCallEvent) (any, error) {
		var n int
		if _, err := c.Store.Get("calls", &n); err != nil {
			return nil, err
		}
		if err := c.Store.Set("calls", n+1); err != nil {
			return nil, err
		}
		return c.Pass()
	})

	app.OnCommand("tally", func(c *mods.Ctx, e mods.CommandRunEvent) (any, error) {
		var n int
		if _, err := c.Store.Get("calls", &n); err != nil {
			return nil, err
		}
		return mods.Text(fmt.Sprintf("%d tool calls", n)), nil
	})

	app.Run()
}
