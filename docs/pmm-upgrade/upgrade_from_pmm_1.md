# Upgrade from PMM 1

Because of the significant architectural changes between PMM1 and PMM2, there is no direct upgrade path. The approach to making the switch from PMM version 1 to 2 is a gradual transition, outlined in [this blog post](https://www.percona.com/blog/running-pmm1-and-pmm2-clients-on-the-same-host/).

In short, it involves first standing up a new PMM2 server on a new host and connecting clients to it. As new data is reported to the PMM2 server, old metrics will age with the retention period (30 days, by default), at which point you’ll be able to shut down your existing PMM1 server.

Any alerts configured through the Grafana UI will have to be recreated due to the target dashboard id’s not matching between PMM1 and PMM2. In this instance we recommend moving to Alertmanager recipes in PMM2 for alerting which, for the time being, requires a separate Alertmanager instance. We are working on integrating this natively into PMM2 Server and expect to support your existing Alertmanager rules.