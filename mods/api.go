package mods

import "encoding/json"

// Typed helpers for the common mods API methods. Each needs its Method
// declared with App.Uses. Anything without a helper is reachable through
// Ctx.Call.

type (
	UI       struct{ c *Ctx }
	Commands struct{ c *Ctx }
	Tools    struct{ c *Ctx }
	FS       struct{ c *Ctx }
	Store    struct{ c *Ctx }
	HTTP     struct{ c *Ctx }
	Process  struct{ c *Ctx }
	Model    struct{ c *Ctx }
	Session  struct{ c *Ctx }
)

// ToastOptions mirrors $.ui.toast's second argument.
type ToastOptions struct {
	TimeoutMs int `json:"timeoutMs,omitempty"`
}

// Toast shows a transient message ($.ui.toast).
func (u UI) Toast(text string, o ...ToastOptions) error {
	args := []any{text}
	if len(o) > 0 {
		args = append(args, o[0])
	}
	_, err := u.c.Call(UIToast, args...)
	return err
}

// Status sets the status line ($.ui.status); "" clears it.
func (u UI) Status(text string) error {
	var arg any = text
	if text == "" {
		arg = nil
	}
	_, err := u.c.Call(UIStatus, arg)
	return err
}

// Log writes a line to the transcript ($.ui.log).
func (u UI) Log(text string) error {
	_, err := u.c.Call(UILog, text)
	return err
}

// CommandSpec is the argument of $.command.register.
type CommandSpec struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	ArgumentHint string `json:"argumentHint,omitempty"`
	Immediate    bool   `json:"immediate,omitempty"`
}

// Register adds a slash command ($.command.register). Its runs reach the
// handler registered with OnCommand.
func (cm Commands) Register(s CommandSpec) error {
	_, err := cm.c.Call(CommandRegister, s)
	return err
}

// Register adds a tool the model can call ($.tool.register). The spec is
// passed through as given, so it follows Claude Code's ToolSpec.
func (t Tools) Register(spec any) error {
	_, err := t.c.Call(ToolRegister, spec)
	return err
}

// Read returns a file's text ($.fs.read).
func (f FS) Read(path string) (string, error) {
	return decode[string](f.c.Call(FSRead, path))
}

// Write replaces a file's text ($.fs.write).
func (f FS) Write(path, text string) error {
	_, err := f.c.Call(FSWrite, path, text)
	return err
}

// Exists reports whether a path exists ($.fs.exists).
func (f FS) Exists(path string) (bool, error) {
	return decode[bool](f.c.Call(FSExists, path))
}

// Get reads a stored value into out ($.store.get). It reports whether the key
// held anything.
func (s Store) Get(key string, out any) (bool, error) {
	raw, err := s.c.Call(StoreGet, key)
	if err != nil {
		return false, err
	}
	if len(raw) == 0 || string(raw) == "null" {
		return false, nil
	}
	return true, json.Unmarshal(raw, out)
}

// Set stores a JSON value ($.store.set).
func (s Store) Set(key string, value any) error {
	_, err := s.c.Call(StoreSet, key, value)
	return err
}

// Delete removes a key ($.store.delete).
func (s Store) Delete(key string) error {
	_, err := s.c.Call(StoreDelete, key)
	return err
}

// Keys lists stored keys ($.store.keys).
func (s Store) Keys() ([]string, error) {
	return decode[[]string](s.c.Call(StoreKeys))
}

// FetchInit is the optional second argument of $.http.fetch.
type FetchInit struct {
	Method  string            `json:"method,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
}

// Response is $.http.fetch's result.
type Response struct {
	Status  int               `json:"status"`
	OK      bool              `json:"ok"`
	Headers map[string]string `json:"headers"`
	Text    string            `json:"text"`
}

// Fetch makes an HTTP request ($.http.fetch).
func (h HTTP) Fetch(url string, init ...FetchInit) (Response, error) {
	args := []any{url}
	if len(init) > 0 {
		args = append(args, init[0])
	}
	return decode[Response](h.c.Call(HTTPFetch, args...))
}

// RunOptions is the optional second argument of $.process.run.
type RunOptions struct {
	Stdin     string            `json:"stdin,omitempty"`
	Cwd       string            `json:"cwd,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	TimeoutMs int               `json:"timeoutMs,omitempty"`
}

// RunResult is $.process.run's result.
type RunResult struct {
	ExitCode int    `json:"exitCode"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

// Run runs a program to completion ($.process.run).
func (p Process) Run(argv []string, o ...RunOptions) (RunResult, error) {
	args := []any{argv}
	if len(o) > 0 {
		args = append(args, o[0])
	}
	return decode[RunResult](p.c.Call(ProcessRun, args...))
}

// Complete asks the model ($.model.complete). The result shape differs between
// Claude Code versions, so it is returned as JSON.
func (m Model) Complete(req any) (json.RawMessage, error) {
	return m.c.Call(ModelComplete, req)
}

// Cwd is the session's working directory ($.session.cwd).
func (s Session) Cwd() (string, error) {
	return decode[string](s.c.Call(SessionCwd))
}

// ID is the session id ($.session.id).
func (s Session) ID() (string, error) {
	return decode[string](s.c.Call(SessionID))
}
