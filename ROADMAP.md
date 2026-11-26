# Roadmap

Ordered by how much each entry limits a real deployment.

1. **A live deny and a live `tool.check`.** `tool.call` rewrite is now driven end to end through a
   command's `$.tool.call`; denying a call and `tool.check` answers are still unit-tested only.
2. **`claude plugin test` support.** It stubs `$.process.*`, so a Go mod cannot use it; ship a
   test double for the bridge or a `modgo test` that runs the live path.
3. **Bridge more of `$`.** Callback methods (`$.clock.every`, `$.state`), `$.env` and the drawing
   calls (`$.ui.open`) are not bridged.
4. **Track upstream.** Re-check `UPSTREAM.md` each maintenance pass and turn new mods API surface
   into entries here.
