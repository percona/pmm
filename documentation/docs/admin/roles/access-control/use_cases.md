# Use Cases

An overview of the infrastructure can be seen in the diagram below. PMM monitors several services. The metrics that are stored in VictoriaMetrics have appropriate labels, for example, **environment** and **region**.

  ![PMM Access Control - Metrics collection](../../../images/lbac/pmm-lbac-collect-metrics.jpg)


## Use case 1 - Simple selectors

This use case demonstrates the following scenario:

**Roles**

- Admin
- DBA
- QA

**Labels**

- environment
  - prod
  - qa
- region
  - us-east
  - emea

The diagram below shows several roles within a company structure that have access to data in PMM, as well as their access to a subset of metrics:

- Admin role: access to all metrics, in both **prod** and **qa** environments and in both regions
- DBA role: access to metrics within **environment=prod** only, but in both regions
- QA role has access to metrics within **environment=qa** only, but in both regions

  ![PMM Access Control - Roles](../../../images/lbac/pmm-lbac-query-metrics-1.jpg)


## Use case 2 - Compound selectors

This use case is a modification of the prior one, it demonstrates the following scenario:

**Roles**

- Admin
- DBA
- QA

**Labels**

- environment
  - prod
  - qa
- region
  - us-east
  - emea

The diagram below shows several roles within a company structure that have access to PMM, as well as the permissions they should be granted:

- Admin role: access to all metrics, in both **prod** and **qa** environments, and in both regions
- DBA role: has access to metrics within **environment=prod** only, but in both regions
- QA role: has access to metrics within **environment=qa** only, but in both regions

  ![PMM Access Control - Roles](../../../images/lbac/pmm-lbac-query-metrics-2.jpg)


|            |**Role assigned**|**Labels applied to the role**|**Accessible Metrics** |
|------------|-----------------|------------------------------|-----------------------|
| **User 1** | role_postresql|dev, service_name=postgresql|The metrics for service postgresql will be accessible.|
| **User 2** | role_mysql    |prod, service_name=mysql|The metrics for service mysql will be accessible.|
| **User 3** | role_postgresql and role_mysql|dev, service_name=postgresql and </br> prod, service_name=mysql |The metrics for both the services mysql and postresql will be accessible.|
