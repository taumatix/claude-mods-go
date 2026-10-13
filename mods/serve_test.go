package mods

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func testApp() *App {
	a := New("t").Uses(FSRead, UIToast)
	a.OnCommand("show", func(c *Ctx, e CommandRunEvent) (any, error) {
		txt, err := c.FS.Read("/x/" + e.Args)
		if err != nil {
			return nil, err
		}
		if err := c.UI.Toast("read " + txt); err != nil {
			return nil, err
		}
		return Text(strings.ToUpper(txt)), nil
	})
	a.OnToolCall("Bash", func(c *Ctx, e ToolCallEvent) (any, error) {
		return c.Next(map[string]any{"tool": e.Tool, "tool_use_id": e.ToolUseID, "command": "true"})
	})
	return a
}

type pipe struct{ t *testing.T }

func (p pipe) post(base, path, token string, v any) (*http.Response, []byte) {
	b, _ := json.Marshal(v)
	req, _ := http.NewRequest("POST", base+path, bytes.NewReader(b))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		p.t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	return res, body
}

func TestDaemonRoundTripOverHTTP(t *testing.T) {
	a := testApp()
	r, w := io.Pipe()
	go func() { _ = a.serve(w, "secret") }()
	line := make([]byte, 64)
	n, err := r.Read(line)
	if err != nil {
		t.Fatal(err)
	}
	base := "http://" + strings.TrimSpace(strings.TrimPrefix(string(line[:n]), "LISTEN "))
	p := pipe{t}

	if res, _ := p.post(base, "/start", "wrong", Start{}); res.StatusCode != 401 {
		t.Fatalf("bad token got %d, want 401", res.StatusCode)
	}

	_, body := p.post(base, "/start", "secret", Start{Reg: 0, Event: "command.run", E: json.RawMessage(`{"command":"show","args":"f"}`)})
	var s Step
	_ = json.Unmarshal(body, &s)
	if s.Op != OpCall || s.Method != "fs.read" || string(s.Args[0]) != `"/x/f"` {
		t.Fatalf("first step = %+v", s)
	}
	_, body = p.post(base, "/resume", "secret", Resume{Run: s.Run, CID: s.CID, OK: true, Value: json.RawMessage(`"hi"`)})
	_ = json.Unmarshal(body, &s)
	if s.Method != "ui.toast" {
		t.Fatalf("second step = %+v", s)
	}
	_, body = p.post(base, "/resume", "secret", Resume{Run: s.Run, CID: s.CID, OK: true})
	_ = json.Unmarshal(body, &s)
	if s.Op != OpDone || string(s.Value) != `{"text":"HI"}` {
		t.Fatalf("final = %+v", s)
	}
}

func TestDaemonSurfacesHostError(t *testing.T) {
	a := testApp()
	st := Start{Reg: 0, Event: "command.run", E: json.RawMessage(`{"args":"f"}`)}
	d := &fakeDriver{answers: []Resume{{OK: false, Error: "ENOENT"}}}
	got := a.execute(d, "r", st)
	if got.Op != OpError || !strings.Contains(got.Message, "fs.read: ENOENT") {
		t.Fatalf("got %+v", got)
	}
}

type fakeDriver struct {
	answers []Resume
	seen    []Step
}

func (f *fakeDriver) step(s Step) (Resume, error) {
	f.seen = append(f.seen, s)
	r := f.answers[0]
	f.answers = f.answers[1:]
	return r, nil
}

func TestUndeclaredMethodIsRefused(t *testing.T) {
	a := New("t").Uses(FSRead)
	a.OnSessionStart(func(c *Ctx, e SessionStartEvent) (any, error) {
		return nil, c.UI.Toast("x")
	})
	got := a.execute(&fakeDriver{}, "r", Start{Reg: 0, Event: "session.start", E: json.RawMessage(`{}`)})
	if got.Op != OpError || !strings.Contains(got.Message, "ui.toast is not declared") {
		t.Fatalf("got %+v", got)
	}
}

