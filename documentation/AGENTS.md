# PMM Documentation Development Guidelines

> **Parent guide**: [AGENTS.md](../AGENTS.md) — product overview, architecture, domain model, global conventions
> **Related**: [api/AGENTS.md](../api/AGENTS.md) (protobuf behind the API reference)

This file is the single source of documentation rules. Two human-facing files keep the parts they own and this guide links to them instead of restating: [`WRITERS-NOTES.md`](WRITERS-NOTES.md) for the admonition colour table, the icon list and symbols, and [`CONTRIBUTING.md`](CONTRIBUTING.md) for the external-contributor workflow. When this guide and an older note disagree, this guide wins — the older note is corrected, never forked.

Every rule below was checked against the published pages under `docs/` rather than carried over on faith. Where a rule deliberately departs from what is published today, it says so and tells you not to sweep the old pages.

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

**The site is published from the release branch of the latest GA version, not from `main`.** `.github/workflows/documentation.yml` runs `mike deploy 3 -b publish -p` from `pmm-<latest GA>` when that branch gets a push touching `documentation/**` (except `documentation/api/**`), and when its `v<latest GA>` tag is created. Documentation merged to `main` goes live with the release that ships it; the edit button on the site opens the file on that release branch.

To fix a live page, open the PR against `pmm-<latest GA>`, then bring the commit to `main` with `git cherry-pick -x <sha>`. The `Documentation fixes reached main` job in the same workflow fails, on each release-branch push and every weekday, until every documentation commit made on the release branch since GA is on `main`.

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
- For task steps, use the imperative: "Click **Save**", not "You should click **Save**".
- Passive voice is acceptable when the system performs the action ("The index is created automatically"), when the focus belongs on the receiver rather than the actor, or when active voice would read as blaming the user for an error.
- Friendly and factual. No marketing adjectives — "easy", "simple", "powerful", "robust", "seamless".
- Emotionally neutral. PMM documentation reaches a multicultural audience. Avoid humor, idioms, and culturally specific references that do not translate.
- Contractions are fine and common ("doesn't", "you'll", "isn't").
- Short sentences. Max two clauses per sentence; aim for 15–20 words. Say what the reader does, then what happens.
- Put conditional clauses before the instruction: "If the service is running, click **Stop**" — not "Click **Stop** if the service is running".
- Organize ideas from common to specific, known to unknown.
- Use "must" for a real requirement and "can" for a real option. Remove "should" and "may" when the sentence means the same thing without them.
- Use one term per concept consistently. Don't vary terminology for style; readers and search both depend on consistent names.
- Repeat the technical noun rather than replacing it with "it", "this", or "they" when the referent could be ambiguous.
- Include "that" when it introduces a subordinate clause — don't drop it for brevity ("Ensure that the service is running", not "Ensure the service is running").
- Avoid stacking multiple adjectives before a noun ("new advanced configurable settings" → "advanced settings that you can configure").
- Define a technical term or abbreviation at its first occurrence: spell it out, then put the abbreviation in parentheses.
- Lead with the primary use case. Cover advanced variations and edge cases after the main flow works.
- When a step has a verifiable outcome, say what the reader should see: "The **Status** column shows **Connected**".

### Page structure

- One `# H1` per page, and it is the page title. Headings step down one level at a time.
- Open with 1–3 sentences saying what the page covers. There is no "Introduction" or "Overview" heading — the text sits directly under the title.
- Introduce a topic with 1–3 sentences before the first list or table.
- Section titles are short and sentence case: capitalize the first word and proper nouns only. Task sections take the imperative ("Configure the client"), reference and concept sections take a noun phrase ("Connection timeout settings").
- Never stack two headings with no text between them.
- Don't end a heading with a period or colon.
- `## Prerequisites` is the heading for what the reader needs before starting. Plenty of existing pages say "Before you start"; change them on a page you are already editing, don't sweep them. `## Next steps` closes a task page with links to what follows.

### Markdown conventions

**Lists** — general rules that apply to both ordered and bullet lists:

- Every list has an intro sentence ending with a colon.
- Keep lists to 2–7 items. Fewer than two items usually reads better as prose; more than seven should be split or reorganized.
- All items use parallel grammatical structure — start them the same way (all verbs, all nouns, etc.).
- Apply sentence-ending punctuation only when every item is a complete sentence; omit it otherwise.
- Maximum two nesting levels. Sub-lists need at least two items.
- Use numbered lists when order matters, bullet lists when it does not. When in doubt, use a numbered list.

**Ordered lists** require an intro sentence followed by `{.power-number}` on its own line, then a blank line, then the list items. `docs/css/design.css` keys PMM's stepped-list styling off that class by targeting `{.power-number}+ol`, so the class only works when directly preceded by the intro sentence — a list without it, or with the class elsewhere, renders as a plain `<ol>`. Number the items explicitly (`1.`, `2.`, `3.`), not with repeated `1.`:

```markdown
To secure your system:
{.power-number}

1. First step.
2. Second step.
```

**Admonitions** use the plain Material form. New content does not add the `alert alert-*` classes: nothing in `docs/css/` or `overrides/` styles them, they are a leftover from a pre-Material theme, and the reworked `install-pmm` chapter dropped them. Leave the existing ones alone — both forms render identically.

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

