# Percona Monitoring and Management (PMM) Documentation
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fpercona%2Fpmm-doc.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fpercona%2Fpmm-doc?ref=badge_shield)

[Percona Monitoring and Management (PMM)](https://www.percona.com/software/database-tools/percona-monitoring-and-management) is a free, open-source, database monitoring solution.

PMM technical documentation is at <https://www.percona.com/doc/percona-monitoring-and-management/>.

This repo holds the source files for it.

If you're a user of PMM's technical documentation and would like to contribute, here's what you can do.

- **Report a problem**: Each page has a link, *Report a problem with this page*, a shortcut to  to this repo's *Issues*. Click it and describe your issue and we'll try to fix it as quick as we can. If it's a general problem, open an Issue here or in [Percona's Jira](https://jira.percona.com/browse/PMM).

- **Fix a problem**: There is also an *Edit this page* link that will take you to the Markdown source file for that page. Make your changes (forking if necessary) and submit a PR which we'll review and adjust where necessary before merging. If the changes are more than a few lines, you might want to build the website locally to see how it looks in context. To do that, read on.

## Introduction

The documentation is in the `docs` directory. It's written in [Markdown](https://daringfireball.net/projects/markdown/) ready for [MkDocs](https://www.mkdocs.org/) to convert it into a static HTML website. We call that process [*building the documentation*](#building-the-documentation).

> **Branches**
> PMM2 is in the `master` branch.
> PMM1 is in `1.x`.

To know about other files in this repo, jump to [Directories and files](#directories-and-files).


## Before you start

You'll need to know:

- what git, [Python 3](https://www.python.org/downloads/) and [Docker](https://docs.docker.com/get-docker/) are;
- what Markdown is and how to edit it;
- and, how to install and run those things on the command line.

(If you don't, open an Issue instead.)

## Building the documentation

1. Clone this repository.

2. `cd pmm-doc`

3. Decide whether you want to install MkDocs and dependencies on your machine, or run MkDocs via our Docker image. (Docker is easier so we'll show it first.)
### With Docker

1. Run the image to *build the documentation* (create a static web site in the `site` subdirectory):

		docker run --rm -v $(pwd):/docs perconalab/pmm-doc-md

2. Find the `site` directory, open `index.html` in a browser to view the first page of documentation.

Documentation built this way has no styling because it is built with a custom theme for our CMS (there is no outer `<html>` tag, anything in `<head>` is ignored, and there's some custom stuff for navigation and [version switching](#versioning)).

A themed version looks much better and is just as easy. Run this instead:

	docker run --rm -v $(pwd):/docs perconalab/pmm-doc-md mkdocs build -t material

If you'd like to see how things look as you edit, MkDocs has a built-in server for live previewing. After (or instead of) building, run:

	docker run --rm -v $(pwd):/docs -p 8000:8000 perconalab/pmm-doc-md mkdocs serve -t material --dev-addr=0.0.0.0:8000


and point your browser to [http://localhost:8000](http://localhost:8000).

### Without Docker

1. Install [Python 3](https://www.python.org/downloads/)

2. Install MkDocs and required extensions:

        pip install -r requirements.txt

3. Start the site:

        mkdocs serve -t material

4. View the site: visit <http://localhost:8000>

## Versioning

We are trialing the use of [mike](https://github.com/jimporter/mike) to build different versions.

With this, MkDocs is run locally and the HTML committed (and optionally pushed) to the `publish` branch. The whole branch then copied (by us, naturally) to our web server.

(This is why the PMM1 docs (previously in <https://github.com/percona/pmm>) have been migrated from Sphinx/rst to Markdown/md and moved to the `1.x` branch of this repository.)

## Directories and files

Here's what you'll find in the `master` branch (PMM2). (`1.x` for PMM1 has only `docs`)

- `bin`:
    - `glossary.tsv`: Export from a spreadsheet of glossary entries
    - `make_glossary.pl`: Script to write Markdown page from `glossary.tsv`
    - `grafana-dashboards-descriptions.py`: Script to extract dashboard descriptions from <https://github.com/percona/grafana-dashboards/>

- `docs`: Base directory for MkDocs

- `resources`:
    - `*.puml`: PlantUML diagrams
    - `*.odg`: Original architecture diagrams in LibreOffice Draw format (for historical interest only)

- `templates`: Stylesheet for PDF output (used by [`mkdocs-with-pdf`](https://github.com/orzih/mkdocs-with-pdf))

- `theme`: MkDocs templates that produce HTML output for percona.com hosting

- `mkdocs.yml`: For building HTML.
- `mkdocs-pdf.yml`: For building the PDF.

> The HTML version includes an SVG site map that's not in the PDF. This is done by having two index pages (`index.md` for HTML, `index-pdf.md` for PDF) both including `welcome.md`, the core of the home page.

- `release.yml`: The latest PMM release and version numbers.
- `extra.yml`: Miscellaneous values and website links.
- `mkdocs.yml`: The `extra` element has text for page links.
- `icon.yml`: A convenience list of icon variables. Use them in Markdown with `{{ icon.NAME }}`.
- `requirements.txt`: Python package dependencies for MkDocs


## License
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fpercona%2Fpmm-doc.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fpercona%2Fpmm-doc?ref=badge_large)
