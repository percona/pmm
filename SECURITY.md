# Security Policy

## Supported Versions

Only PMM 3.x receives security fixes. PMM 1.x and 2.x are end of life.

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :x:                |
| 2.x.x   | :x:                |
| 3.x.x   | :white_check_mark: |

Fixes are delivered in the next PMM release; we do not backport to older minor versions.

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues, pull requests, or the forums.**

Report privately through [GitHub private vulnerability reporting](https://github.com/percona/pmm/security/advisories/new).

Please include as much of the following as you can:

- PMM version (`pmm-admin --version`, or the PMM Server image tag) and deployment type (Docker, OVF, AMI, Helm).
- The affected component (pmm-managed, pmm-agent, pmm-admin, an exporter, a bundled third-party package).
- Steps to reproduce, a proof of concept, and the impact you believe it has.
- Any CVE identifier if the issue is in a third-party dependency we ship.

Please give us a reasonable opportunity to fix the issue before any public disclosure, and coordinate the disclosure date with us.

## Remediation Timelines

Once a vulnerability is confirmed, we aim to deliver a fix within the following targets:

- Critical: immediately, and no later than 14 days
- High: as soon as possible, and no later than 30 days
- Medium: within 60 days
- Low: best effort within 90 days

The same targets apply to vulnerabilities in third-party software we package or distribute, measured from the date the upstream vendor makes a patch available.

Severity is assessed on the impact to PMM, so it may differ from the severity recorded by an upstream advisory or a scanner where compensating controls reduce the practical risk.

## Scope

In scope: the PMM components built in this repository (pmm-managed, pmm-agent, pmm-admin, qan-api2, vmproxy, the UI), the PMM Server distributions, and third-party components we package and ship.

Out of scope: Percona's websites and corporate infrastructure, issues that only affect unsupported versions, and findings that require an already-compromised host or PMM administrator credentials. Vulnerabilities in other Percona products should be reported against their own repositories.

## Security Contact

- [@aaronormerod](https://github.com/aaronormerod)
