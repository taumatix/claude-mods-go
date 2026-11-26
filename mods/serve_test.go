package mods

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
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

// listen starts the daemon with the given idle timeout and reports its base URL
// and a channel that yields serve's return value.
func listen(t *testing.T, a *App, idle time.Duration) (string, <-chan error) {
	t.Helper()
	r, w := io.Pipe()
	done := make(chan error, 1)
	go func() { done <- a.serveIdle(w, "secret", idle) }()
	line := make([]byte, 64)
	n, err := r.Read(line)
	if err != nil {
		t.Fatal(err)
	}
	return "http://" + strings.TrimSpace(strings.TrimPrefix(string(line[:n]), "LISTEN ")), done
}

func exits(t *testing.T, done <-chan error, within time.Duration) {
	t.Helper()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serve returned %v, want a clean exit", err)
		}
	case <-time.After(within):
		t.Fatalf("daemon still running after %s", within)
	}
}

func TestDaemonExitsWhenIdle(t *testing.T) {
	base, done := listen(t, testApp(), 50*time.Millisecond)
	exits(t, done, 5*time.Second)
	if _, err := http.Post(base+"/start", "application/json", nil); err == nil {
		t.Fatal("the port should be closed after an idle exit")
	}
}

func TestRequestsPostponeTheIdleExit(t *testing.T) {
	base, done := listen(t, testApp(), 300*time.Millisecond)
	p := pipe{t}
	for i := 0; i < 5; i++ {
		time.Sleep(100 * time.Millisecond)
		if res, _ := p.post(base, "/start", "wrong", Start{}); res.StatusCode != 401 {
			t.Fatalf("round %d: got %d", i, res.StatusCode)
		}
	}
	exits(t, done, 5*time.Second)
}

func TestDaemonStaysWhileARunIsLive(t *testing.T) {
	base, done := listen(t, testApp(), 50*time.Millisecond)
	p := pipe{t}
	_, body := p.post(base, "/start", "secret", Start{Reg: 0, Event: "command.run", E: json.RawMessage(`{"command":"show","args":"f"}`)})
	var s Step
	_ = json.Unmarshal(body, &s)

	select {
	case err := <-done:
		t.Fatalf("daemon exited with a run waiting on the shim: %v", err)
	case <-time.After(500 * time.Millisecond):
	}

	for _, v := range []string{`"hi"`, ``} {
		_, body = p.post(base, "/resume", "secret", Resume{Run: s.Run, CID: s.CID, OK: true, Value: json.RawMessage(v)})
		_ = json.Unmarshal(body, &s)
	}
	if s.Op != OpDone {
		t.Fatalf("final = %+v", s)
	}
	exits(t, done, 5*time.Second)
}

func TestAbandonedRunIsDroppedAndTheDaemonExits(t *testing.T) {
	old := runLease
	runLease = 100 * time.Millisecond
	defer func() { runLease = old }()

	base, done := listen(t, testApp(), 50*time.Millisecond)
	p := pipe{t}
	p.post(base, "/start", "secret", Start{Reg: 0, Event: "command.run", E: json.RawMessage(`{"command":"show","args":"f"}`)})
	exits(t, done, 5*time.Second)
}

func TestNoIdleTimeoutMeansNeverExit(t *testing.T) {
	_, done := listen(t, testApp(), 0)
	select {
	case err := <-done:
		t.Fatalf("daemon exited with idle timeout disabled: %v", err)
	case <-time.After(300 * time.Millisecond):
	}
}

func TestIdleTimeoutOption(t *testing.T) {
	a := New("t")
	if a.idleTimeout != DefaultIdleTimeout {
		t.Fatalf("default = %s, want %s", a.idleTimeout, DefaultIdleTimeout)
	}
	if got := a.IdleTimeout(time.Second).idleTimeout; got != time.Second {
		t.Fatalf("got %s", got)
	}
	if got := a.IdleTimeout(0).idleTimeout; got != 0 {
		t.Fatalf("got %s", got)
	}
	if got := a.IdleTimeout(time.Millisecond).idleTimeout; got != MinIdleTimeout {
		t.Fatalf("got %s, want it raised to %s", got, MinIdleTimeout)
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
