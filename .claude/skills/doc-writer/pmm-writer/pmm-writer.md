---
name: pmm-writer
description: >
  Use this skill for ANY PMM documentation task. Triggers: "write a new
  topic", "draft a doc", "create a reference page", "write a how-to", "draft
  release notes", "document this feature", "write the overview", "write a
  deprecation notice", "write UI copy", "write a tooltip", "write microcopy",
  "write a blog post", "draft an announcement", "write a changelog entry",
  "edit this doc", "rewrite this section", "improve this draft", "fix the
  style", "clean up this", "polish this", "revise this page", "check against
  the style guide", "style review", "audit this doc", "find style issues",
  "check for passive voice", "restructure this section", "reorganize these
  topics", "split this page", "merge these pages", "where should this topic go",
  "suggest a nav structure", "review the IA", "help me plan the docs", "review
  this docs PR", "add doc inline comments", "add doc inline suggestions", "add
  doc comments with suggestions", "set up release", "prepare release", "new
  release",
  "scaffold release", "create release notes file", "set up PMM X.Y.Z",
  "prepare docs for release".
---

# PMM Documentation Writer

This skill covers every documentation task a technical writer or developer might
need on the PMM or Percona docs corpus: writing new content, editing existing
content, style checking, IA and restructuring work, UI copy, deprecation
notices, PR review, and file operations.

## Trigger phrases

Always load this skill when the user says any of the following.

**Writing new content:**
"write a new topic", "draft a doc for", "create a reference page", "write a
how-to for", "write a guide for", "draft release notes", "document this
feature", "write the overview for", "write a conceptual overview for",
"write a deprecation notice", "document the deprecation of", "write a migration
notice for", "write a removal notice for", "write UI copy for", "write a tooltip
for", "write placeholder text for", "write a button label for", "write an error
message for", "write an empty state for", "write microcopy for", "write a blog
post about", "draft an announcement for", "draft a release announcement",
"write a changelog entry", "write a what's new post for"

**Editing existing content:**
"edit this doc", "rewrite this section", "improve this draft", "fix the style
in", "clean up this content", "polish this", "update this topic for version X",
"revise this page", "rephrase this", "reword this", "improve this tooltip",
"shorten this label"

**Style and quality checks:**
"check this against the style guide", "style review", "audit this doc", "find
style issues in", "check for passive voice", "check headings", "is this on
brand?", "does this follow our standards?", "what's wrong with this draft?",
"review this for style", "give me a style audit"

**Restructuring and IA:**
"restructure this section", "reorganize these topics", "split this page into",
"merge these pages", "where should this topic go?", "suggest a nav structure
for", "review the IA for", "help me plan the docs for this feature", "what
topics do I need for", "what's the right structure for"

**PR review:**
"review this docs PR", "review this documentation pull request", "add doc
comments with suggestions", "give me feedback on this docs PR", "what do you
think of this docs PR", "add doc inline comments", "add doc inline
suggestions", "leave doc suggestions directly on the files", "add doc
suggestions directly in the file", "comment on the docs diff", "add doc line
comments"

**Jira-driven doc work:**
"check this ticket", "does PMM-XXXXX need docs", "what docs does this ticket
need", "review this Jira for docs", "does this Jira impact documentation",
"generate release notes for this ticket", "write release notes for PMM-XXXXX",
"document this Jira", "what release notes entry does this need"

**New release setup:**
"set up release", "prepare release", "new release", "scaffold release",
"create release notes file", "set up PMM X.Y.Z", "prepare docs for release"

## Step 0 — New release setup

