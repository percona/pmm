# Percona Monitoring and Management (PMM) Documentation
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fpercona%2Fpmm-doc.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fpercona%2Fpmm-doc?ref=badge_shield)

Here are documentation source files for [Percona Monitoring and Management](https://www.percona.com/software/database-tools/percona-monitoring-and-management), a free, open-source, database monitoring solution.

> **Note**
>
> This repository is for Percona Monitoring and Management version 2.

The HTML documentation is published at [percona.com/doc](https://www.percona.com/doc/percona-monitoring-and-management/2.x/index.html).

We welcome any contributions. This page explains how you can do that.

## Overview

The documentation is in `docs`. It comprises `.md` files in [Markdown](https://daringfireball.net/projects/markdown/) syntax for processing by [MkDocs](https://www.mkdocs.org/).

## How to Contribute

You'll need to know how git works, and the syntax of Markdown.

You need to install [MkDocs and extensions](#install-mkdocs-and-extensions), or have [Docker](https://docs.docker.com/get-docker/) installed to preview any changes. (Of these, Docker is by far the simplest.)

There are three ways to get changes made to the documentation. Two are 'do it yourself', one is 'ask someone to do it'.

### Option 1: 'Do it yourself': Edit via Github

1. Each page of [PMM 2 documentation](https://www.percona.com/doc/percona-monitoring-and-management/2.x/index.html) has a link to the `.md` version of the page.

2. Click the link to be taken to the github edit page.

3. Make your changes and commit. Unless you are a member of the Percona team, you'll be asked to fork the repository.

4. Do so and make a pull request for merging your changes.

### Option 2: 'Do it yourself': Edit a cloned copy

1. Fork and clone this repository.

2. Make your changes in the `pmm-doc/docs` directory.

3. For all but the simplest changes, you should [preview the documentation](#preview-the-documentation).

4. Commit and push the changes.

5. Make a pull request to merge the changes.

### Option 3: 'Ask someone': Create a ticket

1. Create a ticket in our [Jira](https://jira.percona.com/projects/PMM/issues) system.

2. Describe the problem or improvement needed in as much detail as possible, by providing, for example:
   - links to the relevant pages or sections;
   - explaining what is wrong and why;
   - suggesting changes or links to sources of further information.

3. You can use Jira to communicate with developers and technical writers, and be notified of progress.

## Preview the documentation

### With Docker

1. Install [Docker](https://docs.docker.com/get-docker/).

2. Clone this repository.

3. `cd pmm-doc`

4. `docker run --rm -v $(pwd):/docs perconalab/pmm-doc-md`

5. Open `site/index.html` in a browser to view the first page of documentation.

> **Tip**
>
> Documentation built this way has no styling because it is intended for hosting on percona.com.
> You can build a themed version for local viewing by changing the command in step 3 to:
>
> `docker run --rm -v $(pwd):/docs perconalab/pmm-doc-md mkdocs build -f mkdocs-preview.yml`
>
> Alternatively, you can use the MkDocs built-in web server to live preview local edits:
>
> `docker run --rm -v $(pwd):/docs -p 8000:8000 perconalab/pmm-doc-md mkdocs serve -f mkdocs-preview.yml --dev-addr=0.0.0.0:8000`
>
> and point your browser to [http://localhost:8000](http://localhost:8000).

### Without Docker

To build the documentation without Docker, you must [install MkDocs and extensions](#install-mkdocs-and-extensions).

1. Install MkDocs:

   `pip install mkdocs`

    ([Reference: MkDocs installation](https://www.mkdocs.org/#installing-mkdocs))

2. Install required extensions:

    `pip install mkdocs-macros-plugin mkdocs-exclude mkdocs-material mkdocs-with-pdf markdown-blockdiag`

3. View the site:

   `mkdocs serve -f mkdocs-preview.yml`

   and visit <http://localhost:8000>


## License
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fpercona%2Fpmm-doc.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fpercona%2Fpmm-doc?ref=badge_large)