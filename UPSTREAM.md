# Upstream

claude-mods-go targets Claude Code's **mods** API, which ships inside Claude Code and has no
version of its own. This file says which Claude Code the bridge was built and tested against.

```yaml
- name: claude-code
  kind: npm
  package: "@anthropic-ai/claude-code"
  version: 2.1.325
  checked: 2026-11-16
  note: >-
    the end-to-end tests run `claude -p` against this release (mods docs describe 2.1.290+); re-run green against 2.1.325 on 2026-11-13.
    The public type definitions lag the docs: `model.complete`'s result shape and the
    presence of `$.process.spawn` differ between releases, so typed results are lenient
    and the generated shim falls back from the daemon to the one-shot transport.

- name: mods-documentation
  kind: literal
  value: "https://code.claude.com/docs/en/plugins/mods/create.md (read 2026-11-16)"
  checked: 2026-11-16
  note: >-
    the validator rules the shim is shaped around: literal event names, `$` only ever
    written as `$.noun.method(` and never passed anywhere, relative imports only.
    Compared against the docs text on 2026-11-16 (section "Check what Claude Code reads
    from your mod"): event names must be string literals (matches); `$` must be written in
    full as `$.namespace.method(`, never assigned, destructured or indexed, but may be passed
    to a top-level function of the same file (the shim's stricter rule is safe); imports must
    be relative files inside the plugin, plus the bare `claude-code` (the shim is stricter);
    dynamic `import()` fails, and `on` must not be shadowed inside `register`. The earlier
    URL https://claude.dev/mods/ is only a landing page that links to the docs.
```
