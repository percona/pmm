# AI agent guides

Guidance for AI agents lives in `AGENTS.md` files: the root [`AGENTS.md`](../../../AGENTS.md) for the whole product, plus one guide per component. They follow the cross-tool `AGENTS.md` convention, so most agents load them automatically. Some tools read a thin pointer file that routes them there instead:

| Tool | Pointer file |
|------|--------------|
| Claude Code | `CLAUDE.md` at the root and next to every component `AGENTS.md` |
| Gemini CLI | `GEMINI.md` |
| GitHub Copilot | `.github/copilot-instructions.md` |

Claude Code reads `CLAUDE.md`, not `AGENTS.md`, so every directory holding a component guide also carries a `CLAUDE.md` whose only line is `@AGENTS.md`. Claude Code loads a nested `CLAUDE.md` when it first reads a file in that directory, so a component guide enters context only for the component being worked on. Keep the pointers to that one import; guidance belongs in `AGENTS.md`.

Cursor loads root and nested `AGENTS.md` files natively, so it needs no rule file of its own.

Personal AI-tool files (`.claude/`, `.cursor/`) are gitignored for local experimentation, except `.claude/settings.json`, the committed team configuration.
