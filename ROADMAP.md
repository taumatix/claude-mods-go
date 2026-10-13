# Roadmap

Ordered by how much each entry limits a real deployment.

1. **Daemon idle exit.** The daemon relies on Claude Code killing it at session end (observed on
   2.1.296). Add an idle timeout so a leaked one cannot linger, and test a module reload.
2. **Windows.** The shim picks its binary with `uname -sm`, and `modgo build` defaults to the host.
   Detect Windows without `uname`, name the binary `.exe`, and cover it in CI.
3. **Event coverage.** Typed events exist for six hooks. Add the rest (`session.end`,
   `turn.start`, `prompt.fill`, `tool.describe`, ...) from the d.ts, with a test per event.
4. **A live `tool.call` test.** `next(e)` through a real tool call and a rewrite are covered only by
   unit tests; the live tests need no model and so cannot raise one. Drive it with a model-free
   `$.tool.call` from a command, or a gated live run.
5. **`claude plugin test` support.** It stubs `$.process.*`, so a Go mod cannot use it; ship a
   test double for the bridge or a `modgo test` that runs the live path.
6. **Bridge more of `$`.** Callback methods (`$.clock.every`, `$.state`), `$.env` and the drawing
   calls (`$.ui.open`) are not bridged.
7. **Track upstream.** Re-check `UPSTREAM.md` each maintenance pass and turn new mods API surface
   into entries here.
