# Release notes waiting for their release

Each user-facing change adds one file here, named after its ticket: `PMM-12345.md`.

```markdown
---
target: 3.11.0
type: fix
---
Fixed an issue where PMM Client could get stuck showing as **Disconnected** after a network interruption. The connection now recovers automatically within about a minute.
```

- `target` is the release the change ships in. It has no default: choose it.
- `type` picks the section of the release page:

  | type | Section |
  |------|---------|
  | `improvement` | 📈 Improvements (Feature and Improvement tickets) |
  | `fix` | ✅ Fixed issues (Bug tickets) |
  | `known-issue` | 🔶 Known issues |

- A `known-issue` also needs a `title:`, which becomes the `###` heading.
- Release highlights and security updates are written by the doc team on the release page, not as notes.
- Write the text in the form it will publish in, following the release notes rules in [AGENTS.md](../../../AGENTS.md#release-notes). Don't add the ticket link; it is generated from the file name.
- Write links as they appear on the release page, which sits one level up: `../images/<name>.png`, `../reference/nomad.md`.

This folder is never published. When `pmm-<target>` is branched, the Release notes workflow builds `<target>.md` from the notes on that branch and opens a "[Doc] Release <target>" PR that adds the page and removes the notes from `main`.

CI checks every note in a PR:

- On `main`, the target must not be branched yet. Once `pmm-3.11.0` exists, a 3.11.0 note goes in the PR to `pmm-3.11.0`; the same change on `main` needs no note.
- On a `pmm-X.Y.Z` branch, the target must be `X.Y.Z`.

Run the same check locally, from the repository root: `documentation/resources/bin/release-notes.sh check main my-branch documentation/docs/release-notes/unreleased/PMM-12345.md`.
