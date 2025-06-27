# Use Case

## Use case 1

This use case demonstrates the following scenario:

**Labels**

-  Environments: **prod** and **qa**

-  Projects: **shop** and **bank**

**Roles**

- Roles: Admin, Dev and QA

An overview of the infrastructure can be seen in the diagram below. PMM monitors several services. The metrics that are stored in VictoriaMetrics have the appropriate labels.

   ![!](../../../images/PMM_access_control_usecase_metrics.jpg)

 This diagram shows several roles within a company structure that have access to PMM, as well as the permissions they should be granted:

- Admin role - has access to all the metrics
- DBA role - has access to all metrics within **env=prod** only
- QA role - has access to all metrics within **env=qa** only

    ![!](../../../images/PMM_access_control_usecase_roles.jpg)


## Use case 2

The use case demonstrates the following scenario:

**Labels**

- Environments: prod and dev

- Services: postgresql and mysql

**Roles**

- role_postgresql
- role_mysql


|           |**Role assigned**|**Labels applied to the role**|**Accessible Metrics**                                                                                                  |
|----------|--------|---------------------------------------------- |-------------------------------------------------------------------------------------------------------------|
| **User 1**  | role_postgresql|dev, service_name=postgresql|The metrics for service postgresql will be accessible.|                                          
| **User 2**  | role_mysql    |prod, service_name=mysql|The metrics for service mysql will be accessible.|                                          
| **User 3**  | role_postgresql and role_mysql|dev, service_name=postgresql and </br> prod, service_name=mysql |The metrics for both the services mysql and postgresql will be accessible.|   