**When to run this step:** Run when the user asks to set up, scaffold, or
prepare documentation for a new PMM release (e.g. "set up release", "prepare
docs for PMM 3.8.0", "new release").

### 0.1 — Gather inputs

If the user has not already provided them, ask:

1. **New version number** — e.g. `3.8.0`
2. **Previous version number** — e.g. `3.7.1` (the one currently at the top of
   the nav)
3. **Release date** — in `YYYY-MM-DD` format (e.g. `2026-07-01`)

Do not proceed until you have all three values.

### 0.2 — Make the following changes

Once you have the version and date, update these three files without asking for
further confirmation:

**1. `docs/release-notes/index.md`**
Add a new bullet at the top of the list:
```markdown
- [Percona Monitoring and Management X.Y.Z](X.Y.Z.md)
```

**2. `mkdocs-base.yml`**
Add a new nav entry directly above the previous version's entry in the
Release notes section:
```yaml
- "PMM X.Y.Z (YYYY-MM-DD)": release-notes/X.Y.Z.md
```

**3. `mkdocs-pdf.yml`**
Update the `with-pdf` plugin block:
```yaml
output_path: "pdf/PerconaMonitoringAndManagement-X.Y.Z.pdf"
cover_subtitle: X.Y.Z (Month D, YYYY)
```

### 0.3 — Create the release notes stub

Create a new file at `docs/release-notes/X.Y.Z.md` using the template below.
Fill in the version number and release date. Leave all content sections empty
for the writer to populate.

```markdown
# Percona Monitoring and Management X.Y.Z

**Release date**: Month Dth, YYYY

Percona Monitoring and Management (PMM) is an open source database monitoring, management, and observability solution for MySQL, PostgreSQL, MongoDB, Valkey and Redis. PMM empowers you to: 

- monitor the health and performance of your database systems
- identify patterns and trends in database behavior
- diagnose and resolve issues faster with actionable insights
- manage databases across on-premises, cloud, and hybrid environments

## 📋 Release summary


## ✨ Release highlights

### 
## 📦 Components upgrade 



## 🔒 Security updates
PMM X.Y.Z fixes the following security vulnerabilities:
## 📈 Improvements

## ✅ Fixed issues


## 🚀 Ready to upgrade to PMM X.Y.Z?

- [New installation](../quickstart/quickstart.md)
- [Upgrading from PMM 3](../pmm-upgrade/index.md)
- [Upgrading from PMM 2](../pmm-upgrade/migrating_from_pmm_2.md)
```

### 0.4 — Confirm

After making all changes, report:
- Which files were updated and what changed in each
- The path to the new release notes stub

---

## Step 1 — Read the Jira ticket and assess doc impact

**When to run this step:** Run this step whenever the user provides a Jira
ticket ID (e.g., `PMM-12345`) or asks whether a ticket requires documentation.
If the user did not provide a ticket ID, ask for one before continuing.

### 1.1 — Fetch the ticket

Use the Atlassian MCP tool `mcp__atlassian__getJiraIssue` with the ticket key
(e.g., `PMM-12345`). The Jira board is at
`https://perconadev.atlassian.net/browse/PMM`.

Read from the response:
- **Summary** — the one-line ticket title
- **Issue type** — Bug, Story, Task, Epic, Improvement, Sub-task
- **Status** — to-do, in progress, done, closed, etc.
- **Fix version(s)** — the PMM release this lands in
- **Labels** — look for `docs`, `no-docs`, `breaking-change`, `deprecation`
- **Description** — the full ticket description (what changed, why, user impact)
- **Linked issues** — related tickets, parent epics, or duplicates

### 1.2 — Decide if the ticket impacts documentation

A ticket **requires documentation** if any of the following are true:

| Signal | Examples |
|--------|---------|
| New user-visible feature | New UI panel, new CLI flag, new API endpoint, new integration |
| Changed user-visible behavior | Default value changed, workflow changed, output format changed |
| Bug fix that changes behavior users relied on | Workarounds in existing docs become invalid |
| Deprecation or removal | Any feature, flag, endpoint, or platform being deprecated or removed |
| New or dropped platform/OS support | Added RHEL 9, dropped Ubuntu 18.04 |
| Breaking change | Anything requiring user action on upgrade |
| Security fix with user-visible impact | Changed configuration, new requirement, advisory |

A ticket **does not require documentation** if:
- It is a pure internal refactor with no user-visible change
- It is a test, CI, or build-only change
- It is a performance improvement with no behavior change
- It is already covered by a parent epic that has its own doc ticket
- The label `no-docs` is present

**If the ticket does not require documentation:** Tell the user which signal(s)
led to this conclusion and confirm. If uncertain, present your reasoning and ask
before proceeding.

### 1.3 — If the ticket requires documentation: generate a release notes entry

Identify the entry type based on the issue type and description:

| Ticket type | Entry format |
|-------------|-------------|
| Bug fix | Single bullet line (see template below) |
| New feature, Story, Epic | Named section with H3 heading (see template below) |
| Improvement | Named section if significant; single bullet if minor |
| Deprecation / removal | Named section with deprecation warning admonition |
| Breaking change | Named section, flag prominently |

Draft the release notes entry using the templates in the
[Reference: content type templates](#reference-content-type-templates) section.
Fill in from the ticket:
- The ticket ID and link (`PMM-XXXXX` → `https://perconadev.atlassian.net/browse/PMM-XXXXX`)
- Fix version from the ticket's **Fix Version** field
- A user-focused description (what changed, what it enables — not internal
  implementation details)

**Output to the user:**
1. A one-line verdict: "This ticket **requires** / **does not require**
   documentation."
2. The reasoning (which signals triggered the decision).
3. If it requires docs: the draft release notes entry in a code block.
4. Ask whether the user also wants a fuller doc topic (how-to, reference, etc.)
   drafted for this ticket, then continue with the appropriate steps below.

---

## Step 2 — Read the style guide

Before writing or editing anything, read the relevant sections of the style
guide at:

```
../doc-style-guide/docs/
```

**Always read these core files first:**
- `voice-and-tone.md` — tone, person, tense, active voice rules
- `headings.md` — imperative mood, sentence case, no stacked headings
- `structure.md` — how to organize content, notes, forward references
- `lists.md` — intro sentences, punctuation, length, nesting rules
- `capitalization.md` — sentence-style headings, UI element matching

**Read these based on the content type requested:**

| Content type | Additional files to read |
|---|---|
| Reference (CLI, config, API) | `rules-for-docs.md`, `code.md`, `cross-reference.md` |
| How-to / task guide | `modes.md`, `grammar.md`, `markup.md` |
| Conceptual / overview | `paragraphs.md`, `text.md`, `grammar.md` |
| Release notes | `structure-release-notes.md`, `word-use.md` |
| UI copy / microcopy | `voice-and-tone.md`, `capitalization.md` |
| Deprecation notices | `structure.md`, `word-use.md` |
| Blog posts / announcements | `voice-and-tone.md`, `word-use.md` |

Use `Read` to read each file. Load only what the task requires — do not load
all files for every request.

## Step 3 — Scan all PMM docs for context and tone calibration

The PMM docs live at:

```
documentation/docs/
```

**Do this every time you are writing or editing content, without exception:**

1. Run `Glob` on the docs root to get the full directory tree
2. Scan subdirectory listings to find any topics related to the content you are
   about to write or edit
3. Read the relevant existing files to:
   - Understand what is already documented about this topic
   - Identify where the new content should live (new file, new section in an
     existing file, new subsection)
   - Calibrate tone and structure from real examples in context

**Prioritize these as tone/structure benchmarks** (read first if relevant):

| Reference doc | Path | Best for |
|---|---|---|
| Release notes | `release-notes/3.6.0.md` | Release notes tone, section order, highlight framing |
| Get Started | `quickstart/quickstart.md` | Task-based how-to tone, numbered steps, tabbed content |
| Back up and restore | `backup/index.md` | Conceptual overview tone, intro structure |

But do not stop there — if you find a more directly relevant existing topic,
read that too and use it as the primary reference. Use `Read` with `offset` and
`limit` to read specific sections of long files rather than the full file.

**Also check the MkDocs navigation config:**

Read `mkdocs.yml` (or `mkdocs-base.yml` if it exists) at the documentation
root:

```
documentation/
```

Find the `nav:` section. Check where the new topic should appear in the nav
tree, whether a new nav entry is needed or the content slots into an existing
section, and whether any existing nav entries need reordering or renaming.

**Also check the Percona Software Support Lifecycle page:**

Fetch `https://www.percona.com/services/policies/percona-software-support-lifecycle`
and check whether the new content involves anything that would require an update
there:

- New platform or OS support added (e.g., a new RHEL, Debian, or Ubuntu version)
- A platform being deprecated or removed
- A new PMM major or minor version reaching GA
- End-of-life dates changing for any supported component

**Before drafting, tell the user:**
- What related content already exists and where
- Your recommendation for where the new file should go (suggested filename
  following existing naming conventions)
- Where it should sit in the `nav:` tree, with the exact YAML nav entry to add
- Whether the support lifecycle page needs updating, and specifically what to
  change
- Whether any existing docs need updating or cross-referencing once the new
  content is added

## Step 4 — Identify content type and gather inputs

Ask the user for any missing inputs before drafting. For ambiguous requests,
identify the content type first, then ask only for what is missing.

**For all content types:**
- What is the feature, command, or topic being documented?
- Who is the target user (admin, DBA, developer)?
- Any specific details, flags, steps, or context to include?

**For reference topics:** flag names, types, defaults, examples  
**For how-to guides:** the goal the user wants to achieve, prerequisites,
numbered steps  
**For conceptual topics:** what the feature does, why it matters, what it
enables  
**For release notes:** version, release date, ticket IDs, what changed and why  
**For UI copy:** the UI element type, its location in the interface, the action
it triggers, character limits if known  
**For deprecation notices:** what is being deprecated, the version it was
deprecated in, the version it will be removed in, the migration path  
**For blog posts / announcements:** the audience, the top 2–3 changes to
highlight, links to related docs  
**For editing tasks:** the file path or pasted content, the scope of edits
(style only, restructure, full rewrite), any constraints  
**For style audits:** the content to audit (pasted or file path), content type  
**For restructuring tasks:** the scope (a section, a chapter, the whole nav),
any known goals or constraints  

## Step 5 — Draft, edit, or review the content

Apply these rules consistently — they are non-negotiable for all content types.

### Voice and tone
- Second person ("you"), present tense, active voice
- Friendly and professional — not playful, not promotional
- No "easy", "simple", "powerful", "robust", or subjective attributes
- No modal verbs (must, may, could, should) unless they add real meaning
- Contractions are acceptable (isn't, you'll, it's)
- Keep sentences short. Max two clauses per sentence.

### Headings
- Sentence case — only capitalize the first word and proper nouns
- Imperative mood for task topics ("Connect a database", "Configure the client")
- Noun phrase for reference and conceptual topics ("Connection timeout settings")
- No heading immediately followed by another heading (no stacked headings)
- No period or colon at the end of a heading
- Only one H1 per document

### Structure
- Start with what the user can do or what the topic covers — no preamble
- Introduce the topic in 1–3 sentences before any list or table
- Lists: 2–7 items, parallel structure, intro sentence ending in colon
- Use numbered lists for ordered steps, bullet lists for unordered items
- Notes use `!!! note` admonition syntax (MkDocs Material)
- Warnings use `!!! warning "Label"` with a descriptive label in quotes (e.g.,
  `!!! warning "Technical Preview status"`)

### Formatting
- Code, commands, flags, paths, and filenames in backtick code spans
- UI element names match the interface capitalization exactly, shown in **bold**
- Tabbed content with `=== "Tab name"` syntax for multi-platform steps
- Line wrap at 80 characters (soft guideline, not strict for drafts)

### Word choices
- "enables you to" not "allows you to"
- "As of version X" not "Starting from version X"
- metadata, timestamp, filesystem (one word each)
- Reference tables/figures as "the following table" / "in the previous example"
  — not "above" or "below"
- "deprecated" not "legacy" or "old"
- "removed" not "dropped" or "killed"

### Additional rules by task type

**When editing existing content:**
- Do not change technical meaning or invent details the user did not provide
- If something is technically ambiguous, flag it rather than guess
- Apply all style rules above as fixes, not suggestions
- Return the revised content in a code block, then an **Edit notes** section
  listing every change made and the rule it applied

**When doing a style audit:**
- Return a numbered list of issues. For each: quote the offending text, name
  the rule it breaks, give a corrected version
- End with a **Summary** line: total issues found and the main patterns
- Do not return a corrected full draft unless the user asks for one

**When restructuring:**
- Read all files in the scope before proposing changes
- Identify what to keep, split, merge, rename, or move
- Propose a revised nav YAML snippet for the affected section
- Flag content gaps — topics that should exist but do not
- Do not make file changes until the user approves the structure

**When writing UI copy:**
- Tooltips: one sentence or noun phrase, present tense, no period if a noun
  phrase
- Button labels: verb + noun ("Save changes", "Add dashboard")
- Placeholder text: concise example value or hint, not instructions
- Error messages: what went wrong + what to do next, no blame language
- Empty states: what is empty + one action to get started
- Return up to three variants if constraints are ambiguous, with a
  recommendation and rationale

**When writing deprecation notices:**
- Always identify every location where the notice should appear: the feature's
  own page, the release notes entry, and any overview pages that mention it
- Return the notice draft, the list of files to update, and the release notes
  entry text as separate items

**When writing blog posts / announcements:**
- Allowed to open with a framing sentence rather than a direct action, but no
  promotional language or superlatives
- Structure: opening paragraph → H2 per feature highlight → upgrade/get started
  section → sign-off with link to release notes

## Step 6 — Output the draft and nav update

Present two things in the chat:

**1. The Markdown draft** in a code block. Do not save to file until the user
approves.

**2. The nav update** — the exact YAML snippet to add to `mkdocs.yml`, showing
the full parent path for context:

```yaml
# Add under: [parent section > subsection]
- New topic title: path/to/new-file.md
```

If the content goes into an existing file rather than a new one, show the
heading placement instead (e.g., "Add as a new H2 under `## Supported
configurations` in `backup/index.md`").

After both, add a **Draft notes** section (plain text) flagging:
- Any assumptions made (inferred flag defaults, assumed audience, etc.)
- Placeholders needing real values (ticket IDs, version numbers, flag defaults)
- Related pages that should link to this new content
- Whether the [Percona Software Support Lifecycle](https://www.percona.com/services/policies/percona-software-support-lifecycle)
  page needs updating and what to change — or explicitly confirm it does not
  apply
- Any style decisions worth calling out

Wait for user feedback before saving or iterating.

**File operations:**

Only write or modify files when the user explicitly confirms the content is
approved and says to save or apply it. The workflow is always: draft in chat
→ user approves → write to file.

When writing a new file:
1. Confirm the target path follows existing naming conventions (kebab-case,
   descriptive, matches the nav entry)
2. Use `Write` to create the file at the approved path
3. Confirm the file was created and give the full path
4. Remind the user to add the nav entry to `mkdocs.yml` and show the exact YAML
   snippet again

When editing an existing file:
1. Use `Read` to read the current file immediately before making any changes
2. Use `Edit` for targeted edits; use `Write` only if rewriting the whole file
3. After editing, use `Read` to verify the change is correct
4. Report what changed and the current state of the file

---

## PR review: general comment

**Trigger phrases:** "review this docs PR", "review this documentation pull
request", "add doc comments with suggestions", "give me feedback on this docs
PR"

1. Fetch the PR diff using the GitHub MCP tool (`pull_request_read` with
   `get_diff`)
2. Read the relevant style guide files from Step 2 based on content type
3. Review the diff against all style rules in Step 5
4. Post a single comment using `add_issue_comment` with:
   - A one-sentence positive summary of what works
   - Numbered issues, each with: quoted offending text, rule broken, suggested
     rewrite
   - A brief **Positives** section at the end
5. Do not post the comment until the full review is complete

---

## PR review: inline suggestions

**Trigger phrases:** "add doc inline comments", "add doc inline suggestions",
"leave doc suggestions directly on the files", "add doc suggestions directly
in the file", "comment on the docs diff", "add doc line comments"

1. Fetch the PR diff (`pull_request_read` with `get_diff`) and note the head
   commit SHA
2. Fetch the changed file(s) using `get_file_contents` with
   `ref: refs/pull/{number}/head` to get exact line numbers
3. Map each style issue to the exact line number in the new file version
4. Use `gh api repos/{owner}/{repo}/pulls/{number}/reviews` via Bash with:
   - `commit_id`: the head commit SHA
   - `event`: `"COMMENT"`
   - `comments`: array of objects with `path`, `line`, `side: "RIGHT"`, `body`
   - For direct text fixes, use a suggestion fenced block so GitHub renders a
     one-click apply button:

     ````
     ```suggestion
     corrected line here
     ```
     ````
5. Only comment on lines present in the diff (added or context lines)
6. Do not fall back to a general comment if inline comments were requested — use
   the `gh api` approach directly

---

## Reference: content type templates

### Release notes entry — bug fix
```markdown
- Fixed an issue where [description of problem]. ([PMM-XXXXX](https://perconadev.atlassian.net/browse/PMM-XXXXX))
```

### Release notes entry — new feature or improvement
```markdown
### Feature or improvement title

[1–2 sentences: what changed and what it enables for the user.]

Key changes:
- [change 1]
- [change 2]
```

### How-to guide
```markdown
# Do [task]

[1–2 sentence intro: what this guide helps you do and why.]

## Prerequisites

Before you start, make sure that:

- [prerequisite 1]
- [prerequisite 2]

## [Step group heading]

1. [Step 1]
2. [Step 2]

## Next steps

- [Link to related task]
```

### Reference topic (CLI, config, API)
```markdown
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
```

### Conceptual overview
```markdown
# [Topic name]

[2–3 sentence intro: what this feature is and what problem it solves.]

## How it works

[Explanation without jargon. Present tense, active voice.]

## Supported configurations

[Table or list of what is and is not supported.]

## Next steps

- [Link to task topic]
```

### Deprecation notice (inline warning)
```markdown
!!! warning "Deprecated: [Feature name]"

    [Feature name] is deprecated as of PMM [version] and will be removed in
    PMM [version or timeframe].

    To continue [doing X], [migration action]. For details, see [link].
```

### UI tooltip
```
[Verb phrase or noun phrase describing what this setting does, under 15 words.]
```

### Error message
```
[What went wrong.] [What to do next.]
```
