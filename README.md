# Percona Monitoring and Management (PMM) management daemon

[![Build Status](https://travis-ci.org/percona/pmm-managed.svg?branch=master)](https://travis-ci.org/percona/pmm-managed)
[![Go Report Card](https://goreportcard.com/badge/github.com/percona/pmm-managed)](https://goreportcard.com/report/github.com/percona/pmm-managed)
[![CLA assistant](https://cla-assistant.io/readme/badge/percona/pmm-managed)](https://cla-assistant.io/percona/pmm-managed)

pmm-managed manages configuration of [PMM](https://www.percona.com/doc/percona-monitoring-and-management/index.html)
server components (Prometheus, Grafana, etc.) and exposes API for that. Those APIs are used by
[pmm-admin tool](https://github.com/percona/pmm-client).
