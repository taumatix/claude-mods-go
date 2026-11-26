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

// Origin names who provides a tool, command or other offered item.
type Origin struct {
	Plugin string `json:"plugin"`
	Tier   string `json:"tier"`
}

// Origin of a draft, suggestion or setting change: Kind is "engine",
// "suggestion", "composer", "bridge" or "plugin" (then Name says which).
type Source struct {
	Kind string `json:"kind"`
	Name string `json:"name,omitempty"`
}

// SessionEndEvent is the session.end event. Reason is clear, resume, logout,
// prompt_input_exit, other or unload.
type SessionEndEvent struct {
	Reason    string `json:"reason"`
	SessionID string `json:"sessionId"`
	Resume    struct {
		ID string `json:"id"`
	} `json:"resume"`
}

// SessionStopEvent is the session.stop event: the user stopped Claude (Esc).
type SessionStopEvent struct {
	IsTurnRunning bool `json:"isTurnRunning"`
}

// SessionCompactEvent is the session.compact event. Trigger is manual, auto,
// plugin or precompute. Answer with {"skip": reason} to prevent it.
type SessionCompactEvent struct {
	Trigger      string          `json:"trigger"`
	AgentID      string          `json:"agentId,omitempty"`
	Instructions string          `json:"instructions,omitempty"`
	Messages     json.RawMessage `json:"messages"`
}

// TurnStartEvent is the turn.start event.
type TurnStartEvent struct {
	Text    string `json:"text"`
	TurnID  string `json:"turnId"`
	AgentID string `json:"agentId,omitempty"`
}

// PromptFillEvent is the prompt.fill event: text about to go into the prompt
// box as a draft. Mode is replace, append or insert.
type PromptFillEvent struct {
	Text   string `json:"text"`
	Mode   string `json:"mode"`
	Origin Source `json:"origin"`
	Cursor *int   `json:"cursor,omitempty"`
}

// PromptSuggestEvent is the prompt.suggest event: a dim suggestion proposed
// for the prompt box.
type PromptSuggestEvent struct {
	Text   string `json:"text"`
	Origin Source `json:"origin"`
}

// ToolDescribeEvent is the tool.describe event. Answer with
// {"description": ...} to change what Claude reads.
type ToolDescribeEvent struct {
	Tool        string          `json:"tool"`
	Description string          `json:"description"`
	IsDeferred  bool            `json:"isDeferred,omitempty"`
	Provider    Origin          `json:"provider"`
	MCPServer   json.RawMessage `json:"mcpServer,omitempty"`
}

// ToolOfferEvent is the tool.offer event. For is "list" or "call". Answer
// with {"isOffered": false} to withhold the tool from Claude.
type ToolOfferEvent struct {
	Tool         string `json:"tool"`
	Provider     Origin `json:"provider"`
	For          string `json:"for"`
	AgentID      string `json:"agentId,omitempty"`
	SubagentType string `json:"subagentType,omitempty"`
}

// CommandDescribeEvent is the command.describe event, once for each command
// in the command list.
type CommandDescribeEvent struct {
	Command      string `json:"command"`
	Description  string `json:"description"`
	ArgumentHint string `json:"argumentHint,omitempty"`
	IsHidden     bool   `json:"isHidden"`
	Immediate    bool   `json:"immediate"`
	Provider     Origin `json:"provider"`
}

// ConfigSetEvent is the config.set event: a /config row is about to change.
// Value and Previous are a bool, string, number or list of strings, as JSON.
type ConfigSetEvent struct {
	Key      string          `json:"key"`
	Value    json.RawMessage `json:"value"`
	Previous json.RawMessage `json:"previous"`
	Provider Origin          `json:"provider"`
	Origin   Source          `json:"origin"`
}

