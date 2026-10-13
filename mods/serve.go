package mods

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
)

// TokenEnv names the environment variable carrying the daemon's bearer token.
const TokenEnv = "MODGO_TOKEN"

// Run dispatches on os.Args and exits: the mod binary's whole main.
//
//	manifest   print the Manifest as JSON (modgo reads it)
//	serve      long-lived daemon on a loopback port
//	once       handle one event from stdin, replaying its journal
func (a *App) Run() {
	cmd := ""
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	var err error
	switch cmd {
	case "manifest":
		var m Manifest
		if m, err = a.Manifest(); err == nil {
			err = json.NewEncoder(os.Stdout).Encode(m)
		}
	case "serve":
		err = a.serve(os.Stdout, os.Getenv(TokenEnv))
	case "once":
		err = a.once(os.Stdin, os.Stdout)
	default:
		err = fmt.Errorf("usage: %s manifest|serve|once", os.Args[0])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// ---- daemon ----

type liveRun struct {
	out chan Step
	in  chan Resume
}

func (r *liveRun) step(s Step) (Resume, error) {
	r.out <- s
	return <-r.in, nil
}

type daemon struct {
	app   *App
	token string
	mu    sync.Mutex
	runs  map[string]*liveRun
	next  int
}

func (a *App) serve(stdout io.Writer, token string) error {
	if token == "" {
		return fmt.Errorf("mods: serve needs %s set", TokenEnv)
	}
	if _, err := a.Manifest(); err != nil {
		return err
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	d := &daemon{app: a, token: token, runs: map[string]*liveRun{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/start", d.handle(d.start))
	mux.HandleFunc("/resume", d.handle(d.resume))
	fmt.Fprintf(stdout, "LISTEN %s\n", ln.Addr())
	if f, ok := stdout.(interface{ Sync() error }); ok {
		_ = f.Sync()
	}
	return http.Serve(ln, mux)
}

func (d *daemon) handle(f func(io.Reader) (Step, int, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if r.Method != http.MethodPost || subtle.ConstantTimeCompare([]byte(got), []byte(d.token)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		s, code, err := f(http.MaxBytesReader(w, r.Body, 32<<20))
		if err != nil {
			http.Error(w, err.Error(), code)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(s)
	}
}

func (d *daemon) start(body io.Reader) (Step, int, error) {
	var st Start
	if err := json.NewDecoder(body).Decode(&st); err != nil {
		return Step{}, http.StatusBadRequest, err
	}
	d.mu.Lock()
	d.next++
	id := strconv.Itoa(d.next)
	run := &liveRun{out: make(chan Step), in: make(chan Resume)}
	d.runs[id] = run
	d.mu.Unlock()
	go func() { run.out <- d.app.execute(run, id, st) }()
	return d.await(id, run), 0, nil
}

func (d *daemon) resume(body io.Reader) (Step, int, error) {
	var r Resume
	if err := json.NewDecoder(body).Decode(&r); err != nil {
		return Step{}, http.StatusBadRequest, err
	}
	d.mu.Lock()
	run := d.runs[r.Run]
	d.mu.Unlock()
	if run == nil {
		return Step{}, http.StatusNotFound, fmt.Errorf("no run %q", r.Run)
	}
	run.in <- r
	return d.await(r.Run, run), 0, nil
}

func (d *daemon) await(id string, run *liveRun) Step {
	s := <-run.out
	if s.Op == OpDone || s.Op == OpError {
		d.mu.Lock()
		delete(d.runs, id)
		d.mu.Unlock()
	}
	return s
}

// ---- one-shot ----

type suspendSignal struct{}

type replay struct {
	journal []JournalEntry
	i       int
	pending *Step
}

func (r *replay) step(s Step) (Resume, error) {
	if r.i < len(r.journal) {
		e := r.journal[r.i]
		r.i++
		if e.Op != s.Op || e.Method != s.Method {
			return Resume{}, fmt.Errorf("mods: handler is not deterministic: step %d was %s %s, now %s %s", r.i, e.Op, e.Method, s.Op, s.Method)
		}
		return Resume{CID: s.CID, OK: e.OK, Value: e.Value, Error: e.Error}, nil
	}
	r.pending = &s
	panic(suspendSignal{})
}

// once replays the journal through the handler. When it reaches a step with no
// recorded answer it prints that step and stops; the shim answers it, extends
// the journal and calls again. The handler is therefore re-run from the top
// each time and must make the same calls in the same order.
func (a *App) once(in io.Reader, out io.Writer) error {
	var st Start
	if err := json.NewDecoder(in).Decode(&st); err != nil {
		return fmt.Errorf("mods: reading Start: %w", err)
	}
	r := &replay{journal: st.Journal}
	final := a.execute(r, "", st)
	if r.pending != nil {
		final = *r.pending
	}
	return json.NewEncoder(out).Encode(final)
}
