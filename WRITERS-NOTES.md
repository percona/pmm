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

Use for tips, hints, non-essential but useful advice.

```
!!! tip alert alert-success "Tip"
    Tip

!!! tip alert alert-success "Tips"
    - One
    - Two

!!! tip alert alert-success ""
    Tip
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