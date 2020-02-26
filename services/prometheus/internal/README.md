Files in this directory were extracted from the Prometheus project:

* https://github.com/prometheus/common
* https://github.com/prometheus/prometheus

Exact version see in files headers.

We have them there for three reasons:

* That's a *huge* dependency, but we need only a small part of it.
* `dep` crashes trying to vendor it.
* We need a way to read passwords without custom secrets handling to be able to compare the new configuration file with the old one to know if we need to reload Prometheus configuration. If we read `***` instead of passwords, we will always think that configuration file changed.

Right now we use only StaticConfig, but that may change in the future.
