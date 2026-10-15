# Annotation in dashboards

Annotations mark a moment in time. They are useful for marking system changes or other significant application events. They can be set globally or for specific nodes or services. Annotations for nodes and services are called scoped annotations.

You create them on the command line with the [`pmm-admin annotate` command](../../commands/pmm-admin/config.md#pmm-admin-annotate), or with the `POST /v1/management/annotations` API endpoint. To scope an annotation, use the `--node`, `--node-name`, `--service`, or `--service-name` flag, or the `node_name` or `service_names` field of the API request.

Annotations show as a vertical dashed line on a dashboard graph. Reveal the annotation text by mousing over the caret indicator below the line.

![!image](../../../images/pmm-server.mysql-overview.mysql-client-thread-activity.1.png)

## Show annotations on dashboards

You turn annotations on or off with the switches in the second row menu bar. Dashboards for nodes and services have two:

- **PMM Annotations**: shows global annotations
- **PMM Annotations: nodes and services**: shows scoped annotations

On most dashboards, both switches are on by default.

![PMM Annotations switch](../../../images/PMM_Common_Controls_Annotation_Toggle.jpg)

## Scoped annotations on dashboards

Scoped annotations show when the **PMM Annotations: nodes and services** switch is on, including when the node or service variables are set to **All**. Which scoped annotations a dashboard shows depends on the type of dashboard:

- The OS dashboards, such as **Node Summary**, and the **Home Dashboard** show node annotations.
- The service dashboards, such as **MySQL Instance Summary** and the Valkey dashboards, show service annotations.
- Some overview dashboards, such as **MySQL Instances Overview** and **PostgreSQL Instances Overview Extended**, show both node and service annotations.

A few dashboards, such as **ProxySQL Instance Summary**, don't have the **PMM Annotations: nodes and services** switch and show global annotations only.

The other variables on the dashboard, such as environment or region, filter the annotations in the same way that they filter the graphs. If you select one node on a dashboard that shows node annotations, or one service on a dashboard that shows service annotations, the dashboard shows only the annotations of that node or service. On a dashboard that shows both, selecting a service narrows the service annotations but not the node annotations.

## Access control

Label-based access control (LBAC) applies to scoped annotations on dashboards. On a dashboard, users see the scoped annotations of the nodes and services whose metrics they can see. For more information, see [Access control in PMM](../../../admin/roles/access-control/intro.md).

## Retention

Scoped annotations show on dashboards for the metrics data retention period. By default, this is 30 days. To change the period, see [Data retention](../../../configure-pmm/advanced_settings.md#data-retention).

## Find the ID of an annotation

Each scoped annotation has an ID. The ID shows in:

- the output of `pmm-admin annotate`
- the `annotation_id` field of the response from the `POST /v1/management/annotations` API endpoint
- the title of the annotation marker on a dashboard, in the form `Annotation <id>`

For example, the following command prints the ID of the new annotation:

```shell
pmm-admin annotate "Schema migration completed" --service-name=mysql-prod
```

```text
Annotation added.
Annotation ID: 17
```

The marker of this annotation is titled `Annotation 17`.

## Delete a scoped annotation

To delete a scoped annotation, call the `DELETE /v1/management/annotations/{annotation_id}` API endpoint. This endpoint is available to users with the Admin role only. For how to authenticate, see [Authentication](../../../api/authentication.md).

For example, to delete the annotation with ID `17`:

```shell
curl -X DELETE -H 'Authorization: Bearer <SERVICE_TOKEN>' https://127.0.0.1/v1/management/annotations/17
```

If PMM deletes the annotation but can't remove it from dashboards, the endpoint returns an `Unavailable` error (HTTP status 503). PMM retries the removal every 5 minutes, so the annotation usually disappears from dashboards within about 5 minutes. The removal needs the VictoriaMetrics delete API, so the annotation stays on dashboards while PMM can't reach that API.

If you delete a scoped annotation with the Grafana annotations API instead, PMM removes it from dashboards in the same way, usually within about 5 minutes.

## Limitations

Scoped annotations have the following limitations:

- You can't edit or delete a scoped annotation from its marker on a dashboard.
- Only users with the Admin role can delete a scoped annotation through the PMM API.

To delete a scoped annotation, see [Delete a scoped annotation](#delete-a-scoped-annotation).

## Label limit

PMM stores a scoped annotation in VictoriaMetrics as a `pmm_annotation_event` series. The series carries the labels of the node or service, together with the `pmm_annotation_id`, `pmm_annotation_text`, and `pmm_annotation_tags` labels for the annotation ID, text, and tags. A scoped annotation can have at most 40 labels, and the metric name counts as one of them.

If a node or service carries so many labels that the annotation would exceed this limit, PMM rejects the request and doesn't create the annotation. For example:

```text
Cannot annotate service 'mysql-prod': too many labels: 45 labels, at most 40 are allowed.
```

To fix this error, remove custom labels from the node or service, then create the annotation again.
