# Writer's Notes

## Admonitions

Admonitions use a combined MkDocs/Bootstrap definition to get acceptable and similar rendering on both Percona.com (Drupal-based) and Netlify.

Percona.com uses Bootstrap 4. Admonitions are styled as [Alerts](https://getbootstrap.com/docs/4.0/components/alerts/).

**General advice**

Admonitions are to highlight something special, not to make every point significant. When used in this way, they are ignored in the same way as 'the boy who cried wolf'. But they help to break up large blocks of text, and add a little colour.

- Use sparingly.

- Consider whether the same text can be emphasised with normal means (italics or bold).

## Overview

By using a subset of all MkDocs, we can get some alignment between those and Bootstrap.

| Admonition                  | MkDocs colour | Bootstrap colour |
|-----------------------------|---------------|------------------|
| Notes, info                 | Blue          | Blue             |
| See also                    | Blue          | Turqoise         |
| Tip                         | Green         | Green            |
| Caution, Warning, Important | Amber         | Amber            |
| Danger                      | Red           | Red              |



### Note, Info

Use as a side panel, an 'aside', a note detached from the main flow of the text.

Preferred use is without the label (first form).

```
!!! note alert alert-primary ""
    Text ...

!!! note alert alert-primary "Note"
    Text ...

!!! note alert alert-primary "Side topic"
    Text ...
```

### Caution, Warning, Important

Uses same type but different label text:

- Caution: Use to mean 'Continue with care'. Less strong than 'Warning' IMHO.

- Important: A significant point that deserves emphasis. (MkDocs default for 'important' admonition is green which is why I don't use it.)

Style:

- MkDocs: Amber with triangle/! icon
- Bootstrap: Yellow with no icon

```
!!! caution alert alert-warning "Caution"
!!! caution alert alert-warning "Important"
```

### Danger

Anything that has the potential to damage or compromise a user's data or system.

- MkDocs: Red with bolt icon.
- Boostrap: Red with no icon.

```
!!! danger alert alert-danger "Danger"
```

### Tip

Use for tips, hints, non-essential but useful advice. Note that `tip` renders badly in Percona.com. `hint` is better and looks the same as `tip` in Material theme.

```
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

```
!!! summary alert alert-info "Summary"
```

### See Also

Used to highlight other sections or external links.

Group them at the end of the section.

An exception would be when there is an equivalent or closely related section elsewhere.

- MkDocs: Blue with pen icon (same as note).
- Bootstrap: Turquoise with no icon.

```
!!! seealso alert alert-info "See also"
```


## Variables

We use the `mkdocs-macros` plugin for variable expansion. For example, the variable `release` in `variables.yml` is used in the code so that the current PMM release number is always up to date. (Search the markdown files for `{{release}}`.)

This plugin can have problems when Jinja-like constructs are used in code. This happens when refering to Docker variables. Work arounds are explained here: https://github.com/fralau/mkdocs_macros_plugin/blob/master/webdoc/docs/advanced.md#solutions

In some places we have used variables themselves to solve the problem. In others, `{% raw %}/{% endraw %}` surrounds the conflicting text.


## Language

We have attempted to eschew traditional terminology used in software manuals. Some examples:

- "Setting up" instead of "installation and configuration"
- "Before you start" instead of "Prerequisites"

There are no "introduction" or "overview" sections. These texts are just there under the title.

Section titles are deliberately short. For example, in Setting up/Server/Docker, the 'Run' section shows how to run the docker image for PMM Server. The docs are for 'PMM', the section is 'Server' and subsection 'Docker'. That's what we're running.