package mods

import (
	"encoding/json"
	"reflect"
	"testing"
)

type eventCase struct {
	event   string
	match   map[string]string
	payload string
	reg     func(a *App, got *any)
	want    any
}

func TestTypedEventsDecodeTheirPayloads(t *testing.T) {
	cases := []eventCase{
		{"session.end", nil, `{"reason":"clear","sessionId":"s1","resume":{"id":"r1"}}`,
			func(a *App, got *any) {
				a.OnSessionEnd(func(c *Ctx, e SessionEndEvent) (any, error) {
					*got = e.Reason + e.SessionID + e.Resume.ID
					return nil, nil
				})
			}, "clears1r1"},
		{"session.stop", nil, `{"isTurnRunning":true}`,
			func(a *App, got *any) {
				a.OnSessionStop(func(c *Ctx, e SessionStopEvent) (any, error) { *got = e.IsTurnRunning; return nil, nil })
			}, true},
		{"session.compact", nil, `{"trigger":"auto","instructions":"keep it short","messages":[1]}`,
			func(a *App, got *any) {
				a.OnSessionCompact(func(c *Ctx, e SessionCompactEvent) (any, error) {
					*got = e.Trigger + "|" + e.Instructions + "|" + string(e.Messages)
					return map[string]string{"skip": "no"}, nil
				})
			}, "auto|keep it short|[1]"},
		{"turn.start", nil, `{"text":"hi","turnId":"t1","agentId":"a1"}`,
			func(a *App, got *any) {
				a.OnTurnStart(func(c *Ctx, e TurnStartEvent) (any, error) { *got = e.Text + e.TurnID + e.AgentID; return nil, nil })
			}, "hit1a1"},
		{"prompt.fill", nil, `{"text":"draft","mode":"append","origin":{"kind":"plugin","name":"x"},"cursor":3}`,
			func(a *App, got *any) {
				a.OnPromptFill(func(c *Ctx, e PromptFillEvent) (any, error) {
					*got = e.Text + e.Mode + e.Origin.Kind + e.Origin.Name + string(rune('0'+*e.Cursor))
					return nil, nil
				})
			}, "draftappendpluginx3"},
		{"prompt.suggest", nil, `{"text":"run tests","origin":{"kind":"engine"}}`,
			func(a *App, got *any) {
				a.OnPromptSuggest(func(c *Ctx, e PromptSuggestEvent) (any, error) { *got = e.Text + e.Origin.Kind; return nil, nil })
			}, "run testsengine"},
		{"tool.describe", map[string]string{"tool": "Bash"}, `{"tool":"Bash","description":"d","isDeferred":true,"provider":{"plugin":"p","tier":"builtin"}}`,
			func(a *App, got *any) {
				a.OnToolDescribe("Bash", func(c *Ctx, e ToolDescribeEvent) (any, error) {
					*got = e.Tool + e.Description + e.Provider.Plugin + e.Provider.Tier
					return map[string]string{"description": "new"}, nil
				})
			}, "Bashdpbuiltin"},
		{"tool.offer", map[string]string{"tool": "Edit"}, `{"tool":"Edit","provider":{"plugin":"p","tier":"t"},"for":"call","subagentType":"Explore"}`,
			func(a *App, got *any) {
				a.OnToolOffer("Edit", func(c *Ctx, e ToolOfferEvent) (any, error) {
					*got = e.Tool + e.For + e.SubagentType
					return map[string]bool{"isOffered": false}, nil
				})
			}, "EditcallExplore"},
		{"command.describe", map[string]string{"command": "pid"}, `{"command":"pid","description":"d","argumentHint":"<x>","isHidden":false,"immediate":true,"provider":{"plugin":"p","tier":"t"}}`,
			func(a *App, got *any) {
				a.OnCommandDescribe("pid", func(c *Ctx, e CommandDescribeEvent) (any, error) {
					*got = e.Command + e.ArgumentHint + e.Provider.Plugin
					return nil, nil
				})
			}, "pid<x>p"},
		{"config.set", map[string]string{"key": "theme"}, `{"key":"theme","value":"dark","previous":"light","provider":{"plugin":"p","tier":"t"},"origin":{"kind":"bridge"}}`,
			func(a *App, got *any) {
				a.OnConfigSet("theme", func(c *Ctx, e ConfigSetEvent) (any, error) {
					*got = e.Key + string(e.Value) + string(e.Previous) + e.Origin.Kind
					return nil, nil
				})
			}, `theme"dark""light"bridge`},
		{"agent.spawn", map[string]string{"subagentType": "Explore"}, `{"tool_use_id":"u1","prompt":"p","description":"d","subagentType":"Explore","provider":{"plugin":"p","tier":"t"},"parentModel":"m","background":true,"fork":false,"resume":{"agentId":"a9"}}`,
			func(a *App, got *any) {
				a.OnAgentSpawn("Explore", func(c *Ctx, e AgentSpawnEvent) (any, error) {
					*got = e.ToolUseID + e.SubagentType + e.ParentModel + e.Resume.AgentID
					return map[string]string{"deny": "no"}, nil
				})
			}, "u1Explorema9"},
		{"agent.offer", map[string]string{"agent": "Plan"}, `{"agent":"Plan","description":"d","source":"built-in","provider":{"plugin":"p","tier":"t"}}`,
			func(a *App, got *any) {
				a.OnAgentOffer("Plan", func(c *Ctx, e AgentOfferEvent) (any, error) {
					*got = e.Agent + e.Source + e.Provider.Tier
					return map[string]bool{"isOffered": false}, nil
				})
			}, "Planbuilt-int"},
	}
	for _, tc := range cases {
		t.Run(tc.event, func(t *testing.T) {
			var got any
			a := New("t")
			tc.reg(a, &got)
			m, err := a.Manifest()
			if err != nil {
				t.Fatal(err)
			}
			if len(m.Hooks) != 1 || m.Hooks[0].Event != tc.event {
				t.Fatalf("hooks = %+v, want one for %s", m.Hooks, tc.event)
			}
			if !reflect.DeepEqual(m.Hooks[0].Match, tc.match) {
				t.Errorf("match = %v, want %v", m.Hooks[0].Match, tc.match)
			}
			step := a.execute(nil, "r", Start{Reg: 0, Event: tc.event, E: json.RawMessage(tc.payload)})
			if step.Op != OpDone {
				t.Fatalf("step = %+v", step)
			}
			if got != tc.want {
				t.Errorf("decoded %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestUnfilteredEventHooksCarryNoMatcher(t *testing.T) {
	a := New("t")
	h := func(c *Ctx, e ToolOfferEvent) (any, error) { return nil, nil }
	a.OnToolOffer("", h)
	m, _ := a.Manifest()
	if m.Hooks[0].Match != nil {
		t.Errorf("match = %v, want none", m.Hooks[0].Match)
	}
}
