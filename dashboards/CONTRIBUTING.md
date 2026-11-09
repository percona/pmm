Welcome to the repository hosting grafana dashboards for PMM!

We encourage contributions and are always looking for new members that are as dedicated to serving the community as we are.

You can also reach us on our [Forums](https://forums.percona.com).

## Prerequisites

Before submitting code contributions or documentation updates, you should first complete the following prerequisites.

### Sign the CLA

Before you can contribute, we kindly ask you to sign our [Contributor License Agreement](https://cla-assistant.percona.com/percona/pmm) (CLA). You can do this using your GitHub account and one click.

## Submitting a Bug

If you find a bug in Percona MongoDB Exporter or one of the related projects, you should submit a report to that project's [JIRA](https://perconadev.atlassian.net) issue tracker.

Your first step should be [to search](https://perconadev.atlassian.net/issues/?jql=project=PMM%20AND%20component=MongoDB_Exporter) the existing set of open tickets for a similar report. If you find that someone else has already reported your problem, then you can upvote that report to increase its visibility.

If there is no existing report, submit a report following these steps:

1. [Sign in to Percona JIRA.](https://perconadev.atlassian.net/login.jsp) You will need to create an account if you do not have one.
2. [Go to the Create Issue screen.](https://perconadev.atlassian.net/projects/PMM/issues)
3. Fill in the fields of Summary, Description, Steps To Reproduce, and Affects Version to the best you can. If it is a bug, attach the stack trace from the logs.

An excellent resource is [Elika Etemad's article on filing good bug reports.](http://fantasai.inkedblade.net/style/talks/filing-good-bugs/).

As a general rule of thumb, please try to create bug reports that are:

- _Reproducible._ Include steps to reproduce the problem.
- _Specific._ Include as much detail as possible: which version, what environment, etc.
- _Unique._ Do not duplicate existing tickets.
- _Scoped to a Single Bug._ One bug per report.

## Setup your local development environment

### In the PMM devcontainer (recommended)

The PMM devcontainer ships Node 22 + Yarn and wires the QAN plugin straight into the bundled Grafana. From the repo root **on the host**:

```bash
make env-up      # first run only; reuses the container afterwards
make env         # shell into the container
```

Then **inside the container**:

```bash
make run-qan-ui
```

`run-qan-ui` runs `yarn install` under `dashboards/pmm-app`, symlinks `dashboards/pmm-app/dist` into `/srv/grafana/plugins/pmm-app/dist`, injects the livereload snippet into Grafana's `index.html` via `setup-livereload`, and starts the webpack watcher.

Open `https://localhost/graph/d/pmm-qan/` — the livereload server on port `35730` reloads the page whenever webpack finishes a rebuild.

For a one-off build (no watch), use `make build-qan-ui` instead.

### Standalone with Docker Compose

If you'd rather not run the PMM devcontainer, the `pmm-app` folder ships a minimal Docker Compose setup that bundles enough tooling to populate the panels with sample data:

```
cd pmm-app
docker-compose up -d
yarn dev
```

For a much simpler development environment you could run `docker compose up -d pmm-server` just to start setup the PMM server inside of `pmm-app` folder.

Please note, that we map the `./pmm-app/dist` folder as a subfolder of `/var/lib/grafana/plugins` so that front-end artifacts,
i.e. panels and dashboards, can be picked up by grafana server running in the docker container.

Then to run it:

```bash
cd pmm-app
docker-compose -f ./docker-compose.local.yml up -d
yarn dev
```

## Submitting a Pull Request

1.  Find the task in [JIRA](https://jira.percona.com/issues/?jql=project+%3D+PMM+AND+component+%3D+%22Grafana+Dashboards%22) or issue here in github. If no a similar task is found, please create a new task. Use a component filter Grafana Dashboards.

2.  You need to make a fork of our repository in your GitHub account.

3.  Make a clone of your repository on your computer.

4.  Create a new branch for your task. We recommend as a best practice to mention an issue number or just short desc in a branch name.

    Example:

        PMM-5053-add-missed-tooltips

5.  Make changes to the code in your branch.

6.  Make a commit. It is essential to provide a meaningful description. Use the following formula: "[JIRA_ISSUE_ID] What is being done."

    Example:

        git add .
        git commit -m "PMM-5053 Add a tooltip for Head Block widget on Prometheus Dashboard"

7.  Push your branch into your repository. Check that your branch only contains code relevant to the issue.

    Example:

        git push origin PMM-5053-dbazhenov-tooltip

8.  Make a Pull Request from your branch to the right branch in percona/grafana-dashboards.

    Example:

        from dbazhenov:PMM-5053-dbazhenov-tooltip to percona:main

### Code Reviews

9. After submitting your PR please add `pmm-review-fe` team as a reviewer - that would auto assign reviewers to your PR.

10. Your Pull Request must pass certain checks, such as CI linters, and Contributor License Agreement.
    You need to open the Contributor License Agreement page, read it, and accept it.

11. Wait for our experts to review your code. You may need to answer questions or to address requests for changes.

12. Our Engineers will merge your branch into the release branch.

## After your Pull Request is merged

Once your pull request is merged, you are an official Percona Community Contributor. Welcome to the community!

We're looking forward to your contributions and hope to hear from you soon on our [Forums](https://forums.percona.com).
