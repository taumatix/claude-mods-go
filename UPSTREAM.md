# Upstream

claude-mods-go targets Claude Code's **mods** API, which ships inside Claude Code and has no
version of its own. This file says which Claude Code the bridge was built and tested against.

```yaml
- name: claude-code
  kind: npm
  package: "@anthropic-ai/claude-code"
  version: 2.1.296
  checked: 2026-10-13
  note: >-
    the end-to-end tests run `claude -p` against this release (mods docs describe 2.1.290+).
    The public type definitions lag the docs: `model.complete`'s result shape and the
    presence of `$.process.spawn` differ between releases, so typed results are lenient
    and the generated shim falls back from the daemon to the one-shot transport.

- name: mods-documentation
  kind: literal
  value: "https://claude.dev/mods/ (read 2026-10-13)"
  checked: 2026-10-13
  note: >-
    the validator rules the shim is shaped around: literal event names, `$` only ever
    written as `$.noun.method(` and never passed anywhere, relative imports only.
```
