#!/bin/bash -ex

podman volume create pmm-data

podman network create pmm_default

# Allow non-root users to bind to privileged ports (required for port 443)        
# Make the setting persistent
echo "net.ipv4.ip_unprivileged_port_start=443" | sudo tee /etc/sysctl.d/99-pmm.conf
sudo sysctl -p /etc/sysctl.d/99-pmm.conf

systemctl --user enable --now podman.socket

mkdir -p ~/.config/systemd/user/

cat > ~/.config/systemd/user/pmm-server.service <<EOF
[Unit]
Description=pmm-server
Wants=network-online.target
After=network-online.target
After=nss-user-lookup.target nss-lookup.target
After=time-sync.target
[Service]
EnvironmentFile=%h/.config/systemd/user/pmm-server.env
Environment=PMM_VOLUME_NAME=pmm-data
TimeoutStartSec=480
Restart=on-failure
RestartSec=20
ExecStart=/usr/bin/podman run \
    --volume=${PMM_VOLUME_NAME}:/srv \
    --rm --replace=true --name %N \
    --env-file=%h/.config/systemd/user/pmm-server.env \
    --net pmm_default \
    --cap-add=net_admin,net_raw \
    --userns=keep-id:uid=1000,gid=1000 \
    -p 443:8443/tcp --ulimit=host ${PMM_IMAGE}
ExecStop=/usr/bin/podman stop -t 10 %N
[Install]
WantedBy=default.target
EOF

echo "PMM_IMAGE=docker.io/percona/pmm-server:3" > ~/.config/systemd/user/pmm-server.env

systemctl --user enable --now pmm-server

timeout 60 podman wait --condition=running pmm-server

podman ps --all
