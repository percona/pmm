# Share dashboards

When you need to share a dashboard with your team members, you can either send them a direct link to the dashboard, or render and send the dashboard as a .PNG image.

## Share as direct link
1. Go to the dashboard that you want to share.
2. Click at the top of the dashboard to display the panel menu.
3. Select **Share** to reveal the **Share Panel** and either:  
   - copy and send the full URL for the dashboard, OR
   - toggle the **Short URL** option to generate a simple link with a unique identifier

!!! hint alert alert-success "Tip"
       If your current domain is different than the one specified in the Grafana .INI configuration file, PMM will ask you to correct this mismatch before you can generate a short URL:
    ![!image](../images/PMM_Common_Panel_Menu_Share.png)
    To fix this
    
## Share as a PNG file

Rendering images requires the Image Renderer plug-in. If your PMM Admin has not installed this for your PMM instance, you will see the following error message under **Share Panel > Link**.
![!image](../images/No_Image_Render_Plugin.png)

To install the dependencies:

1. Connect to your PMM Server Docker container.

    ```sh
    docker exec -it pmm-server bash
    ```

2. Install Grafana plug-ins.

    ```sh
    grafana-cli plugins install grafana-image-renderer
    ```

3. Restart Grafana.

    ```sh
    supervisorctl restart grafana
    ```

4. Install libraries.

    ```sh
    yum install -y libXcomposite libXdamage libXtst cups libXScrnSaver pango \
    atk adwaita-cursor-theme adwaita-icon-theme at at-spi2-atk at-spi2-core \
    cairo-gobject colord-libs dconf desktop-file-utils ed emacs-filesystem \
    gdk-pixbuf2 glib-networking gnutls gsettings-desktop-schemas \
    gtk-update-icon-cache gtk3 hicolor-icon-theme jasper-libs json-glib \
    libappindicator-gtk3 libdbusmenu libdbusmenu-gtk3 libepoxy \
    liberation-fonts liberation-narrow-fonts liberation-sans-fonts \
    liberation-serif-fonts libgusb libindicator-gtk3 libmodman libproxy \
    libsoup libwayland-cursor libwayland-egl libxkbcommon m4 mailx nettle \
    patch psmisc redhat-lsb-core redhat-lsb-submod-security rest spax time \
    trousers xdg-utils xkeyboard-config alsa-lib
    ```

To render the image: 

1. Go to the dashboard that you want to share.
2. Click at the top of the dashboard to display the panel menu.
3. Select **Share** to reveal the **Share Panel**.
4. Click **Direct link rendered image**. This opens a new browser tab.
5. Wait for the image to be rendered, then use your browser's Image Save function to download the image.