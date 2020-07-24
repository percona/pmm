# Navigating across Dashboards

Beside the *Dashboard Dropdown* button you can also Navigate across
Dashboards with the navigation menu which groups dashboards by
application. Click the required group and then select the dashboard
that matches your choice.


* PMM Query Analytics: See The Query Analytics Dashboard


* OS: The operating system status


* MySQL: MySQL and Amazon Aurora


* MongoDB: State of MongoDB hosts


* HA: High availability


* Cloud: Amazon RDS and Amazon Aurora


* Insight: Summary, cross-server and Prometheus


* PMM: Server settings



![image](/_images/metrics-monitor.menu.png)

## Zooming in on a single metric

On dashboards with multiple metrics, it is hard to see how the value of a single
metric changes over time. Use the context menu to zoom in on the selected metric
so that it temporarily occupies the whole dashboard space.

Click the title of the metric that you are interested in and select the
*View* option from the context menu that opens.



![image](/_images/metrics-monitor.metric-context-menu.1.png)

The selected metric opens to occupy the whole dashboard space. You may now set
another time range using the time and date range selector at the top of the
Metrics Monitor page and analyze the metric data further.



![image](/_images/metrics-monitor.cross-server-graphs.load-average.1.png)

To return to the dashboard, click the *Back to dashboard* button next to the time range selector.

The *Back to dashboard* button returns to the dashboard; this button appears when you are zooming in on one metric.



![image](/_images/metrics-monitor.time-range-selector.1.png)

Navigation menu allows you to navigate between dashboards while maintaining the
same host under observation and/or the same selected time range, so that for
example you can start on *MySQL Overview* looking at host serverA, switch to
MySQL InnoDB Advanced dashboard and continue looking at serverA, thus saving you
a few clicks in the interface.
