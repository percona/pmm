# PMM Documentation Development Guidelines

> **Parent guide**: [AGENTS.md](../AGENTS.md) — product overview, architecture, domain model, global conventions
> **Related**: [api/AGENTS.md](../api/AGENTS.md) (protobuf behind the API reference)

This file is the single source of documentation rules. Two human-facing files keep the parts they own and this guide links to them instead of restating: [`WRITERS-NOTES.md`](WRITERS-NOTES.md) for the admonition colour table, the icon list and symbols, and [`CONTRIBUTING.md`](CONTRIBUTING.md) for the external-contributor workflow. When this guide and an older note disagree, this guide wins — the older note is corrected, never forked.

Every rule below was checked against the published pages under `docs/` rather than carried over on faith. Where a plausible-sounding rule turned out to be absent from the corpus, it is listed in [Rules that are not rules](#rules-that-are-not-rules) rather than silently dropped.

The counts quoted throughout come from one sweep of `docs/` in September 2026. They are here to show how one-sided a call was, not to track the corpus — re-measure before overturning a rule, and don't bother refreshing them in passing.

## Scope

| Path | What it is | Who writes it |
|------|-----------|---------------|
| `documentation/docs/` | User documentation, MkDocs Material | The developer shipping the change (PMM-15502); `@percona/pmm-docs` approves via CODEOWNERS |
| `documentation/docs/release-notes/` | One page per released version | Scaffolded by the `pmm-rn-create` skill, filled in on the release branch |
| `documentation/api/` | Public API reference, synced to a separate product by `.github/workflows/api-docs.yml` | Update when endpoints change; not governed by the style rules below |
| `documentation/mkdocs-base.yml` | Site config **and the `nav:` tree** | Anyone adding or moving a page |
| `documentation/variables.yml` | `{{release}}`, `{{version}}`, `{{release_date}}` macros | Release process only |
| `dev/docs/` | Developer process docs | Not user documentation — different audience, no rules here |

Dashboard panel `description` strings and PMM UI strings are in-product copy, not documentation. They live in `dashboards/` and `ui/` and are covered by those guides.

## Publishing

**A merge to `main` publishes immediately.** `.github/workflows/documentation.yml` runs `mike deploy 3 -b publish -p` on every push to `main` touching `documentation/**` (except `documentation/api/**`). There is no staging step and no release gate, so do not merge documentation for a feature that has not shipped.

CI on a documentation PR:

| Check | What it does | Blocking |
|-------|--------------|----------|
| `linkspector` (`.github/workflows/linkspector.yml`) | Resolves every link under `documentation/docs/`, config in `documentation/.linkspector.yml` | Yes — `fail_level: error` |
| `make doc-check-images` (inside `documentation.yml`) | Lists images under `docs/images/` that no page references | No — reports only; still clean up after yourself with `make doc-remove-images` |

## Layout

```
documentation/
├── docs/                    # one directory per chapter
│   ├── install-pmm/         # reworked chapter — use as the style benchmark
│   ├── quickstart/ use/ admin/ configure-pmm/ backup/ alert/ advisors/
│   ├── reference/           # dashboards, glossary, FAQ, third-party components
│   ├── release-notes/       # one file per version, newest linked from index.md
│   ├── troubleshoot/ pmm-upgrade/ uninstall-pmm/ discover-pmm/
│   └── images/ assets/ css/ js/ fonts/
├── api/                     # public API reference (separate publishing pipeline)
├── mkdocs-base.yml          # site config + nav; mkdocs.yml and mkdocs-pdf.yml INHERIT it
├── variables.yml            # release variables
├── WRITERS-NOTES.md         # admonition colours, icons, symbols
└── CONTRIBUTING.md          # contributor workflow
```

Pages carry **no YAML front matter** — not one does. Start the file with its `# H1` and nothing else.

## Writing rules

### Voice and tone

- Second person, present tense, active voice. Address the reader as "you".
- Friendly and factual. No marketing adjectives — "easy", "simple", "powerful", "robust", "seamless".
- Contractions are fine and common ("doesn't", "you'll", "isn't").
- Short sentences. Say what the reader does, then what happens.
- Use "must" for a real requirement and "can" for a real option. Don't reach for "should" or "may" when the sentence works without them.

### Page structure

- One `# H1` per page, and it is the page title. Headings step down one level at a time.
- Open with one or two sentences saying what the page covers. There is no "Introduction" or "Overview" heading — the text sits directly under the title.
- Section titles are short and sentence case: capitalize the first word and proper nouns only. Task sections take the imperative ("Configure the client"), reference and concept sections take a noun phrase ("Connection timeout settings").
- Never stack two headings with no text between them.
- `## Prerequisites` is the current heading for what the reader needs first (29 uses against 12 for "Before you start", and 22 of them in the reworked `install-pmm` chapter). `## Next steps` closes a task page with links to what follows.

### Markdown conventions

**Ordered lists** carry `{.power-number}` on its own line between the intro sentence and the list — 294 uses across 128 pages, styled by `docs/css/design.css`. Number the items explicitly (`1.`, `2.`, `3.`), not with repeated `1.`:

```markdown
To secure your system:
{.power-number}

1. First step.
2. Second step.
```

**Admonitions** use the plain Material form. New content does not add the `alert alert-*` classes: nothing in `docs/css/` or `overrides/` styles them, they are a leftover from a pre-Material theme, and the reworked `install-pmm` chapter dropped them (58 of its 75 admonitions are plain). Leave the 95 existing ones alone — both forms render identically.

```markdown
!!! note ""
    Text with no label.

!!! warning "Technical Preview status"
    A labelled warning.

??? info "Collapsed by default"
    A collapsible block.
```

Types in use: `note`, `caution`, `warning`, `hint`, `tip`, `seealso`, `danger`, `info`. Prefer `hint` to `tip` — they look the same in the Material theme and `tip` renders badly elsewhere. See [`WRITERS-NOTES.md`](WRITERS-NOTES.md#admonitions) for which colour each renders and when to reach for it.

**Tabs** (`pymdownx.tabbed`) carry per-platform or per-deployment variants:

```markdown
=== "Docker"

    Indented content.

=== "Podman"

    Indented content.
```

**Links** are relative and point at the `.md` file, including the extension — 862 relative links against 71 absolute `docs.percona.com` ones. `use_directory_urls` is `false`, so MkDocs rewrites `../reference/nomad.md#configure` to the published URL. Cross-product links are absolute.

**Code, commands, flags, paths, filenames, environment variables** go in backticks. **UI element names** go in bold and match the interface capitalization exactly: `**Query Analytics > Real-Time**`.

**Images** live in `docs/images/` and are embedded with plain `![alt](../images/name.png)` — no sizing attributes are used anywhere in the corpus. Delete the image when you delete its last reference.

**Do not hard-wrap.** One paragraph is one line; 4310 lines in the corpus are over 120 characters. Editors soft-wrap, and a hard wrap makes every later diff touch the whole paragraph.

**Variables** come from `variables.yml` through `mkdocs-macros`: `{{release}}`, `{{version}}`, `{{release_date}}`. Where a Jinja-like construct in a code block collides with the macro plugin, wrap it in `{% raw %}` / `{% endraw %}`.

**Icons** are Material theme shortcodes only (`:material-cog:`, `:fontawesome-brands-github:`). The table in [`WRITERS-NOTES.md`](WRITERS-NOTES.md#icons) lists the ones already in use. Use `→` rather than `-->`.

### Word choices

| Prefer | Over |
|--------|------|
| enables you to, lets you | allows you to |
| deprecated | legacy, old |
| removed | dropped |
| Prerequisites | Before you start |
| metadata, timestamp, filesystem | meta data, time stamp, file system |

For the version a change landed in, write "Starting with PMM X.Y.Z" — it leads "As of PMM X.Y.Z" 12 to 3, and neither "As of version X" nor "Starting from version X" appears at all.

### Adding a page

1. Write the file under the chapter it belongs to, kebab-case filename.
2. Add the `nav:` entry in `mkdocs-base.yml` — **not** `mkdocs.yml`, which only inherits from it.
3. Link to it from the pages a reader would arrive from; an unlinked page is only reachable through search.
4. Preview with `make doc-build-preview` before opening the PR.

## Release notes

Every user-facing change gets one release-note entry on the page for the version it ships in. Write it in the form it will publish in:

```markdown
## Fixed issues

- [PMM-15310](https://perconadev.atlassian.net/browse/PMM-15310): Fixed an issue where PMM Client could get stuck showing as **Disconnected** after a network interruption, with metrics and Query Analytics data no longer flowing. The connection now recovers automatically within about a minute.
```

Rules for the entry:

- It goes under one of `## Improvements` (Feature and Improvement tickets), `## Fixed issues` (Bug tickets) or `## Known issues` (a defect you are shipping knowingly).
- Improvements and fixed issues are a single bullet opening with the linked ticket key: `- [PMM-XXXXX](https://perconadev.atlassian.net/browse/PMM-XXXXX): `.
- Known issues take a `### Title ([PMM-XXXXX](url))` heading followed by prose — what the reader sees, and what to do about it.
- Write the user-visible effect, not the implementation. If the fix changes something the reader already did, say what they have to redo.
- Link to the documentation page that covers it, relative from `release-notes/` (`../reference/nomad.md`).

The version page itself — the release summary, highlights, security updates and the `nav`/`variables.yml` bump — is scaffolded by the `pmm-rn-create` skill and written by the doc team on the release branch. Don't hand-create `documentation/docs/release-notes/X.Y.Z.md`.

## Build and preview

```shell
# Live reload at http://127.0.0.1:8000/ (Docker; amd64 image)
make doc-build-preview

# One-off build, as CI runs it
make doc-build

# List images no page references (ACTION=remove to delete them)
make doc-check-images

# PDF — Percona staff only; bump the version in mkdocs-base.yml first
make doc-build-pdf
```

## Rules that are not rules

These come up repeatedly — from general technical-writing advice and from the draft skill in [percona/pmm#5334](https://github.com/percona/pmm/pull/5334) — and the published corpus contradicts them. Don't apply them, and don't reintroduce them in another copy of the style guide.

| Claimed rule | Why not |
|--------------|---------|
| Hard-wrap at 80 characters | The corpus does not wrap at all. Applying it would rewrite every paragraph you touch. |
| Say "the following table", never "above" or "below" | "table below" outnumbers "the following table" 8 to 2, and "above"/"below" appear 170+ times. Not worth a mass rewrite. |
| Release-note bug entries read `- Fixed an issue where … ([PMM-XXXXX](url))` | Published entries open with the linked key: `- [PMM-XXXXX](url): …`. |
| Use "Before you start", not "Prerequisites" | Reversed since [`WRITERS-NOTES.md`](WRITERS-NOTES.md) was written; `Prerequisites` now leads 29 to 12. |
| Check the Percona Software Support Lifecycle page on every change | Zero pages reference it. It is an external percona.com page, and only a platform-support change touches it. |
| Scaffold a release-notes page as part of writing docs | Owned by the `pmm-rn-create` skill and the release branch. |
| Add `alert alert-*` classes to admonitions | Inert — no stylesheet in this repo defines them. |
| Read a `doc-style-guide/` sibling repo before writing | No such repo is checked out or referenced by this one. This guide is the style guide. |

## Key Files to Reference

- `documentation/mkdocs-base.yml` — site config, markdown extensions, plugins, and the `nav:` tree
- `documentation/variables.yml` — `{{release}}`, `{{version}}`, `{{release_date}}`
- `documentation/WRITERS-NOTES.md` — admonition colours, icon table, symbols
- `documentation/CONTRIBUTING.md` — contributor-facing workflow and local preview
- `documentation/docs/install-pmm/` — the reworked chapter; the closest thing to a house-style reference
- `documentation/docs/release-notes/3.9.1.md` — a release notes page in its published shape
- `documentation/.linkspector.yml` — link checker config, including the ignore list
- `.github/workflows/documentation.yml` — the publish job
- `documentation/Makefile` — `doc-build`, `doc-build-preview`, `doc-check-images`
