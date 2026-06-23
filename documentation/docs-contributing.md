# Contributing to PMM documentation

We're glad you're here! Whether you're fixing a typo or adding new content, your contributions make our documentation better for everyone.

By contributing, you agree to the [Percona Community code of conduct](https://percona.community/contribute/coc/).

## Quick feedback

### Rate and comment

Each page has a **Rate this page** feature at the bottom. Rate (1-5 stars) and add specific comments about what needs improvement.

### Join the forum

Discuss documentation on the [Percona Community Forum](https://forums.percona.com/c/percona-product-documentation/71). Ask questions, share feedback, or suggest improvements.

### Report an issue

Create a Jira ticket for formal tracking: [Create PMM documentation issue](https://perconadev.atlassian.net/secure/CreateIssueDetails!init.jspa?pid=11600&issuetype=1)

## Edit the documentation

Ready to make changes? The docs are written in [Markdown](https://www.markdownguide.org/) and live on [Github](https://github.com/percona/pmm/tree/main/documentation/docs).

### Quick edits online

1. Click **Edit this page on GitHub** (pencil icon) at the top of any page
2. Make your changes using Markdown
3. Preview using the **Preview** tab
4. Commit changes and create a pull request

Want more details? Check out [GitHub's guide to editing files](https://docs.github.com/en/repositories/working-with-files/managing-files/editing-files).

### Work locally

1. Clone the repository:

    ```shell
    git clone https://github.com/<your_github_name>/pmm.git
    cd pmm
    ```

2. Add upstream and sync:

    ```shell
    git remote add upstream https://github.com/percona/pmm.git
    git checkout main
    git pull upstream main
    ```

3. Create a branch and make your changes:

    ```shell
    git checkout -b my_changes
    # Edit files in documentation/docs/
    ```

4. Push and create a pull request:

    ```shell
    git add documentation/docs/example.md
    git commit -m 'Document feature X'
    git push -u origin my_changes
    ```

## Build and preview

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) or [Podman](https://podman.io/getting-started/installation)

Before submitting your changes, you can build and preview the documentation locally to see how it will appear:

```shell
# Build the documentation
make doc-build

# Preview the documentation with live reload (recommended)
make doc-build-preview

# Build the PDF (Percona staff only — update the version in mkdocs-base.yml first)
make doc-build-pdf
```

That's it! The `make doc-build-preview` command will start a local server at `http://127.0.0.1:8000/` that automatically reloads when you save changes.

## What happens next

Our team will review your pull request and provide feedback. When everything looks good, we'll merge your changes. We may make minor edits to maintain consistency with our style guide.

Thanks for taking the time to improve our docs!
