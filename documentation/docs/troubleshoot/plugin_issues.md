
# Plugin issues

## PMM does not allow to install, upgrade or remove plugins

Users have encountered issues with installing, updating and removing plugins from PMM. The cause of this issue is the incorrect permissions assigned to the `/srv/grafana/plugins` directory. These permissions are preventing the grafana component from writing to the directory.

## Solution

Set the ownership on the directory`/srv/grafana/plugins` to `grafana:grafana`.