func TestPanicBecomesError(t *testing.T) {
	a := New("t")
	a.OnPromptSubmit(func(*Ctx, PromptSubmitEvent) (any, error) { panic("boom") })
	got := a.execute(&fakeDriver{}, "r", Start{Reg: 0, Event: "prompt.submit", E: json.RawMessage(`{}`)})
	if got.Op != OpError || !strings.Contains(got.Message, "boom") {
		t.Fatalf("got %+v", got)
	}
}

func TestMismatchedBuildIsReported(t *testing.T) {
	a := testApp()
	got := a.execute(&fakeDriver{}, "r", Start{Reg: 0, Event: "tool.call", E: json.RawMessage(`{}`)})
	if got.Op != OpError || !strings.Contains(got.Message, "different builds") {
		t.Fatalf("got %+v", got)
	}
}

func onceStep(t *testing.T, a *App, st Start) Step {
	t.Helper()
	in, _ := json.Marshal(st)
	var out bytes.Buffer
	if err := a.once(bytes.NewReader(in), &out); err != nil {
		t.Fatal(err)
	}
	var s Step
	if err := json.Unmarshal(out.Bytes(), &s); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestOnceReplaysJournalToCompletion(t *testing.T) {
	a := testApp()
	st := Start{Reg: 0, Event: "command.run", E: json.RawMessage(`{"command":"show","args":"f"}`)}

	s := onceStep(t, a, st)
	if s.Op != OpCall || s.Method != "fs.read" {
		t.Fatalf("step 1 = %+v", s)
	}
	st.Journal = append(st.Journal, JournalEntry{Op: OpCall, Method: "fs.read", OK: true, Value: json.RawMessage(`"hi"`)})
	s = onceStep(t, a, st)
	if s.Method != "ui.toast" {
		t.Fatalf("step 2 = %+v", s)
	}
	st.Journal = append(st.Journal, JournalEntry{Op: OpCall, Method: "ui.toast", OK: true})
	s = onceStep(t, a, st)
	if s.Op != OpDone || string(s.Value) != `{"text":"HI"}` {
		t.Fatalf("final = %+v", s)
	}
}

func TestOnceNextStep(t *testing.T) {
	a := testApp()
	st := Start{Reg: 1, Event: "tool.call", E: json.RawMessage(`{"tool":"Bash","tool_use_id":"u","command":"rm -rf /"}`)}
	s := onceStep(t, a, st)
	if s.Op != OpNext || !strings.Contains(string(s.Event), `"command":"true"`) {
		t.Fatalf("got %+v", s)
	}
	st.Journal = []JournalEntry{{Op: OpNext, Method: "next", OK: true, Value: json.RawMessage(`{"ok":1}`)}}
	s = onceStep(t, a, st)
	if s.Op != OpDone || string(s.Value) != `{"ok":1}` {
		t.Fatalf("final = %+v", s)
	}
}

func TestOnceDetectsNondeterminism(t *testing.T) {
	a := testApp()
	st := Start{Reg: 0, Event: "command.run", E: json.RawMessage(`{"args":"f"}`),
		Journal: []JournalEntry{{Op: OpCall, Method: "fs.write", OK: true}}}
	s := onceStep(t, a, st)
	if s.Op != OpError || !strings.Contains(s.Message, "not deterministic") {
		t.Fatalf("got %+v", s)
	}
}

func TestManifest(t *testing.T) {
	m, err := testApp().Manifest()
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Hooks) != 2 || m.Hooks[0].Match["command"] != "show" || m.Hooks[1].Match["tool"] != "Bash" {
		t.Fatalf("hooks = %+v", m.Hooks)
	}
	if len(m.Uses) != 2 || m.Uses[0] != FSRead {
		t.Fatalf("uses = %+v", m.Uses)
	}
	if _, err := New("x").Manifest(); err == nil {
		t.Fatal("no hooks should be an error")
	}
	if _, err := New("x").Uses("nope.nope").OnPromptSubmit(func(*Ctx, PromptSubmitEvent) (any, error) { return nil, nil }).Manifest(); err == nil {
		t.Fatal("unknown method should be an error")
	}
}
