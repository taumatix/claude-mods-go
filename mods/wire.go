package mods

import "encoding/json"

// The wire protocol between the generated JavaScript shim and the Go program.
//
// The shim sends a Start for every event a hook receives. The Go handler then
// runs and asks the shim to do things only JavaScript can: call a mods API
// method ($.fs.read, ...) or call next(e). Each ask is a Step; the shim answers
// with a Resume and the handler continues, until it returns a Step with Op
// "done" or "error".

// Step ops.
const (
	OpCall  = "call"
	OpNext  = "next"
	OpDone  = "done"
	OpError = "error"
)

// Step is one message from the Go program to the shim.
type Step struct {
	Run     string            `json:"run,omitempty"`
	Op      string            `json:"op"`
	CID     int               `json:"cid,omitempty"`
	Method  string            `json:"method,omitempty"`
	Args    []json.RawMessage `json:"args,omitempty"`
	Event   json.RawMessage   `json:"e,omitempty"`
	Value   json.RawMessage   `json:"value,omitempty"`
	Message string            `json:"message,omitempty"`
}

// Resume answers a Step of Op call or next.
type Resume struct {
	Run   string          `json:"run,omitempty"`
	CID   int             `json:"cid"`
	OK    bool            `json:"ok"`
	Value json.RawMessage `json:"value,omitempty"`
	Error string          `json:"error,omitempty"`
}

// Start begins one event. Reg indexes Manifest.Hooks. Journal is used only by
// the one-shot transport: it replays the answers to the Steps already taken.
type Start struct {
	Reg     int             `json:"reg"`
	Event   string          `json:"event"`
	E       json.RawMessage `json:"e"`
	Journal []JournalEntry  `json:"journal,omitempty"`
}

// JournalEntry records one answered Step for replay by the one-shot transport.
type JournalEntry struct {
	Op     string          `json:"op"`
	Method string          `json:"method,omitempty"`
	OK     bool            `json:"ok"`
	Value  json.RawMessage `json:"value,omitempty"`
	Error  string          `json:"error,omitempty"`
}

// Hook is one registration in the manifest.
type Hook struct {
	Event string            `json:"event"`
	Match map[string]string `json:"match,omitempty"`
}

// Manifest describes a mod to the generator: what to register and which mods
// API methods the shim may call on its behalf. The shim contains exactly these
// and nothing else, so `claude plugin validate` lists what the mod can do.
type Manifest struct {
	Name        string   `json:"name"`
	Version     string   `json:"version,omitempty"`
	Description string   `json:"description,omitempty"`
	Hooks       []Hook   `json:"hooks"`
	Uses        []Method `json:"uses"`
}
