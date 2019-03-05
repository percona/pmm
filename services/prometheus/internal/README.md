Files in this directory were extracted from the Prometheus project:

* https://github.com/prometheus/common/blob/v0.2.0/config/http_config.go
* https://github.com/prometheus/prometheus/blob/v2.7.1/config/config.go
* https://github.com/prometheus/prometheus/blob/v2.7.1/discovery/config/config.go
* https://github.com/prometheus/prometheus/blob/v2.7.1/discovery/targetgroup/targetgroup.go
* https://github.com/prometheus/prometheus/blob/v2.7.1/pkg/labels/labels.go
* https://github.com/prometheus/prometheus/blob/v2.7.1/pkg/relabel/relabel.go

We have them there for three reasons:

* That's a *huge* dependency, but we need only a small part of it.
* `dep` crashes trying to vendor it.
* We need a way to read passwords without custom secrets handling to be able to compare the new configuration file with the old one to know if we need to reload Prometheus configuration. If we read `***` instead of passwords, we will always think that configuration file changed.

Right now we use only StaticConfig, but that may change in the future.
