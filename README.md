# claude-mods-go

Write [Claude Code mods](https://claude.dev/mods/) in Go.

A mod's hooks module is JavaScript with no Node APIs, so Go cannot be loaded directly. This library
splits the work: you write handlers in Go against package `mods`; `modgo build` compiles them and
generates a small JavaScript shim that forwards each event to your binary and carries out the
`$` calls and `next(e)` your handler asks for.

## Install

```sh
go install github.com/taumatix/claude-mods-go/cmd/modgo@latest
go get github.com/taumatix/claude-mods-go/mods
```

Requires Go 1.22+ and Claude Code with mods (2.1.290+). Tested against the release pinned in
[UPSTREAM.md](UPSTREAM.md).

## Quickstart

```go
package main

import "github.com/taumatix/claude-mods-go/mods"

func main() {
	app := mods.New("hello").Uses(mods.CommandRegister)

	app.OnSessionStart(func(c *mods.Ctx, e mods.SessionStartEvent) (any, error) {
		err := c.Command.Register(mods.CommandSpec{Name: "hello", Description: "Say hello"})
		if err != nil {
			return nil, err
		}
		return c.Pass() // next(e)
	})
	app.OnCommand("hello", func(c *mods.Ctx, e mods.CommandRunEvent) (any, error) {
		return mods.Text("hello from Go"), nil
	})
	app.Run()
}
```

```sh
modgo build -o plugin .
modgo validate plugin
claude --plugin-dir plugin        # then /hello
```

See [examples/tally](examples/tally) for a mod that keeps state in `$.store`.

## How it works

- `Uses(...)` declares which `$` methods the mod calls. The shim contains exactly those, so
  `claude plugin validate` and administrators see the real reach. An undeclared call fails with
  an error naming it.
- Handlers return what the hook returns: `c.Pass()` / `c.Next(e)` to continue, or an answer such
  as `mods.Text(...)`, `mods.Deny(...)`. Returning nil continues with `next(e)`.
- `c.Call(method, args...)` reaches any bridgeable method that lacks a typed helper.
- Handler failures and failed `$` calls come back as errors; a panic is reported, not fatal.

### Transports

`modgo build -transport auto|daemon|once` (default `auto`).

| | daemon | once |
|---|---|---|
| Process | one Go process per session, loopback HTTP with a per-session bearer token | a process per step |
| State between events | kept in memory | none (use `$.store`) |
| Needs | `$.process.spawn`, `$.http.fetch` | `$.process.run` |
| Handler rule | none | deterministic: it is re-run from the top, replaying recorded answers |

`auto` starts the daemon and falls back to `once` if it cannot.

## Limits

- Not bridged: callback methods (`$.clock.every`, `$.state`), `$.env`, `$.ui.open/resolve/blit`.
- Binaries are located with `uname`, so Windows is not supported yet. See [ROADMAP.md](ROADMAP.md).
- Hooks have a 10 s budget of their own execution; awaits on bridged calls do not count.

## Testing

`go test ./...` runs unit tests and, when `claude` is on `PATH` (or `CLAUDE_BIN`), end-to-end tests
that build mods and run them with `claude -p "/command" --plugin-dir`, in all three transports.
`claude plugin test` stubs `$.process.*`, so it cannot exercise a Go mod.

License: Apache-2.0.
