# Writer's Notes

## Formatting

**Line wrapping**

Most files don't use line wrapping. Each paragraph or sentence is a complete string of text without newline characters. The rationale is that most viewers and editors have configurable soft-wrap abilities, and every author tends to choose a different hard-wrap column.

## Admonitions

Admonitions use an MkDocs definition to get acceptable rendering on Render.com.

Material for MkDocs theme: https://squidfunk.github.io/mkdocs-material/reference/admonitions/#supported-types

**General advice**

Admonitions are to highlight something special, not to make every point significant. When used in this way, they are ignored in the same way as 'the boy who cried wolf'. But they help to break up large blocks of text, and add a little colour.

- Use sparingly.

- Consider whether the same text can be emphasised with normal means (italics or bold).

## Overview

The table below summarizes the use of colors in admonitions.

| Admonition                  | MkDocs colour |
| --------------------------- | ------------- |
| Notes, info                 | Blue          |
| See also                    | Blue          |
| Tip                         | Green         |
| Caution, Warning, Important | Amber         |
| Danger                      | Red           |
| Summary                     | Turquoise     |

### Note, Info

Use as a side panel, an 'aside', a note detached from the main flow of the text.

Preferred use is without the label (first form).

```txt
!!! note alert alert-primary ""
    Text ...

!!! note alert alert-primary "Note"
    Text ...

!!! note alert alert-primary "Side topic"
    Text ...
```

### Caution, Warning, Important

Uses same type but different label text:

- Caution: Used to mean 'Continue with care'. It is less strong than 'Warning'.

