package mods

import (
	"encoding/json"
	"fmt"
)

// Typed events carry the fields most mods read. Fields Claude Code adds later
// are ignored; use App.On with the raw JSON (or Ctx.Raw) to see them.

// SessionStartEvent is the session.start event.
type SessionStartEvent struct {
	Cwd           string `json:"cwd"`
	IsInteractive bool   `json:"isInteractive"`
}

// CommandRunEvent is the command.run event.
type CommandRunEvent struct {
	Command string `json:"command"`
	Args    string `json:"args"`
}

// ToolCallEvent is the tool.call event. Input holds the tool's own arguments
// (command for Bash, file_path for Read) as Claude Code flattens them.
type ToolCallEvent struct {
	Tool      string          `json:"tool"`
	ToolUseID string          `json:"tool_use_id"`
	AgentID   string          `json:"agentId,omitempty"`
	Input     json.RawMessage `json:"-"`
}

// PromptSubmitEvent is the prompt.submit event.
type PromptSubmitEvent struct {
	Text string `json:"text"`
}

// ToolCheckEvent is the tool.check event.
type ToolCheckEvent struct {
	Tool  string          `json:"tool"`
	Input json.RawMessage `json:"input"`
}

// TurnCompleteEvent is the turn.complete event.
type TurnCompleteEvent struct {
	Reason     string `json:"reason"`
	DurationMs int    `json:"durationMs"`
}

// Text answers command.run with a transcript line.
func Text(s string) map[string]any { return map[string]any{"text": s} }

// Deny answers tool.call with a refusal the model reads as an error.
func Deny(reason string) map[string]any { return map[string]any{"deny": reason} }

// Allow, Ask and DenyCheck answer tool.check.
func Allow(reason string) map[string]any { return decision("allow", reason) }
func Ask(reason string) map[string]any   { return decision("ask", reason) }
func DenyCheck(reason string) map[string]any {
	return decision("deny", reason)
}

func decision(d, reason string) map[string]any {
	m := map[string]any{"decision": d}
	if reason != "" {
		m["reason"] = reason
	}
	return m
}

func typed[E any](h func(*Ctx, E) (any, error)) Handler {
	return func(c *Ctx, raw json.RawMessage) (any, error) {
		var e E
		if err := json.Unmarshal(raw, &e); err != nil {
			return nil, fmt.Errorf("mods: decoding event: %w", err)
		}
		if tc, ok := any(&e).(*ToolCallEvent); ok {
			var all map[string]json.RawMessage
			if err := json.Unmarshal(raw, &all); err == nil {
				for _, k := range []string{"tool", "tool_use_id", "agentId"} {
					delete(all, k)
				}
				tc.Input, _ = json.Marshal(all)
			}
		}
		return h(c, e)
	}
}

func (a *App) OnSessionStart(h func(*Ctx, SessionStartEvent) (any, error)) *App {
	return a.On("session.start", typed(h))
}

// OnCommand handles a slash command registered by this mod (or any other).
func (a *App) OnCommand(name string, h func(*Ctx, CommandRunEvent) (any, error)) *App {
	return a.On("command.run", typed(h), Where("command", name))
}

// OnToolCall handles tool.call; tool narrows it to one tool ("" for all).
func (a *App) OnToolCall(tool string, h func(*Ctx, ToolCallEvent) (any, error)) *App {
	if tool == "" {
		return a.On("tool.call", typed(h))
	}
	return a.On("tool.call", typed(h), Where("tool", tool))
}

func (a *App) OnToolCheck(tool string, h func(*Ctx, ToolCheckEvent) (any, error)) *App {
	if tool == "" {
		return a.On("tool.check", typed(h))
	}
	return a.On("tool.check", typed(h), Where("tool", tool))
}

func (a *App) OnPromptSubmit(h func(*Ctx, PromptSubmitEvent) (any, error)) *App {
	return a.On("prompt.submit", typed(h))
}

func (a *App) OnTurnComplete(h func(*Ctx, TurnCompleteEvent) (any, error)) *App {
	return a.On("turn.complete", typed(h))
}
