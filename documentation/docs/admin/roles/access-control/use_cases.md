# Implementing LBAC: practical scenarios

Here are a few practical examples of how label-based access control can be implemented in PMM to meet specific organizational needs.

## Infrastructure overview
The diagram below shows a sample infrastructure monitored by PMM. Notice how the metrics stored in VictoriaMetrics include labels like **environment** and **region** that can be used for access control.

  <!-- source: https://miro.com/app/board/uXjVPfHchvM=/ -->
  ![PMM Access Control - Metrics collection](../../../images/lbac/pmm-lbac-collect-metrics.jpg)

## Use case 1: Simple selectors

This scenario demonstrates how to create three distinct roles with different levels of access:

![PMM Access Control - Basic Roles](../../../images/lbac/pmm-lbac-query-metrics-1.jpg)

| Role | Access needs | Label selectors | Effect |
|------|--------------|-----------------|--------|
| **Admin** | Complete visibility across all environments | `environment=prod` OR `environment=qa` | Full access to all metrics in both production and QA environments across all regions |
| **DBA** | Production database management | `environment=prod` | Access to all production metrics across all regions, but no visibility into QA |
| **QA** | Testing environment monitoring | `environment=qa` | Access to all QA metrics across all regions, but no visibility into production |

This approach allows for a clear separation of responsibilities while ensuring each team has access to exactly what they need.

## Use case 2 - Compound selectors

This advanced use case demonstrates how compound selectors create more granular access control by combining multiple label conditions using logical operators (AND, OR). 

By requiring matches on multiple labels simultaneously, you can implement sophisticated access patterns that reflect real-world organizational structures and security requirements.

<!-- source: https://miro.com/app/board/uXjVPfHchvM=/ -->
![PMM Access Control - Roles](../../../images/lbac/pmm-lbac-query-metrics-2.jpg)


| Role | Access needs | Label selectors | Effect |
|------|--------------|-----------------|--------|
| **Admin** | Complete visibility across all environments and regions | `environment=prod` OR `environment=qa` | Full access to all metrics in both production and QA environments across all regions |
| **DBA** | Production database management in EMEA region | `environment=prod` AND `region=emea` | Access only to production metrics in the EMEA region |
| **QA** | Testing environment monitoring in US-East region | `environment=qa` AND `region=us-east` | Access only to QA metrics in the US-East region |