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
TimeoutStartSec=480
Restart=on-failure
RestartSec=20
ExecStart=/usr/bin/podman run \
    --rm --replace=true --name %N \
    --volume=pmm-data:/srv \
    --env-file=%h/.config/systemd/user/pmm-server.env \
    --network=pmm_default \
    --cap-add=net_admin,net_raw \
    -p 443:8443/tcp --ulimit=host \
    docker.io/percona/pmm-server:3
ExecStop=/usr/bin/podman stop -t 10 %N
[Install]
WantedBy=default.target
EOF

cat > ~/.config/systemd/user/pmm-server.env <<EOF
GF_SECURITY_ADMIN_PASSWORD=strong-password
PMM_ENABLE_ACCESS_CCONTROL=1
EOF

systemctl --user enable --now pmm-server

# Give it some time to download the image and start
sleep 60

podman wait --condition=running pmm-server

if [ "$(curl -sk -o /dev/null -w "%{http_code}" https://127.0.0.1:443/v1/server/readyz)" -ne 200 ]; then
  echo "pmm-server container is NOT ready"
  exit 1
fi

echo "🍏 pmm-server container is running and accepting connections 🍏"
