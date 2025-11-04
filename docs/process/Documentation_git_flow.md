# PMM documentation Git workflow process

This topic explains how we manage the Percona Monitoring and Management (PMM) documentation across different Git branches and product releases.

It outlines our workflow for handling new features and updates and hotfixes to ensure the [documentation site](https://docs.percona.com/percona-monitoring-and-management/) remains accurate, current, and aligned with the latest PMM release.

## Branch structure

- v3: The main production branch for documentation. This is the source for the live documentation site deployed via Render.com. 

- pmm-doc-* – Version-specific release branches. For example,`pmm-doc-3.2.0`, `pmm-doc-3.3.0`. These branches are created from `v3` and contain only the documentation changes for that specific release.

Add all documentation changes for the upcoming release to the corresponding `pmm-doc-* `branch.
Once the release is finalized, merge those updates back into `v3`.

## Feature development

When new PMM features are developed, the documentation team works alongside developers to make sure all updates are ready for release:

1. **PMM development start:** Developers begin work on a new feature in the `v3` branch and create a dedicated feature branch (for example, `PMM-1234-feature`).

2. **Documentation branch**: When a feature needs documentation, the doc team creates a separate branch for the related updates.

3. **Pull request:** The documentation PR should target the corresponding `pmm-doc-*` branch for the release that will include the feature.

4. **Feature Merge:** Once a feature is completed and reviewed, the feature branch is merged back into `v3`. The documentation branch is merged into the relevant `pmm-doc-*` branch. 

This keeps `v3` up-to-date with completed code changes, while documentation for unreleased features stays isolated until the release is published.

## Release flow

We align the documentation release with PMM's product release cycle so that the documentation always reflects PMM's current release.

1. **Create the release branch**: When preparing a new release, create a version branch from `v3`, as the basis for all documentation work for this release. For example: `pmm-doc-3.2.0`, `pmm-doc-3.3.0`. 

2. **Submit PRs**: All pull requests for that release should target the `pmm-doc-*` branch.

3. **Merge and prepare**: When the release is complete, merge changes from `pmm-doc-*` back into `v3`.

4. **Deploy:** Render.com automatically deploys everything from `v3`, reflecting the latest updates.

## Quick fixes 

For urgent or minor fixes:

1. Create a branch like `quick-fix-*` directly from `v3`.

2. Make the fix and merge it straight back into `v3` for immediate deployment.

## Automation

Automation helps streamline updates and deployments:

- **Release merges:** The merging of `pmm-doc-*` branches into `v3` during PMM 3 releases should also be automated. 

- **Continuous deployment**: Render.com automatically updates the live documentation site whenever `v3` changes.

## Common issues

Merging development changes into `v3` too early will cause unintended deployments. To prevent this, always use the proper branch (`pmm-doc-*`).

### Documentation workflow (v3.5+)

![image.png](../assets/docflow.png)