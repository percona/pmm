# PMM Documentation Git Workflow Process

# **Overview**

This document outlines the comprehensive workflow process for managing Percona Monitoring and Management (PMM) documentation across multiple branches and releases. It describes how feature development, releases, production documentation, and urgent fixes are coordinated to ensure the live site reflects the most current information.

## **1. Branch Structure**

### **Main Branches**

- v3 – Production documentation branch. This is the source of truth for the live documentation site deployed via Render.com. Documentation changes related to the upcoming release should go to pmm-doc-* branch until release is done and then be merged back to v3.
- pmm-doc-* – Version-specific release branches (e.g., pmm-doc-3.2.0, pmm-doc-3.3.0). These are cut from v3 with only this release specific changes.

## **2. Feature Development**

The feature development process is the initial stage of new PMM functionalities and their associated documentation.

1. **Development Start:** New feature development for PMM begins in the v3 branch.
2. **Feature Branch Creation:** For each individual update or new functionality, a feature branch (e.g., PMM-1234-feature) should be created, branching off from v3.
3. **Documentation Inclusion:** Documentation updates related to the new features should be developed in a separate branch and the PR for documentation changes should be created against. pmm-doc-* branch.
4. **Feature Merge:** Once a feature is completed and reviewed, the feature branch should be merged back into v3. The documentation branch should be merged to pmm-doc-* branch. This ensures that v3 always contains the latest completed features, but doesn’t include documentation changes that should be deployed to production website.

## **3. Release Lifecycle**

The documentation release process is tightly integrated with the PMM product release cycle, ensuring that the production documentation always reflects the current release state.

### **General Release Flow**

1. New release is created and contains tasks.
2. **Release Branch Creation:** A version-specific release branch (`pmm-doc-*`, e.g., `pmm-doc-3.2.0`, `pmm-doc-3.3.0`) should be created from v3. This branch serves as the basis for all documentation work for this release.
3. **PRs for Release Preparation:** During release preparation, all new Pull Requests (PRs) containing changes intended for the release should be based on and merged into the `pmm-doc-*` branch.
4. **Documentation Integration for Release:** Documentation changes relevant to the release should be integrated into v3 from the pmm-* branch. This process leverages automation for efficient updates.
5. **Deployment:** Render.com should continuously deploy the documentation from v3, reflecting the latest updates.

## **4. Quick Fix Process**

Urgent documentation issues or small corrections should be handled efficiently via a dedicated quick-fix workflow.

1. **Branch Creation:** Hotfix branches (quick-fix-*) should be created directly from v3.
2. **Direct Merge:** These quick-fix branches should be merged directly into v3 to expedite the deployment of urgent changes.

## **5. Automation**

Automation is a key component of the PMM documentation workflow, and these processes **should be implemented and maintained** to streamline updates and deployments.

- **Release Merges:** The merging of pmm-doc-* branches into v3 during PMM 3 releases **should also be automated**.
- **Continuous Deployment:** Render.com **is configured to continuously deploy** any changes pushed to v3, ensuring that the live documentation site is always current.

## Faced issues

- Accidental deployment of dev changes made in feature branches and merged to v3.

### New Documentation Workflow (v3.5+)

![image.png](../assets/docflow.png)