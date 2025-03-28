# How to add Victorialogs to PMM Server image

1. Clone this repository and checkout `PMM-13391-integrate-victorialogs` branch:

```bash
git clone https://github.com/percona/pmm.git
cd pmm
git fetch
git checkout PMM-13391-integrate-victorialogs
```

2. Build the image with the following command (mind the dot!):

```bash
docker buildx build --platform=linux/amd64 --progress=plain -t local/pmm-server:victorialogs-1.17.0 -f ./build/ansible/roles/victorialogs/files/Dockerfile.victorialogs .
```

3. Launch PMM server:

```bash
docker run -d --name pmm-server -p 443:8443 local/pmm-server:victorialogs-1.17.0
```

4. Access PMM server at https://localhost:443.

5. Login with default credentials (admin/admin).

6. You can now query the logs by navigating to the "Explore" tab in the PMM server interface. Choose the VictoriaLogs datasource and enjoy!
