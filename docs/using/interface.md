# User Interface

You can access the PMM web interface using the IP address of the host where
PMM Server is running.  For example, if PMM Server is running on a host with
IP 192.168.100.1, access the following address with your web browser:
`http://192.168.100.1`.

The PMM home page that opens provides an overview of the environment that you
have set up to monitor by using the `pmm-admin` tool.

From the PMM home page, you can access specific monitoring tools, or
dashboards. Each dashboard features a collection of metrics. These are graphs of
a certain type that represent one specific aspect showing how metric values
change over time.



![image](../_images/PMM_Home_Dashboard.jpg)

By default the PMM home page lists most recently used dashboards and helpful
links to the information that may be useful to understand PMM better.

The PMM home page lists all hosts that you have set up for monitoring as well
as the essential details about their performance such as CPU load, disk
performance, or network activity.



## Understanding Dashboards

The Metrics Monitor tool provides a historical view of metrics that are critical to a database server. Time-based graphs are separated into dashboards by themes: some are related to MySQL or MongoDB, others provide general system metrics.

## Opening a Dashboard

The default PMM installation provides more than thirty dashboards. To make it
easier to reach a specific dashboard, the system offers two tools. The
*Dashboard Dropdown* is a button in the header of any PMM page. It lists
all dashboards, organized into folders. Right sub-panel allows to rearrange
things, creating new folders and dragging dashboards into them. Also a text box
on the top allows to search the required dashboard by typing.

With *Dashboard Dropdown*, search the alphabetical list for any dashboard.

![image](../_images/metrics-monitor.dashboard-dropdown.png)

## Viewing More Information about a Graph

Each graph has a descriptions to display more information about the monitored
data without cluttering the interface.

These are on-demand descriptions in the tooltip format that you can find by
hovering the mouse pointer over the *More Information* icon at the top left
corner of a graph. When you move the mouse pointer away from the *More Information*
button the description disappears.

Graph descriptions provide more information about a graph without claiming any space in the interface.

![image](../_images/metrics-monitor.description.1.png)


## Rendering Dashboard Images

PMM Server can't currently directly render dashboard images exported by Grafana without these additional set-up steps.

**Part 1: Install dependencies**

1. Connect to your PMM Server Docker container.

    ```sh
    docker exec -it pmm-server bash
    ```

2. Install Grafana plugins.

    ```sh
    grafana-cli plugins install grafana-image-renderer
    ```

3. Restart Grafana.

    ```sh
    supervisorctl restart grafana
    ```

4. Install additional libraries.

    ```sh
    yum install -y libXcomposite libXdamage libXtst cups libXScrnSaver pango atk adwaita-cursor-theme adwaita-icon-theme at at-spi2-atk at-spi2-core cairo-gobject colord-libs dconf desktop-file-utils ed emacs-filesystem gdk-pixbuf2 glib-networking gnutls gsettings-desktop-schemas gtk-update-icon-cache gtk3 hicolor-icon-theme jasper-libs json-glib libappindicator-gtk3 libdbusmenu libdbusmenu-gtk3 libepoxy liberation-fonts liberation-narrow-fonts liberation-sans-fonts liberation-serif-fonts libgusb libindicator-gtk3 libmodman libproxy libsoup libwayland-cursor libwayland-egl libxkbcommon m4 mailx nettle patch psmisc redhat-lsb-core redhat-lsb-submod-security rest spax time trousers xdg-utils xkeyboard-config alsa-lib
    ```

**Part 2 - Share your dashboard image**

1. Navigate to the dashboard you want to share.

2. Open the panel menu (between the PMM main menu and the navigation breadcrumbs).

    ![image](../_images/PMM_Common_Panel_Menu_Open.jpg)

3. Select *Share* to reveal the *Share Panel*.

    ![image](../_images/PMM_Common_Panel_Menu_Share.jpg)

4. Click *Direct link rendered image*.

5. A new browser tab opens. Wait for the image to be rendered then use your browser's image save function to download the image.


