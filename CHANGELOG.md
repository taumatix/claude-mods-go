# Changelog

## [Unreleased]

## [0.1.0] - 2026-10-13

- `mods`: write a Claude Code mod in Go. Typed handlers for six events, helpers for the common
  `$` calls, `Ctx.Call` for the rest, `Ctx.Next` for `next(e)`.
- `modgen` and `modgo build`: generate the plugin (manifest, hooks.json, JavaScript shim, binaries).
- Two transports behind one shim: a long-lived loopback daemon and a per-step one-shot with journal
  replay; `auto` tries the daemon and falls back.
