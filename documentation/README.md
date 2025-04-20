# Percona Monitoring and Management (PMM) documentation
[![render](https://img.shields.io/badge/pmm--doc-render-Green)](https://pmm-doc.onrender.com/)
[![Build](https://github.com/percona/pmm/actions/workflows/documentation.yml/badge.svg?branch=v3)](https://github.com/percona/pmm/actions/workflows/documentation.yml)
[![Helm](https://github.com/percona/pmm/actions/workflows/helm-tests.yml/badge.svg?branch=v3)](https://github.com/percona/pmm/actions/workflows/helm-tests.yml)
[![Podman](https://github.com/percona/pmm/actions/workflows/podman-tests.yml/badge.svg?branch=v3)](https://github.com/percona/pmm/actions/workflows/podman-tests.yml)

[Percona Monitoring and Management] (PMM) is a database monitoring solution that is free and open-source.

This repo holds the source files for the official [PMM technical documentation].

## Contributing to the docs

You can contribute to the documentation in two ways:

- **report an issue**: [open a Jira] issue.

- **fix a problem yourself**: Click <i class="uil uil-edit"></i> **Edit this page** icon at the top of the topic you want to change to access the Markdown source. Fork the repo, make changes, and submit a PR. For large changes, build the website locally to see how it looks in context. 

## Building the documentation

We use [MkDocs] to convert [Markdown] files into a static HTML website (and optionally a [PDF](#pdf)).

The docs live in the `docs/` directory. Other files in this repo are explained in [Directories and files](#directories-and-files).

PMM versions are managed in branches:

- `v3` is for PMM 3.x (latest)

- `main` is for PMM 2.x 

- `1.x` is for PMM 1.x

### Before you begin

Before editing a page, make sure you have basic understanding of [Git], [Python], Docker, and [Markdown]—including how to install and use them via the command line.
If you're not comfortable with these tools, no worries, just [open a Jira issue] instead of editing the documentation directly.

### Building locally

If you’d like to preview PMM docs locally—or plan to contribute, it helps to build the documentation to see how it will look when published. The easiest way is to use Docker, as this avoids having to install MkDocs and its dependencies.

### With Docker

1. Install [Docker].

2. Clone this repository.

3. Change directory to `pmm`.

4. Use our [PMM documentation Docker image] to build the documentation:

    ```sh
    docker run --rm -v $(pwd):/docs perconalab/pmm-doc-md mkdocs build -f documentation/mkdocs.yml
    ```

5. Find the `site` directory, open `index.html` in a browser to view the first page of documentation.

### Live preview 

If you want to see how things look as you edit, MkDocs has a built-in server for live previewing: 

1. After (or instead of) building, run:

    ```sh
    docker run --rm -v $(pwd):/docs -p 8000:8000 perconalab/pmm-doc-md mkdocs serve --dev-addr=0.0.0.0:8000  -f documentation/mkdocs.yml
    ```

2. Wait until you see `INFO    -  Start detecting changes` then browse to `http://0.0.0.0:8000`.

### Without Docker

If you don't use Docker, you must install MkDocs and all its dependencies.

1. Install [Python].

2. Clone the repo and navigate to `pmm/documentation`.

4. Install MkDocs and required extensions:

    ```sh
    pip install -r requirements.txt
    ```

5. Build the docs:

    ```sh
    mkdocs build
    ```

6. Open `site/index.html` or run the built-in web server:

    ```sh
    mkdocs serve
    ```

7. View the site at `http://0.0.0.0:8000`

## Generating a PDF

To create a PDF version of the documentation:

1. (For Percona staff) If building for a release of PMM, edit `mkdocs-base.yml` and change:

    - The release number in `plugins.with-pdf.output_path`
    - The release number and date in `plugins.with-pdf.cover_subtitle`

2. Build

    - with Docker:

        ```sh
        docker run --rm -v $(pwd):/docs -e ENABLE_PDF_EXPORT=1 perconalab/pmm-doc-md mkdocs build -f documentation/mkdocs-pdf.yml
        ```

    - without Docker:

        ```sh
        ENABLE_PDF_EXPORT=1 mkdocs build -f mkdocs-pdf.yml
        ```

3. Find the PDF in `site/pdf`.

## Repo structure overview

- `mkdocs-base.yml`: Default MkDocs configuration file. Creates (Material) themed HTML for hosting anywhere

- `mkdocs.yml`: MkDocs configuration file. Adds a google tag for hosting on render.com

- `mkdocs-pdf.yml`: MkDocs configuration file. Creates themed [PDF](#pdf)

- `docs`:

    - `*.md`: Markdown files

    - `images/*`: Images, image resources, videos

    - `css`: Styling

    - `js`: JavaScript files

- `resources`:

    - `bin`

        - `glossary.tsv`: Export from a spreadsheet of glossary entries

        - `make_glossary.pl`: Script to write Markdown page from `glossary.tsv`

        - `grafana-dashboards-descriptions.py`: Script to extract dashboard descriptions from <https://github.com/percona/grafana-dashboards/>

    - `templates`: Stylesheet for PDF output (used by [mkdocs-with-pdf](https://github.com/orzih/mkdocs-with-pdf) extension)

- `requirements.txt`: Python package dependencies

- `variables.yml`: Values used throughout the Markdown, including the current PMM version/release number

- `../.github`:

    - `workflows`:

        - `documentation.yml`: Workflow specification for building the documentation via a GitHub action. Uses `mike` which puts HTML in `publish` branch.

- `site`: When building locally, directory where HTML is put

## Version switching

We use [mike] to build different versions of the documentation. Currently, only two are built, the latest PMM 2 and PMM 3 versions.

A [GitHub actions] workflow runs `mike` which in turn runs `mkdocs`. The HTML is committed and pushed to the `publish` branch. The whole branch is then copied (by an internal Percona Jenkins job) to our web server.

## Image overlays

The file`docs/using/interface.md` includes a screenshot of the PMM Home dashboard overlaid with numbered boxes to identify menu bars and control. This approach means the Home dashboard image and its numbered version always look the same:

- `PMM_Home_Dashboard.jpg` is snapped manually and it should be 1280x1280 pixels, to match the overlay image.

- `PMM_Home_Dashboard_Overlay.png` is exported from `documentation/docs/images/PMM_Home_Dashboard_Overlay.drawio` using <https://app.diagrams.net/>.

To update the visual:

1. Access <https://app.diagrams.net/>

2. On first use, choose **Device** for saving diagrams.

3. Click **Open existing diagram**.

4. Navigate to `documentation/docs/images` and select `PMM_Home_Dashboard_Overlay.drawio`.

5. If the dashboard layout has changed, replace the **Guide** layer with a new screenshot and adjust the elements on the **Overlay** layer as needed. 

6. Click **View > Layers** to toggle layers and disable the **Guide** layer before exporting.

7. Click **File > Export as > PNG**.

8. In the *Image settings* dialog, use these settings:

    - **Zoom**: 100%
    - **Border width**: 0
    - **Size**: Page (The page dimensions in inches should be as close to the base image as possible, i.e. 1280x1280)
    - **Transparent Background**: ON
    - **Shadow**: OFF
    - **Grid**: OFF
    - **Include a copy of my diagram**: OFF

9. Click **Export**.
10. Choose *Device* and save as `PMM_Home_Dashboard_Overlay.png`. 
11. Click **Save** and overwrite the current file

### Merging overlays
Use [ImageMagick]'s [composite] tool to merge the overlay with the base image:

```sh
composite documentation/docs/images/PMM_Home_Dashboard_Overlay.png documentation/docs/images/PMM_Home_Dashboard.jpg documentation/docs/images/PMM_Home_Dashboard_Numbered.png
```

This creates a new file `PMM_Home_Dashboard_Numbered.png`ready to be used in the documentation.

## Spelling and grammar checks

By default, the GitHub Actions build job runs a basic spell check. A grammar check is available but currently commented out in the workflow file. You can run both checks locally from the command line if you have [Node.js] installed.

### Spell check

1. Install the markdown-spellcheck tool globally:

    ```sh
    npm i markdown-spellcheck -g
    ```
2. To check a specific file:

    ```sh
    mdspell --report --en-us --ignore-acronyms --ignore-numbers docs/<path to file>.md
    ```

3. To check all Markdown files:

    ```sh
    mdspell --report --en-us --ignore-acronyms --ignore-numbers "docs/**/*.md"
    ```

4. Add any project-specific or technical terms to the `.spelling` file to avoid false positives.


The GitHub job prints spell check results but does not fail the build based on spelling errors.

### Grammar checks

Grammar is checked using [`write-good`](https://github.com/btford/write-good).

1. Install `write-good` globally: 

    ```sh
    npm i write-good -g
    ```

2. To check a specific file:
    ```sh
    write-good docs/<path to file>.md
    ```
3. To check all Markdown files:

    ```sh
    write-good docs/**/*.md
    ```

## Link checking

Broken link detection is handled via the `mkdocs-htmlproofer-plugin`. This plugin is effective but can significantly slow down build times (by 10x to 50x).

The plugin is already included in:
- [PMM documentation Docker image]
- GitHub Action workflow (although it's commented out in `mkdocs.yml`)

To enable it for local builds:

1. Open mkdocs.yml.
2. Uncomment the line with `htmlproofer` in the `plugins` section of `mkdocs.yml` and parse the build output for warnings.
3. Run a local build and check the terminal output for broken link warnings.

[Percona Monitoring and Management]: https://www.percona.com/software/database-tools/percona-monitoring-and-management
[PMM technical documentation]: https://docs.percona.com/percona-monitoring-and-management/
[open a Jira]: https://perconadev.atlassian.net/browse/PMM
[MkDocs]: https://www.mkdocs.org/
[Markdown]: https://daringfireball.net/projects/markdown/
[Git]: https://git-scm.com
[Python]: https://www.python.org/downloads/
[Docker]: https://docs.docker.com/get-docker/
[PMM documentation Docker image]: https://hub.docker.com/repository/docker/perconalab/pmm-doc-md
[mike]: https://github.com/jimporter/mike
[GitHub actions]: https://github.com/percona/pmm/actions
[ImageMagick]: https://imagemagick.org/script/download.php
[composite]: https://imagemagick.org/script/composite.php
[Node.js]: https://nodejs.org/en/download/
