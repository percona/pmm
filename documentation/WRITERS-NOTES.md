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

Currently, we use three different sets of icons:

- Native Mkdocs Material icons and emojis
- Iconscount Unicons
- FontAwesome Icons

The preference should be given to Mkdocs Material icons, since they were initially designed for `mkdocs` project and work very well with it. In rare cases, when an icon cannot be found in Mkdocs Material, you can find a suitable one on Iconscout. FontAwesome icons do not play well with Mkdocs, so use them with caution and don't forget to apply additional CSS styles in case they don't render well in PDF.

Ideally, we should migrate to using just the two first sets.

Use HTML for icons:

- For Material Theme Icons (`:` prefix and suffix), go to <https://squidfunk.github.io/mkdocs-material/reference/icons-emojis/>, find an icon, select it and copy the code here.
- For PMM UI (Grafana) icons (`uil-` prefix), go to <https://iconscout.com/unicons/explore/line>, find an icon, select 'Font' and copy the code here.
- For Font Awesome (`fa-` prefix), go to <https://fontawesome.com/icons>, find an icon, copy the code.

| Unicons icon code                            | Description                        | Used where                           |
| -------------------------------------------- | ---------------------------------- | ------------------------------------ |
| <i class="uil uil-angle-down"></i>           | Down chevron                       | PMM UI                               |
| <i class="uil uil-apps"></i>                 | Four boxes in square               | PMM UI - QAN                         |
| <i class="uil uil-arrow-left"></i>           | Left arrow                         | PMM UI                               |
| <i class="uil uil-bars"></i>                 | 3 horizontal lines                 | PMM UI - HA dashboards               |
| <i class="uil uil-bell"></i>                 | Bell                               | PMM UI - Alerting                    |
| <i class="uil uil-bolt"></i>                 | Lightening flash/bolt              | PMM UI - Nodes compare               |
| <i class="uil uil-caret-right"></i>          | Right caret                        | General                              |
| <i class="uil uil-clock-nine"></i>           | Clock (at nine)                    | PMM UI - Time range selector         |
| <i class="uil uil-cog"></i>                  | Cog wheel                          | PMM UI Configuration                 |
| <i class="uil uil-comment-alt-share"></i>    | Share comment symbol               | PMM UI - Share dashboard image       |
| <i class="uil uil-compass"></i>              | Compass                            | PMM UI - Explore                     |
| <i class="uil uil-copy"></i>                 | Copy                               | PMM UI - Copy (e.g. backup schedule) |
| <i class="uil uil-cube"></i>                 | Cube                               | PMM UI                               |
| <i class="uil uil-ellipsis-v"></i>           | Vertical ellipsis                  | PMM UI column menus                  |
| <i class="uil uil-exclamation-triangle"></i> | Exclamation mark in triangle       | PMM UI - Warnings                    |
| <i class="uil uil-eye-slash"></i>            | Eye with slash                     | PMM UI Password hide                 |
| <i class="uil uil-eye"></i>                  | Eye                                | PMM UI Password reveal               |
| <i class="uil uil-file-alt"></i>             | File symbol                        | PMM UI - Home dashboard              |
| <i class="uil uil-graph-bar"></i>            | 3-bar chart                        | PMM UI link to dashboard             |
| <i class="uil uil-history"></i>              | Backward arrow circle around clock | PMM UI - Backups and checks          |
| <i class="uil uil-list-ul"></i>              | List                               | PMM UI - Alert rules                 |
| <i class="uil uil-monitor"></i>              | Computer monitor                   | PMM UI - Cycle view mode             |
| <i class="uil uil-pen"></i>                  | Pen                                | PMM UI - Edit                        |
| <i class="uil uil-plus-circle"></i>          | Plus within circle                 | PMM UI Inventory->Add Instance       |
| <i class="uil uil-plus-square"></i>          | Plus within square                 | PMM UI - Add                         |
| <i class="uil uil-question-circle"></i>      | Question mark in circle            | PMM UI - Help                        |
| <i class="uil uil-search-minus"></i>         | Minus in magnifying glass          | PMM UI - Time range zoom out         |
| <i class="uil uil-search"></i>               | Magnifying glass                   | PMM UI - Search                      |
| <i class="uil uil-setting"></i>              | Cog wheel                          | PMM UI Configuration->Settings       |
| <i class="uil uil-share-alt"></i>            | Share symbol                       | PMM UI - Share dashboard             |
| <i class="uil uil-shield"></i>               | Shield                             | PMM UI - Server admin                |
| <i class="uil uil-star"></i>                 | Star                               | PMM UI - Dashboard favourites        |
| <i class="uil uil-sync"></i>                 | Twin backward arrows               | PMM UI - Refresh dashboard           |
| <i class="uil uil-thumbs-down"></i>          | Hand, thumbs down                  | For Benefits/Drawbacks tables        |
| <i class="uil uil-thumbs-up"></i>            | Hand, thumbs up                    | For Benefits/Drawbacks tables        |
| <i class="uil uil-times"></i>                | Large 'X'                          | PMM UI                               |
| <i class="uil uil-toggle-off"></i>           | Toggle (off)                       | PMM UI - Toggle switch               |
| <i class="uil uil-toggle-on"></i>            | Toggle (on)                        | PMM UI - Toggle switch               |
| <i class="uil uil-trash-alt"></i>            | Trash can                          | PMM UI - Various 'Delete' operation  |

Custom (in-house design) icons are defined as SVG code in `variables.yml`.

| Usage                  | Description                          | Used where                       |
| ---------------------- | ------------------------------------ | -------------------------------- |
| `{{icon.checks}}`      | Checkbox list items                  | Inventory list                   |
| `{{icon.inventory}}`   | Checkbox list items                  | PMM Inventory & Inventory List   |
| `{{icon.dashboards}}`  | Abstract blocks assembly             | PMM Dashboards                   |
| `{{icon.node}}`        | Circle surrounded by smaller circles | Node dashboards                  |
| `{{icon.percona}}`     | Percona logo                         | Wherever PMM home icon is needed |

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
