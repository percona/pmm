# Release notes waiting for their release

Each user-facing change adds one file here, in the same PR as the change. The name says the ticket and the section:

| File | Section |
|------|---------|
| `PMM-12345-improvement.md` | 📈 Improvements (Feature and Improvement tickets) |
| `PMM-12345-fix.md` | ✅ Fixed issues (Bug tickets) |
| `PMM-12345-known-issue.md` | 🔶 Known issues. The first line is the title. |

The file holds only the text, written as it will publish:

```markdown
Fixed an issue where PMM Client could get stuck showing as **Disconnected** after a network interruption. The connection now recovers automatically within about a minute.
```

- Don't add the ticket link; it comes from the file name. See the release notes rules in [AGENTS.md](../../../AGENTS.md#release-notes).
- Write links as they appear on the release page, which sits one level up: `../images/<name>.png`, `../reference/nomad.md`.
- A note ships in the release cut from the branch it is on. A cherry-pick to `pmm-X.Y.Z` brings it along.

This folder is never published. When `pmm-X.Y.Z` is branched, the Release notes workflow builds `X.Y.Z.md` from the notes on that branch and opens a "[Doc] Release X.Y.Z" PR that adds the page and deletes the notes. Merging it deletes them from `pmm-X.Y.Z` too.

CI fails a file named any other way, an empty note, and a note already published in an earlier release. Run the check locally from the repository root: `documentation/resources/bin/release-notes.sh check my-branch documentation/docs/release-notes/unreleased/PMM-12345-fix.md`.
