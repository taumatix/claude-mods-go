// Toolcall drives next(e) through a real tool call without a model: a command
// raises a Bash call with $.tool.call, and a tool.call hook rewrites it.
package main

import (
	"encoding/json"
	"fmt"

	"github.com/taumatix/claude-mods-go/mods"
)

func main() {
	app := mods.New("toolcall").Uses(mods.CommandRegister, mods.ToolCall)

	app.OnSessionStart(func(c *mods.Ctx, e mods.SessionStartEvent) (any, error) {
		if err := c.Command.Register(mods.CommandSpec{Name: "rewrite", Description: "run Bash through tool.call"}); err != nil {
			return nil, err
		}
		return c.Pass()
	})

	app.OnCommand("rewrite", func(c *mods.Ctx, e mods.CommandRunEvent) (any, error) {
		raw, err := c.Call(mods.ToolCall, map[string]any{"tool": "Bash", "command": "echo original"})
		if err != nil {
			return nil, err
		}
		var res struct {
			Text string `json:"text"`
		}
		_ = json.Unmarshal(raw, &res)
		return mods.Text(fmt.Sprintf("bash said: %s | raw: %s", res.Text, raw)), nil
	})

	app.OnToolCall("Bash", func(c *mods.Ctx, e mods.ToolCallEvent) (any, error) {
		return c.Next(map[string]any{"tool": e.Tool, "tool_use_id": e.ToolUseID, "command": "echo rewritten"})
	})
	app.Run()
}
