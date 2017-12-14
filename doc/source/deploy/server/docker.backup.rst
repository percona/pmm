.. _pmm/docker/backup:

================================================================================
Backing Up and Restoring |pmm| data |docker| Container
================================================================================

When |pmm-server| is run via |docker|, its data are stored in the
|opt.pmm-data| container. To avoid data loss, you can extract them and
store outside of the container.

This example demonstrates how to back up |pmm| data on the computer
where the |docker| container is run.



#. Create a backup directory and make it the current working
   directory. In this example, we use *pmm-data-backup* as the
   directory name.

   .. code-block:: bash

      $ mkdir pmm-data-backup;cd pmm-data-backup

#. Create sub directories:

   .. code-block:: bash

      $ mkdir -p opt/prometheus
      # mkdir -p var/lib

|tip.run-all.root|

#. Stop the docker container:

   .. code-block:: bash

      $  docker stop pmm-server

#. Copy data from the |opt.pmm-data| container:

   .. code-block:: bash

      $ docker cp pmm-data:/opt/prometheus/data opt/prometheus/
      $ docker cp pmm-data:/opt/consul-data opt/
      $ docker cp pmm-data:/var/lib/mysql var/lib/
      $ docker cp pmm-data:/var/lib/grafana var/lib/


Now, you can start |pmm-server|:

.. code-block:: bash

   $ docker start pmm-server


Restore
================================================================================

# don't remove old containers
docker stop pmm-server
docker rename pmm-server pmm-server-old
docker rename pmm-data pmm-data-old
 
# Create pmm-data container
docker create \
   -v /opt/prometheus/data \
   -v /opt/consul-data \
   -v /var/lib/mysql \
   -v /var/lib/grafana \
   --name pmm-data \
percona/pmm-server:1.2.2 /bin/true
 
# Restore data
backup_directory="/backup/pmm-data"
docker cp ${backup_directory}/opt/prometheus/data pmm-data:/opt/prometheus/
docker cp ${backup_directory}/opt/consul-data pmm-data:/opt/
docker cp ${backup_directory}/var/lib/mysql pmm-data:/var/lib/
docker cp ${backup_directory}/var/lib/grafana pmm-data:/var/lib/
 
# Fix rights
docker run --rm --volumes-from pmm-data -it percona/pmm-server:1.2.2 chown -R pmm:pmm /opt/prometheus/data /opt/consul-data
docker run --rm --volumes-from pmm-data -it percona/pmm-server:1.2.2 chown -R grafana:grafana /var/lib/grafana
docker run --rm --volumes-from pmm-data -it percona/pmm-server:1.2.2 chown -R mysql:mysql /var/lib/mysql
 
# Create new pmm-server container
docker run -d \
   -p 80:80 \
   --volumes-from pmm-data \
   --name pmm-server \
   --restart always \
   percona/pmm-server:1.2.2
 
sleep 60
pmm-admin check-network