// AgentSpawnEvent is the agent.spawn event: a subagent or teammate is about
// to start. Answer with {"deny": reason} to refuse it.
type AgentSpawnEvent struct {
	ToolUseID    string `json:"tool_use_id"`
	Prompt       string `json:"prompt"`
	Description  string `json:"description"`
	SubagentType string `json:"subagentType"`
	Provider     Origin `json:"provider"`
	Model        string `json:"model,omitempty"`
	ParentModel  string `json:"parentModel"`
	Background   bool   `json:"background"`
	Fork         bool   `json:"fork"`
	IsTeammate   bool   `json:"isTeammate,omitempty"`
	Name         string `json:"name,omitempty"`
	Cwd          string `json:"cwd,omitempty"`
	Resume       *struct {
		AgentID string `json:"agentId"`
	} `json:"resume,omitempty"`
}

// AgentOfferEvent is the agent.offer event: a subagent type is offered to
// Claude. Answer with {"isOffered": false} to withhold it.
type AgentOfferEvent struct {
	Agent       string `json:"agent"`
	Description string `json:"description"`
	Source      string `json:"source"`
	Provider    Origin `json:"provider"`
	AgentID     string `json:"agentId,omitempty"`
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

func (a *App) OnSessionEnd(h func(*Ctx, SessionEndEvent) (any, error)) *App {
	return a.On("session.end", typed(h))
}

func (a *App) OnSessionStop(h func(*Ctx, SessionStopEvent) (any, error)) *App {
	return a.On("session.stop", typed(h))
}

func (a *App) OnSessionCompact(h func(*Ctx, SessionCompactEvent) (any, error)) *App {
	return a.On("session.compact", typed(h))
}

func (a *App) OnTurnStart(h func(*Ctx, TurnStartEvent) (any, error)) *App {
	return a.On("turn.start", typed(h))
}

func (a *App) OnPromptFill(h func(*Ctx, PromptFillEvent) (any, error)) *App {
	return a.On("prompt.fill", typed(h))
}

func (a *App) OnPromptSuggest(h func(*Ctx, PromptSuggestEvent) (any, error)) *App {
	return a.On("prompt.suggest", typed(h))
}

// OnToolDescribe handles tool.describe; tool narrows it to one tool ("" for all).
func (a *App) OnToolDescribe(tool string, h func(*Ctx, ToolDescribeEvent) (any, error)) *App {
	if tool == "" {
		return a.On("tool.describe", typed(h))
	}
	return a.On("tool.describe", typed(h), Where("tool", tool))
}

// OnToolOffer handles tool.offer; tool narrows it to one tool ("" for all).
func (a *App) OnToolOffer(tool string, h func(*Ctx, ToolOfferEvent) (any, error)) *App {
	if tool == "" {
		return a.On("tool.offer", typed(h))
	}
	return a.On("tool.offer", typed(h), Where("tool", tool))
}

// OnCommandDescribe handles command.describe; command narrows it to one command ("" for all).
func (a *App) OnCommandDescribe(command string, h func(*Ctx, CommandDescribeEvent) (any, error)) *App {
	if command == "" {
		return a.On("command.describe", typed(h))
	}
	return a.On("command.describe", typed(h), Where("command", command))
}

// OnConfigSet handles config.set; key narrows it to one /config row ("" for all).
func (a *App) OnConfigSet(key string, h func(*Ctx, ConfigSetEvent) (any, error)) *App {
	if key == "" {
		return a.On("config.set", typed(h))
	}
	return a.On("config.set", typed(h), Where("key", key))
}

// OnAgentSpawn handles agent.spawn; subagentType narrows it to one type ("" for all).
func (a *App) OnAgentSpawn(subagentType string, h func(*Ctx, AgentSpawnEvent) (any, error)) *App {
	if subagentType == "" {
		return a.On("agent.spawn", typed(h))
	}
	return a.On("agent.spawn", typed(h), Where("subagentType", subagentType))
}

// OnAgentOffer handles agent.offer; agent narrows it to one subagent type ("" for all).
func (a *App) OnAgentOffer(agent string, h func(*Ctx, AgentOfferEvent) (any, error)) *App {
	if agent == "" {
		return a.On("agent.offer", typed(h))
	}
	return a.On("agent.offer", typed(h), Where("agent", agent))
}
