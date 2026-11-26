# Roadmap

Ordered by how much each entry limits a real deployment.

1. **Event coverage.** Typed events exist for six hooks. Add the rest (`session.end`,
   `turn.start`, `prompt.fill`, `tool.describe`, ...) from the d.ts, with a test per event.
2. **A live `tool.call` test.** `next(e)` through a real tool call and a rewrite are covered only by
   unit tests; the live tests need no model and so cannot raise one. Drive it with a model-free
   `$.tool.call` from a command, or a gated live run.
3. **`claude plugin test` support.** It stubs `$.process.*`, so a Go mod cannot use it; ship a
   test double for the bridge or a `modgo test` that runs the live path.
4. **Bridge more of `$`.** Callback methods (`$.clock.every`, `$.state`), `$.env` and the drawing
   calls (`$.ui.open`) are not bridged.
5. **Track upstream.** Re-check `UPSTREAM.md` each maintenance pass and turn new mods API surface
   into entries here.