If the necessary plugins are not installed, a message in the Share Panel will say so.

![image](../_images/PMM_Common_Panel_Menu_Share_Link_Missing_Plugins.jpg)



## Navigating across Dashboards

Beside the *Dashboard Dropdown* button you can also Navigate across
Dashboards with the navigation menu which groups dashboards by
application. Click the required group and then select the dashboard
that matches your choice.

* PMM Query Analytics
* OS: The operating system status
* MySQL: MySQL and Amazon Aurora
* MongoDB: State of MongoDB hosts
* HA: High availability
* Cloud: Amazon RDS and Amazon Aurora
* Insight: Summary, cross-server and Prometheus
* PMM: Server settings

![image](../_images/metrics-monitor.menu.png)

## Zooming in on a single metric

On dashboards with multiple metrics, it is hard to see how the value of a single
metric changes over time. Use the context menu to zoom in on the selected metric
so that it temporarily occupies the whole dashboard space.

Click the title of the metric that you are interested in and select the
*View* option from the context menu that opens.

![image](../_images/metrics-monitor.metric-context-menu.1.png)

The selected metric opens to occupy the whole dashboard space. You may now set
another time range using the time and date range selector at the top of the
Metrics Monitor page and analyze the metric data further.

![image](../_images/metrics-monitor.cross-server-graphs.load-average.1.png)

To return to the dashboard, click the *Back to dashboard* button next to the time range selector.

The *Back to dashboard* button returns to the dashboard; this button appears when you are zooming in on one metric.

![image](../_images/metrics-monitor.time-range-selector.1.png)

Navigation menu allows you to navigate between dashboards while maintaining the
same host under observation and/or the same selected time range, so that for
example you can start on *MySQL Overview* looking at host serverA, switch to
MySQL InnoDB Advanced dashboard and continue looking at serverA, thus saving you
a few clicks in the interface.


## Annotations

The `pmm-admin annotate` command registers a moment in time, marking it with a text string called an *annotation*.

The presence of an annotation shows as a vertical dashed line on a dashboard graph; the annotation text is revealed by mousing over the caret indicator below the line.

Annotations are useful for recording the moment of a system change or other significant application event.

They can be set globally or for specific nodes or services.

![image](../_images/pmm-server.mysql-overview.mysql-client-thread-activity.1.png)

**USAGE**

`pmm-admin annotate [--node|--service] <annotation> [--tags <tags>] [--node-name=<node>] [--service-name=<service>]`

**OPTIONS**

`<annotation>`
: The annotation string. If it contains spaces, it should be quoted.

`--node`
: Annotate the current node or that specified by `--node-name`.

`--service`
: Annotate all services running on the current node, or that specified by `--service-name`.

`--tags`
: A quoted string that defines one or more comma-separated tags for the annotation. Example: `"tag 1,tag 2"`.

`--node-name`
: The node name being annotated.

`--service-name`
: The service name being annotated.

### Combining flags

Flags may be combined as shown in the following examples.

`--node`
: current node

`--node-name`
: node with name

`--node --node-name=NODE_NAME`
: node with name

`--node --service-name`
: current node and service with name

`--node --node-name --service-name`
: node with name and service with name

`--node --service`
: current node and all services of current node

`-node --node-name --service --service-name`
: service with name and node with name

`--service`
: all services of the current node

`--service-name`
: service with name

`--service --service-name`
: service with name

`--service --node-name`
: all services of current node and node with name

`--service-name --node-name`
: service with name and node with name

`--service --service-name -node-name`
: service with name and node with name

!!! note
    If node or service name is specified, they are used instead of other parameters.

### Visibility

You can toggle the display of annotations on graphs with the *PMM Annotations* checkbox.

![image](../_images/pmm-server.pmm-annotations.png)

Remove the check mark to hide annotations from all dashboards.

!!! seealso "See also"

    * [docs.grafana.org: Annotations](http://docs.grafana.org/reference/annotations/)
