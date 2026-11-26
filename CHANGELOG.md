# Changelog

## [Unreleased]

- `mods`: typed events and `OnX` methods for the remaining hooks: `session.end`, `session.stop`,
  `session.compact`, `turn.start`, `prompt.fill`, `prompt.suggest`, `tool.describe`, `tool.offer`,
  `command.describe`, `config.set`, `agent.spawn`, `agent.offer`. The ones with a natural filter
  (`OnToolDescribe`, `OnToolOffer`, `OnCommandDescribe`, `OnConfigSet`, `OnAgentSpawn`,
  `OnAgentOffer`) take it as their first argument; `""` hooks all. Shared `Origin` and `Source` types.
- `mods`: the daemon exits after 30 minutes idle (`App.IdleTimeout`, `DefaultIdleTimeout`), so a
  daemon left behind by a dead session no longer lingers. The shim starts a new one on the next
  event.
- Windows support: `modgo build -targets windows/amd64` (or `arm64`) writes `<name>.exe`; the shim
  turns `file:///C:/...` into a usable path, picks the architecture without `uname`, and runs the
  `.exe`. New `modgen.ExeName`. The end-to-end suite runs on `windows-latest` in CI.

## [0.1.0] - 2026-10-13

- `mods`: write a Claude Code mod in Go. Typed handlers for six events, helpers for the common
  `$` calls, `Ctx.Call` for the rest, `Ctx.Next` for `next(e)`.
- `modgen` and `modgo build`: generate the plugin (manifest, hooks.json, JavaScript shim, binaries).
- Two transports behind one shim: a long-lived loopback daemon and a per-step one-shot with journal
  replay; `auto` tries the daemon and falls back.
