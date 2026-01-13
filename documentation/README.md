# Documentation contributing guide

We're glad you're here and want to help improve the [Percona Monitoring and Management documentation](https://docs.percona.com/percona-monitoring-and-management/). Whether you're fixing a typo, clarifying instructions, or adding new content, your contributions make our documentation better for everyone.

By contributing, you agree to the [Percona Community code of conduct](https://percona.community/contribute/coc/).

Here are the ways you can contribute:

## Rate and comment on documentation pages

Found something confusing or want to share feedback? Each page has a **Rate this page** feature at the bottom where you can rate (1-5 stars) and leave comments.

Here's how:

1. Scroll to the bottom of any page.

2. Click the stars to rate (1 = needs work, 5 = excellent).

3. Add your comments in the text box.

!!! important "Help us help you - be specific"

    When you want us to fix or improve something, detailed comments make all the difference. Instead of "this is confusing," tell us:
    
    * What specific issue you ran into or what improvement you'd like to see
    * Which section or paragraph needs work
    * Examples or use cases that would help clarify
    * Your version and environment (if it matters)
    * Steps to reproduce any problems you found
    
    The more details you give us, the better we can address your needs and improve the docs for everyone.

## Join the conversation on our forum

Want to discuss documentation with the community? The [Percona Community Forum](https://forums.percona.com/) is the place to ask questions, share feedback, or suggest improvements. It's a great way to get input from both the community and our documentation team.

To start a discussion, head to the [Percona Product Documentation category](https://forums.percona.com/c/percona-product-documentation/71), click **New Topic**, fill out the form, and hit **Create Topic**.

## Report an issue in Jira

For formal issue tracking, you can create a Jira ticket. This is especially useful when you want to track a specific documentation bug or request over time.

Here's how to create a ticket:

1. Go to the [PMM Jira project](https://perconadev.atlassian.net/browse/PMM).

2. Sign in (or create a free Percona Jira account if you don't have one).

3. Click **Create**. 

4. Fill in the details:

    * **Summary**: A short description of the issue
    * **Description**: The full story - what's wrong, what needs to change, steps to reproduce, your environment details (version, OS, etc.)
    * Add any other relevant fields like **Version** or **Environment**

5. Click **Create** to submit.

!!! tip "Quick link"

    Skip straight to the issue form: [Create PMM documentation issue](https://perconadev.atlassian.net/secure/CreateIssueDetails!init.jspa?pid=11600&issuetype=1)


## Edit the documentation yourself

Ready to make changes directly? You can edit the docs online through GitHub or work on them locally on your machine. Choose whichever approach fits your workflow.

### What you should know

The documentation is written in [Markdown](https://www.markdownguide.org/), a simple plain text format. You can add notes, tables, code blocks, and other formatting using Markdown syntax.

### What happens next

Once you submit your pull request, our team will review it and provide feedback. When everything looks good, we'll merge your changes. Thanks for taking the time to improve our docs!

!!! note

    We may make minor edits to your contribution to maintain consistency with our style guide.

### Edit online with GitHub

This is the quickest way to fix typos or make small changes:

1. Click the **Edit this page on GitHub** button (the pencil icon) at the top of any page. If you haven't worked with our repository before, GitHub will automatically create a fork for you.

2. Make your changes using [Markdown](https://www.markdownguide.org/) syntax.

3. Preview your changes by clicking the **Preview** tab.

4. Scroll down to **Commit changes**.

5. Write a short commit message (72 characters or less) describing what you changed.
 
6. Select **Create a new branch for this commit and start a pull request**. GitHub will suggest a branch name - you can use it or change it.

7. Click **Commit changes**.

8. GitHub will show you a pull request page with:
   * The branch where your changes will go
   * Your commit message
   * A visual diff showing what you changed

9. Review everything and click **Create pull request**.

Want more details? Check out [GitHub's guide to editing files](https://docs.github.com/en/repositories/working-with-files/managing-files/editing-files).

### Edit locally

If you're comfortable with git and prefer working on your own machine, here's the workflow:

1. Fork the repository on GitHub.

2. Clone your fork:

    ```shell
    git clone https://github.com/<your_github_name>/pmm.git
    cd pmm/documentation
    ```

    !!! note "Using SSH?"
    
        If you have SSH keys set up, use `git@github.com:<your_github_name>/pmm.git` instead.

3. Add the upstream repository so you can sync with the latest changes:

    ```shell
    git remote add upstream https://github.com/percona/pmm.git
    ```

4. Check out the right branch and pull the latest changes:

    ```shell
    git checkout v3
    git pull upstream v3
    ```

    !!! note "Which branch?"
    
        Use `v3` for PMM 3.x docs or `main` for PMM 2.x docs. Git will create a tracking branch automatically if it doesn't exist locally.

5. Create a new branch for your work:

    ```shell
    git checkout -b <my_changes>
    ```

6. Make your edits in the `documentation/docs` directory. Add code examples if needed. You can preview your changes using your editor's built-in preview or by [building the docs locally](#building-the-documentation).

7. Stage your changes:

    ```shell
    git add documentation/docs/example.md
    ```

8. Commit with a descriptive message:

    ```shell
    git commit -m 'Fixed typo in setting-up.md'
    ```

9. Push to your fork:

    ```shell
    git push -u origin <my_changes>
    ```

10. GitHub will show a **Compare & pull request** button - click it to open your PR. Or navigate to your fork and click **Create pull request**.

### Building the documentation

Want to see how your changes will look on the live site? You can build and preview the docs locally using MkDocs.

!!! note "What you'll need"
    
    Python 3.x and Docker. Don't have them? Grab [Python](https://www.python.org/downloads/) and [Docker](https://docs.docker.com/get-docker/) first.

#### With Docker

Docker is the easiest option since it bundles everything you need:

1. From the `pmm` directory (repo root), run:

    ```shell
    docker run --rm -v $(pwd):/docs perconalab/pmm-doc-md mkdocs build -f documentation/mkdocs.yml
    ```

2. Open `site/index.html` in your browser to view the docs.

#### Live preview with Docker

Want to see changes as you type? Use the built-in live preview server:

1. Run:

    ```shell
    docker run --rm -v $(pwd):/docs -p 8000:8000 perconalab/pmm-doc-md mkdocs serve --dev-addr=0.0.0.0:8000 -f documentation/mkdocs.yml
    ```

2. Wait for `INFO - Start detecting changes`, then open `http://0.0.0.0:8000` in your browser. MkDocs will automatically reload whenever you save changes.

#### Without Docker

If you prefer not to use Docker, you can install MkDocs directly:

1. Navigate to `pmm/documentation`.

2. Install MkDocs and extensions:

    ```shell
    pip install -r requirements.txt
    ```

3. Build the docs:

    ```shell
    mkdocs build
    ```

4. Open `site/index.html` in your browser, or start the live preview server:

    ```shell
    mkdocs serve
    ```

5. Browse to `http://127.0.0.1:8000/` to see your changes. The server reloads automatically when you edit files.

6. Your changes will appear at the same path as in the `documentation/docs` directory.

### Building the PDF

Need a PDF version? Here's how to generate one:

1. (Percona staff only) If this is for a release, update `mkdocs-base.yml`:
    * Change the release number in `plugins.with-pdf.output_path`
    * Update the release number and date in `plugins.with-pdf.cover_subtitle`

2. Build the PDF:

    * With Docker:

        ```shell
        docker run --rm -v $(pwd):/docs -e ENABLE_PDF_EXPORT=1 perconalab/pmm-doc-md mkdocs build -f documentation/mkdocs-pdf.yml
        ```

    * Without Docker:

        ```shell
        ENABLE_PDF_EXPORT=1 mkdocs build -f mkdocs-pdf.yml
        ```

3. Find your PDF in `site/pdf`.