- Important: A significant point that deserves emphasis. (MkDocs default for 'important' admonition is green, which is why we don't use it.)

Style:

- MkDocs: Amber with triangle/! icon

```txt
!!! caution alert alert-warning "Caution"
!!! caution alert alert-warning "Important"
```

### Danger

Anything that has the potential to damage or compromise a user's data or system.

- MkDocs: Red with bolt icon.

```txt
!!! danger alert alert-danger "Danger"
```

### Tip

Use for tips, hints, non-essential but useful advice. Note that `tip` renders badly in Percona.com. `hint` is better and looks the same as `tip` in Material theme.

```txt
!!! hint alert alert-success "Tip"
    Tip

!!! hint alert alert-success "Tips"
    - One
    - Two

!!! hint alert alert-success ""
    Tip
```

### Summary

Used to summarise a block of text (a TLDR).

```txt
!!! summary alert alert-info "Summary"
```

### See Also

Used to highlight other sections or external links.

Group them at the end of the section.

An exception would be when there is an equivalent or closely related section elsewhere.

- MkDocs: Blue with pen icon (same as note).

```txt
!!! seealso alert alert-info "See also"
```

## Variables

We use the `mkdocs-macros` plugin for variable expansion. For example, the variable `release` in `variables.yml` is used in the code so that the current PMM release number is always up-to-date. (Search the markdown files for `{{release}}`.)

This plugin can have problems when Jinja-like constructs are used in code. This happens when referring to Docker variables. Workarounds are explained here: https://github.com/fralau/mkdocs_macros_plugin/blob/master/webdoc/docs/advanced.md#solutions

In some places, we have used variables themselves to solve the problem. In others, `{% raw %}/{% endraw %}` surrounds the conflicting text.

## Icons

We use a single set of icons: the ones bundled with the Mkdocs Material theme. It offers 10,000+ icons, which is more than sufficient for all our documentation needs, and it covers four families:

| Family | Shortcode prefix | Example |
| ------ | ---------------- | ------- |
| Material Design Icons | `:material-` | `:material-cog:` |
| Font Awesome | `:fontawesome-brands-`, `:fontawesome-solid-`, `:fontawesome-regular-` | `:fontawesome-brands-github:` |
| Octicons | `:octicons-` | `:octicons-alert-16:` |
| Simple Icons | `:simple-` | `:simple-docker:` |

Prefer Material Design Icons unless another family has a markedly better match — brand logos, for instance, usually come from Font Awesome or Simple Icons.

To add an icon, go to <https://squidfunk.github.io/mkdocs-material/reference/icons-emojis/>, search for one, select it, and copy the shortcode here. Every family renders as inline SVG that inherits the surrounding text colour, so icons follow the light/dark colour scheme automatically and need no extra CSS.

Two icon sets used to be loaded from external stylesheets and have both been removed. Don't reintroduce either:

- **Iconscout Unicons** (`uil-` prefix), the set the PMM UI (Grafana) itself uses. They were plain font glyphs that rendered poorly in both the light and the dark theme.
- **Font Awesome 4.4.0** (`fa-` prefix), loaded as raw HTML. It had no remaining usages, and the version was long past end of life. Use the `:fontawesome-*:` shortcodes above instead — the theme bundles the icons, so there is no external version to keep in sync.

Note: the following list is WIP and will be updated as we go along.

| Icon code                             | Description                        | Used where                           |
| ------------------------------------- | ---------------------------------- | ------------------------------------ |
| :material-alert-outline:              | Exclamation mark in triangle       | PMM UI - Warnings                    |
| :material-arrow-left:                 | Left arrow                         | PMM UI                               |
| :material-bell-outline:               | Bell                               | PMM UI - Alerting                    |
| :material-chart-bar:                  | 3-bar chart                        | PMM UI link to dashboard             |
| :material-chevron-down:               | Down chevron                       | PMM UI                               |
| :material-clipboard-list-outline:     | Clipboard list                     | PMM UI - Inventory                   |
| :material-clock-time-nine-outline:    | Clock (at nine)                    | PMM UI - Time range selector         |
| :material-close:                      | Large 'X'                          | PMM UI                               |
| :material-cog:                        | Cog wheel                          | PMM UI Configuration                 |
| :material-cog-outline:                | Cog wheel                          | PMM UI Configuration->Settings       |
| :material-compass-outline:            | Compass                            | PMM UI - Explore                     |
| :material-content-copy:               | Copy                               | PMM UI - Copy (e.g. backup schedule) |
| :material-cube-outline:               | Cube                               | PMM UI                               |
| :material-dots-circle:                | A circle surrounded by smaller ones| PMM UI - Node dashboards             |
| :material-dots-horizontal:            | Triple dots, aligned horizontally  | PMM UI - Backup in progress          |
| :material-dots-vertical:              | Vertical ellipsis                  | PMM UI column menus                  |
| :material-eye:                        | Eye                                | PMM UI Password reveal               |
| :material-eye-off:                    | Eye with slash                     | PMM UI Password hide                 |
| :material-file-document-outline:      | File symbol                        | PMM UI - Home dashboard              |
| :material-format-list-bulleted:       | List                               | PMM UI - Alert rules                 |
| :material-help-circle-outline:        | Question mark in circle            | PMM UI - Help                        |
| :material-history:                    | Backward arrow circle around clock | PMM UI - Backups and checks          |
| :material-lightning-bolt:             | Lightening flash/bolt              | PMM UI - Nodes compare               |
| :material-magnify:                    | Magnifying glass                   | PMM UI - Search                      |
| :material-magnify-expand:             | Advisors                           | PMM UI - Advisors                    |
| :material-magnify-minus-outline:      | Minus in magnifying glass          | PMM UI - Time range zoom out         |
| :material-menu:                       | 3 horizontal lines                 | PMM UI - HA dashboards               |
| :material-menu-right:                 | Right caret                        | General                              |
| :material-message-arrow-right-outline:| Share comment symbol               | PMM UI - Share dashboard image       |
| :material-monitor:                    | Computer monitor                   | PMM UI - Cycle view mode             |
| :material-pencil-outline:             | Pen                                | PMM UI - Edit                        |
| :material-plus-box-outline:           | Plus within square                 | PMM UI - Add                         |
| :material-plus-circle-outline:        | Plus within circle                 | PMM UI Inventory->Add Instance       |
| :material-share-variant:              | Share symbol                       | PMM UI - Share dashboard             |
| :material-shield-outline:             | Shield                             | PMM UI - Server admin                |
| :material-star-outline:               | Star                               | PMM UI - Dashboard favourites        |
| :material-sync:                       | Twin backward arrows               | PMM UI - Refresh dashboard           |
| :material-thumb-down-outline:         | Hand, thumbs down                  | For Benefits/Drawbacks tables        |
| :material-thumb-up-outline:           | Hand, thumbs up                    | For Benefits/Drawbacks tables        |
| :material-toggle-switch-off-outline:  | Toggle (off)                       | PMM UI - Toggle switch               |
| :material-toggle-switch-outline:      | Toggle (on)                        | PMM UI - Toggle switch               |
| :material-trash-can-outline:          | Trash can                          | PMM UI - Various 'Delete' operation  |
| :material-view-dashboard:             | Abstract blocks assembly           | PMM UI - Dashboards                  |

## Symbols

While MkDocs will automatically replace certain strings with symbols, it's preferable where possible to use unicode symbols for other icons, so that they appear when the raw Markdown is exported as HTML and imported into Google Docs.

| For | Use |
| --- | --- |
| --> | →   |

## Language

We have attempted to eschew traditional terminology used in software manuals. Some examples:

- "Setting up" instead of "installation and configuration"
- "Before you start" instead of "Prerequisites"

There are no "introduction" or "overview" sections. These texts are just there under the title.

Section titles are deliberately short. For example, in Setting up/Server/Docker, the 'Run' section shows how to run the docker image for PMM Server. The docs are for 'PMM', the section is 'Server' and subsection 'Docker'. That's what we're running.

## Numbered lists

Most Markdown processors automatically number lists when they are like this:

```md
1. Item
1. Item
1. Item
   ...
```

But to make the raw Markdown easier to read, we recommend explicitly numbering items:

```md
1. Item
2. Item
3. Item
   ...
```

Other advantages:

- contents can be reused in source code comments by developers;
- encourages authors to pay attention to the order and number of steps in a recipe.
