# Upgrade

!!! caution alert alert-warning "Important"
    Upgrade PMM Server before upgrading PMM Clients.

## Updating a Server

Client and server components are installed and updated separately.

PMM Server can run natively, as a Docker image, a virtual appliance, or an AWS cloud instance. Each has its own installation and update steps.

The preferred and simplest way to update PMM Server is with the *PMM Upgrade* panel on the Home page.

![!image](../_images/PMM_Home_Dashboard_Panels_Upgrade.jpg)

The panel shows:

- the current server version and release date;
- whether the server is up to date;
- the last time a check was made for updates.

Click the refresh button to manually check for updates.

If one is available, click the update button to update to the version indicated.

!!! seealso alert alert-info "See also"
    [PMM Server Docker upgrade](../setting-up/server/docker.md#upgrade)

## Upgrade from PMM 1 {: #upgrade-from-pmm1}

Because of the significant architectural changes between PMM1 and PMM2, there is no direct upgrade path. The approach to making the switch from PMM version 1 to 2 is a gradual transition, outlined [in this blog post](https://www.percona.com/blog/2019/11/27/running-pmm1-and-pmm2-clients-on-the-same-host/).

In short, it involves first standing up a new PMM2 server on a new host and connecting clients to it.  As new data is reported to the PMM2 server, old metrics will age out during the course of the retention period (30 days, by default), at which point you'll be able to shut down your existing PMM1 server.

Any alerts configured through the Grafana UI will have to be recreated due to the target dashboard id's not matching between PMM1 and PMM2.  In this instance we recommend moving to Alertmanager recipes in PMM2 for alerting which, for the time being, requires a separate Alertmanager instance. However, we are working on integrating this natively into PMM2 Server and expect to support your existing Alertmanager rules.
