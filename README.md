# Percona Monitoring and Management (PMM) Documentation
Here are documentation source files for [Percona Monitoring and Management](https://www.percona.com/software/database-tools/percona-monitoring-and-management), a free, open-source, database monitoring solution.

> **Note**
>
> This repository is for Percona Monitoring and Management version 2.

The HTML documentation is published at [percona.com/doc](https://www.percona.com/doc/percona-monitoring-and-management/2.x/index.html).

We welcome any contributions. This page explains how you can do that.

## Overview

There are currently two identical copies of documentation in different formats:

- `.rst` files in [reStructuredText](https://docutils.sourceforge.io/docs/user/rst/quickstart.html) syntax for processing by [Sphinx](https://www.sphinx-doc.org/).

- `.md` files in [Markdown](https://daringfireball.net/projects/markdown/) syntax for processing by [MkDocs](https://www.mkdocs.org/).

You can edit whichever copy you like. The PMM Technical Writers will keep the two copies synchronized until a decision is made about which format to keep.

- Sphinx/rst documentation is stored in the `source` directory.
- MkDocs/md documentation is stored in the `docs` directory.

## How to Contribute

You'll need to know how git works, and the syntax of either reStructuredText or Markdown.

For option 2 (below), you'll need to [install Sphinx and extensions](#install-sphinx-and-extensions), or [MkDocs and extensions](#install-mkdocs-and-extensions), or have [Docker](https://docs.docker.com/get-docker/) installed to preview any changes. (Of these, Docker is by far the simplest.)

There are three ways to get changes made to the documentation. Two are 'do it yourself', one is 'ask someone to do it'.

### Option 1: 'Do it yourself': Edit via Github

1. Each page of [PMM 2 documentation](https://www.percona.com/doc/percona-monitoring-and-management/2.x/index.html) has a link to the `.rst`. and `.md` versions of the page.

2. Click any link to be taken to the github edit page.

3. Make your changes and commit. Unless you are a member of the Percona team, you'll be asked to fork the repository.

4. Do so and make a pull request for merging your changes.

### Option 2: 'Do it yourself': Edit a cloned copy

1. Fork and clone this repository.

2. Make your changes in either the Sphinx/rst version (under `pmm-doc/source`) or MkDocs/md (under `pmm-doc/docs`).

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

### Preview Sphinx/reST documentation with Docker

To build the documentation (convert `.rst` files into HTML), you must [install Sphinx and extensions](#install-sphinx-and-extensions).

If you have [Docker installed](https://docs.docker.com/get-docker/), a more convenient way is to use our Docker image as follows:

1. Clone this repository.

2. `cd pmm-doc`

3. `docker run --rm -v $(pwd):/docs perconalab/percona-doc-sphinx make clean html`

4. Open `build/html/index.html` in a browser to view the first page of documentation.

> **Tip**
>
> Documentation built this way has no styling because it is intended for hosting on percona.com.
> You can build a themed version for local viewing by changing the command in step 3 to:
>
> `docker run --rm -v $(pwd):/docs perconalab/percona-doc-sphinx make clean thtml`

### Preview MkDocs/md documentation with Docker

To build the documentation (convert `.md` files into HTML), you must [install MkDocs and extensions](#install-mkdocs-and-extensions).

If you have [Docker installed](https://docs.docker.com/get-docker/), a more convenient way is to use our Docker image as follows:

1. Clone this repository.

2. `cd pmm-doc`

3. `docker run --rm -v $(pwd):/docs perconalab/pmm-doc-md`

4. Open `site/index.html` in a browser to view the first page of documentation.

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

## Install Sphinx and extensions

1. Follow the [official Sphinx instructions for installation](https://www.sphinx-doc.org/en/master/usage/installation.html).

2. Install required extensions:

    `pip install sphinxcontrib-srclinks`

## Install MkDocs and extensions

1. Follow the [official MkDocs instructions for installation](https://www.mkdocs.org/#installing-mkdocs).

2. Install required extensions:

    `pip install mkdocs-macros-plugin mkdocs-exclude mkdocs-material`
