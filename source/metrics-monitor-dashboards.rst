########################
Understanding Dashboards
########################

The Metrics Monitor tool provides a historical view of metrics that are critical to a database server. Time-based
graphs are separated into dashboards by themes: some are related to MySQL or
MongoDB, others provide general system metrics.

.. _pmm.metrics-monitor.dashboard.opening:

*******************
Opening a Dashboard
*******************

The default PMM installation provides more than thirty dashboards. To make it
easier to reach a specific dashboard, the system offers two tools. The
*Dashboard Dropdown* is a button in the header of any PMM page. It lists
all dashboards, organized into folders. Right sub-panel allows to rearrange
things, creating new folders and dragging dashboards into them. Also a text box
on the top allows to search the required dashboard by typing.

With *Dashboard Dropdown*, search the alphabetical list for any dashboard.

.. image:: /_images/metrics-monitor.dashboard-dropdown.png

.. _pmm.metrics-monitor.graph-description:

**************************************
Viewing More Information about a Graph
**************************************

Each graph has a descriptions to display more information about the monitored
data without cluttering the interface.

These are on-demand descriptions in the tooltip format that you can find by
hovering the mouse pointer over the *More Information* icon at the top left
corner of a graph. When you move the mouse pointer away from the *More Information*
button the description disappears.

Graph descriptions provide more information about a graph without claiming any space in the interface.

.. image:: /_images/metrics-monitor.description.1.png

**See also**

:ref:`Selecting time or date range <pmm.qan.time-date-range.selecting>`

**************************
Rendering Dashboard Images
**************************

PMM Server can't currently directly render dashboard images exported by Grafana without these additional set-up steps.

**Part 1: Install dependencies**

1. Connect to your PMM Server Docker container.

   .. code-block:: sh

      docker exec -it pmm-server bash

2. Install Grafana plugins.

   .. code-block:: sh

      grafana-cli plugins install grafana-image-renderer

3. Restart Grafana.

   .. code-block:: sh

      supervisorctl restart grafana

4. Install additional libraries.

   .. code-block:: sh

      yum install -y libXcomposite libXdamage libXtst cups libXScrnSaver pango atk adwaita-cursor-theme adwaita-icon-theme at at-spi2-atk at-spi2-core cairo-gobject colord-libs dconf desktop-file-utils ed emacs-filesystem gdk-pixbuf2 glib-networking gnutls gsettings-desktop-schemas gtk-update-icon-cache gtk3 hicolor-icon-theme jasper-libs json-glib libappindicator-gtk3 libdbusmenu libdbusmenu-gtk3 libepoxy liberation-fonts liberation-narrow-fonts liberation-sans-fonts liberation-serif-fonts libgusb libindicator-gtk3 libmodman libproxy libsoup libwayland-cursor libwayland-egl libxkbcommon m4 mailx nettle patch psmisc redhat-lsb-core redhat-lsb-submod-security rest spax time trousers xdg-utils xkeyboard-config alsa-lib

**Part 2 - Share your dashboard image**

1. Navigate to the dashboard you want to share.

2. Open the panel menu (between the PMM main menu and the navigation breadcrumbs).

   .. image:: /_images/PMM_Common_Panel_Menu_Open.jpg

3. Select *Share* to reveal the *Share Panel*.

   .. image:: /_images/PMM_Common_Panel_Menu_Share.jpg

4. Click *Direct link rendered image*.

5. A new browser tab opens. Wait for the image to be rendered then use your browser's image save function to download the image.


If the necessary plugins are not installed, a message in the Share Panel will say so.

.. image:: /_images/PMM_Common_Panel_Menu_Share_Link_Missing_Plugins.jpg