**Links** between pages are relative and point at the `.md` file, including the extension. `use_directory_urls` is `false`, so MkDocs rewrites `../reference/nomad.md#configure` to the published URL. Only cross-product links are absolute.

**Code, commands, flags, paths, filenames, environment variables** go in backticks. **UI element names** go in bold and match the interface capitalization exactly: `**Query Analytics > Real-Time**`.

**Images** live in `docs/images/` and are embedded with plain `![alt](../images/name.png)` — no sizing attributes are used anywhere in the corpus. Delete the image when you delete its last reference.

**Do not hard-wrap** — not at 80 columns, not at any width. One paragraph is one line. Editors soft-wrap, and a hard wrap makes every later diff touch the whole paragraph.

**Variables** come from `variables.yml` through `mkdocs-macros`: `{{release}}`, `{{version}}`, `{{release_date}}`. Where a Jinja-like construct in a code block collides with the macro plugin, wrap it in `{% raw %}` / `{% endraw %}`.

**Icons** are Material theme shortcodes only (`:material-cog:`, `:fontawesome-brands-github:`). The table in [`WRITERS-NOTES.md`](WRITERS-NOTES.md#icons) lists the ones already in use. Use `→` rather than `-->`.

### Word choices

| Prefer | Over |
|--------|------|
| enables you to | allows you to, lets you. These imply permission; "enables" describes a capability |
| deprecated | legacy, old |
| removed | dropped |
| Prerequisites | Before you start |
| metadata, timestamp, filesystem | meta data, time stamp, file system |
| the following table, the previous example | the table below, the example above |

Point at content by its place in the reading order, not by where it lands on screen — "below" and "above" stop being true in the PDF, in a narrow viewport, and to a screen reader. Existing pages are full of them: fix the ones on a page you are already editing, don't open a PR to sweep them.

For the version a change landed in, write "Starting with PMM X.Y.Z" — not "As of PMM X.Y.Z", "As of version X" or "Starting from version X".

### Adding a page

1. Write the file under the chapter it belongs to, kebab-case filename.
2. Add the `nav:` entry in `mkdocs-base.yml` — **not** `mkdocs.yml`, which only inherits from it.
3. Link to it from the pages a reader would arrive from; an unlinked page is only reachable through search.
4. Preview with `make doc-build-preview` before opening the PR.

## Release notes

Every user-facing change gets one release-note entry on the page for the version it ships in. Write it in the form it will publish in:

```markdown
## ✅ Fixed issues

- [PMM-15310](https://perconadev.atlassian.net/browse/PMM-15310): Fixed an issue where PMM Client could get stuck showing as **Disconnected** after a network interruption, with metrics and Query Analytics data no longer flowing. The connection now recovers automatically within about a minute.
```

Rules for the entry:

- It goes under one of `## 📈 Improvements` (Feature and Improvement tickets), `## ✅ Fixed issues` (Bug tickets) or `## Known issues` (a defect you are shipping knowingly). The emoji are part of the heading — match the scaffolded template exactly. Other top-level sections in the template: `## 📋 Release summary`, `## 🔒 Security updates`, `## 🚀 Ready to upgrade to PMM X.Y.Z`.
- Improvements and fixed issues are a single bullet opening with the linked ticket key: `- [PMM-XXXXX](https://perconadev.atlassian.net/browse/PMM-XXXXX): `.
- Known issues take a `### Title ([PMM-XXXXX](url))` heading followed by prose — what the reader sees, and what to do about it.
- Write the user-visible effect, not the implementation. If the fix changes something the reader already did, say what they have to redo.
- Link to the documentation page that covers it, relative from `release-notes/` (`../reference/nomad.md`).

The version page itself — the release summary, highlights, security updates and the `nav`/`variables.yml` bump — is scaffolded by the `pmm-rn-create` skill and written by the doc team on the release branch. Don't hand-create `documentation/docs/release-notes/X.Y.Z.md`.

## Content type templates

Use these as starting structures. Fill in the placeholders; don't ship the bracket labels.

### How-to guide

```markdown
# Do [task]

[1–2 sentence intro: what this guide helps you do and why.]

## Prerequisites

Before you start, make sure that:

- [prerequisite 1]
- [prerequisite 2]

## [Step group heading]

[Intro sentence]:
{.power-number}

1. [Step 1.] [Expected outcome if non-obvious.]
2. [Step 2.]

## Next steps

- [Link to related task]
```

### Reference topic (CLI, config, API)

~~~markdown
# [Command or setting name]

[1 sentence: what this command or setting does.]

## Syntax

```bash
[syntax example]
```

## Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `--flag` | string | `""` | [Description.] |

## Examples

[Intro sentence]:

```bash
[example]
```
~~~

### Conceptual overview

```markdown
# [Topic name]

[2–3 sentence intro: what this feature is and what problem it solves.]

## How it works

[Explanation. Present tense, active voice.]

## Supported configurations

[Table or list of what is and is not supported.]

## Next steps

- [Link to task topic]
```

### Deprecation notice

```markdown
!!! warning "Deprecated: [Feature name]"

    [Feature name] is deprecated starting with PMM [version] and will be removed in PMM [version or timeframe].

    To continue [doing X], [migration action]. For details, see [link].
```

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
