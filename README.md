# Percona Monitoring and Management (PMM) Documentation
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fpercona%2Fpmm-doc.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fpercona%2Fpmm-doc?ref=badge_shield)

Here are documentation source files for [Percona Monitoring and Management](https://www.percona.com/software/database-tools/percona-monitoring-and-management/2.x/), a free, open-source, database monitoring solution.

> **Note**
>
> This repository is for Percona Monitoring and Management version 2.

We welcome any contributions. This page explains how you can do that, and how to build a local copy of the documentation.

The documentation consists of [Markdown](https://daringfireball.net/projects/markdown/) files in the `docs` directory. We use [MkDocs](https://www.mkdocs.org/) to convert these into a static HTML website.

## Build the documentation

### First

1. Clone this repository.

2. `cd pmm-doc`

### With Docker

1. Install [Docker](https://docs.docker.com/get-docker/).

2. `docker run --rm -v $(pwd):/docs perconalab/pmm-doc-md`

3. Open `site/index.html` in a browser to view the first page of documentation.

> **Tip**
>
> Documentation built this way has no styling because it is intended for hosting on percona.com.
> You can build a themed version for local viewing by changing the command in step 3 to:
>
> `docker run --rm -v $(pwd):/docs perconalab/pmm-doc-md mkdocs build -t material`
>
> Alternatively, you can use the MkDocs built-in web server to live preview local edits:
>
> `docker run --rm -v $(pwd):/docs -p 8000:8000 perconalab/pmm-doc-md mkdocs serve -t material --dev-addr=0.0.0.0:8000`
>
> and point your browser to [http://localhost:8000](http://localhost:8000).

### Without Docker

1. Install [Python 3](https://www.python.org/downloads/)

2. Install MkDocs and required extensions:

        pip install -r requirements.txt

3. Start the site:

        mkdocs serve -t material

4. View the site: visit <http://localhost:8000>

## How to Contribute

You can change documentation yourself, or ask us to do it.

### Option 1: Do it yourself

1. Each page of [PMM 2 documentation](https://www.percona.com/doc/percona-monitoring-and-management/2.x/) has a link to the `.md` version of the page.

2. Click the link to be taken to the github edit page.

3. Make your changes and commit. Unless you are a member of the Percona team, you'll be asked to fork the repository.

4. Do so and make a pull request for merging your changes.

### Option 2: Ask us

1. Create a ticket in our [Jira](https://jira.percona.com/projects/PMM/issues) system.

2. Describe the problem or improvement needed in as much detail as possible, by providing, for example:
   - links to the relevant pages or sections;
   - explaining what is wrong and why;
   - suggesting changes or links to sources of further information.

3. You can use Jira to communicate with developers and technical writers, and be notified of progress.

## Notes

### Structure

The HTML version includes an SVG site map that's not in the PDF. This is done by having two index pages (`index.md` for HTML, `index-pdf.md` for PDF) both including `welcome.md`, the core of the home page.

### Configuration

There are two MkDocs configuration files:

- `mkdocs.yml`: For building HTML.
- `mkdocs-pdf.yml`: For building the PDF.

### Variables

Variables are in:

- `release.yml`: The latest PMM release and version numbers.
- `extra.yml`: Miscellaneous values and website links.
- `mkdocs.yml`: The `extra` element has text for page links.

### Icons

PMM's user interface is based on Grafana which which uses the [Unicons](https://iconscout.com/unicons/explore/line) icons set.

A convenience list of icon variables is in `icon.yml`. Use them in Markdown with `{{ icon.NAME }}`. (See examples in `docs/using/alerting.md`.)

## License
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fpercona%2Fpmm-doc.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fpercona%2Fpmm-doc?ref=badge_large)
