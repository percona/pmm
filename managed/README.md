# Percona Monitoring and Management (PMM) management daemon

[![Build Status](https://github.com/percona/pmm-managed/workflows/CI/badge.svg?branch=main)](https://github.com/percona/pmm-managed/actions?query=workflow%3ACI+branch%3Amain)
[![codecov.io Code Coverage](https://codecov.io/gh/percona/pmm-managed/branch/main/graph/badge.svg)](https://codecov.io/github/percona/pmm-managed?branch=main)
[![Go Report Card](https://goreportcard.com/badge/github.com/percona/pmm-managed)](https://goreportcard.com/report/github.com/percona/pmm-managed)
[![CLA assistant](https://cla-assistant.percona.com/readme/badge/percona/pmm-managed)](https://cla-assistant.percona.com/percona/pmm-managed)

pmm-managed manages configuration of [PMM](https://www.percona.com/doc/percona-monitoring-and-management/index.html)
server components (VictoriaMetrics, Grafana, etc.) and exposes API for that. Those APIs are used by
[pmm-admin tool](https://github.com/percona/pmm-admin).
