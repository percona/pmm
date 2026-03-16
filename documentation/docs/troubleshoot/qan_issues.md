## ClickHouse memory issues in low-memory environments

If you're running PMM Server with less than 16 GB RAM and seeing "memory limit exceeded" errors in ClickHouse logs, switch to the low-memory configuration.

PMM includes two ClickHouse profiles:

- **default** — optimized for performance (16 GB+ RAM)
- **low-memory** — optimized for constrained environments, based on [ClickHouse recommendations](https://clickhouse.com/docs/operations/tips#using-less-than-16gb-of-ram)

### Switch to low-memory configuration

Run from outside the container:
```bash
docker exec -it pmm-server ./switch-config.sh low
```

To switch back:
```bash
docker exec -it pmm-server ./switch-config.sh default
```

The script stops ClickHouse, updates the configuration, and restarts the service.

### Persistent configuration

If you run PMM Server with the `--rm` flag, run the switch script each time the container starts. For systemd, add to your unit file:
```ini
ExecStartPost=/usr/bin/docker exec pmm-server ./switch-config.sh low
```