# Mongolog

PMM supports collecting MongoDB query metrics from **slow query logs** instead of the profiler by using the `mongolog` query source.

`mongolog` parses MongoDB’s slow query logs from disk in real time. This method **does not rely on the `system.profile` collection** and **does not require enabling the MongoDB profiler**.

---

## ✅ When to Use `mongolog`

Use `mongolog` if:

- You want to avoid the performance overhead of MongoDB's built-in profiler.
- You are using `mongos` or a managed environment where `system.profile` is unavailable or restricted.
- You prefer log-based durability.

---
## ⚙️ MongoDB Configuration

To use `mongolog`, MongoDB must log slow operations to a file. You can configure this using either a config file or command-line flags.

### 🔧 Option 1: Config File (`mongod.conf`)

Use this configuration to enable slow query logging to a file:

```yaml
systemLog:
  destination: file
  path: /var/log/mongodb/mongod.log
  logAppend: true

operationProfiling:
  mode: slowOp
  slowOpThresholdMs: 100
```

✅ This logs slow operations to a file, appends instead of overwriting, and sets the threshold to 100ms.  
🔐 Make sure the log file is readable by the user running the PMM agent.

---

### 🔧 Option 2: Command-Line Flags

Alternatively, start `mongod` with these flags:

```bash
mongod \
  --dbpath /var/lib/mongo \
  --logpath /var/log/mongodb/mongod.log \
  --logappend \
  --profile 1 \
  --slowms 100
```

#### 🧾 Flag Reference

| Flag           | Purpose                                               |
|----------------|--------------------------------------------------------|
| `--logpath`    | Enables logging to a file (required by mongolog)      |
| `--logappend`  | Appends to the log file instead of overwriting        |
| `--profile 1`  | Enables logging of slow operations                     |
| `--slowms 100` | Sets slow operation threshold (in milliseconds)       |
| `--dbpath`     | Required if no config file is used                    |

🛠️ These flags can be adapted to your deployment automation (Docker, systemd, etc).

---

## 🧩 Adding MongoDB with Mongolog to PMM

Use the following `pmm-admin` command to register the MongoDB instance with `mongolog` as the query source:

```bash
pmm-admin add mongodb \
  --query-source=mongolog \
  --username=pmm \
  --password=your_secure_password \
  127.0.0.1
```

**Required options:**

- `--query-source=mongolog`: Enables log-based query analytics  
- `--username`, `--password`: Must match MongoDB credentials  
- MongoDB must be accessible to the PMM agent

---

## 🔁 Log Rotation

To ensure `mongolog` continues reading logs after rotation:

- Use `copytruncate` in your `logrotate` config
- Avoid deleting or renaming log files in-place
- Do not rotate logs by moving the file — `mongolog` tails by path

Example `logrotate` config:
```txt
/var/log/mongodb/mongod.log {
    daily
    rotate 7
    compress
    delaycompress
    copytruncate
    missingok
    notifempty
}
```

---

## 📊 Visibility in PMM

Once added, slow query metrics from `mongolog` will appear in **Query Analytics (QAN)** just like with the profiler source.


---

## 🧠 Comparison: `profiler` vs `mongolog`

| Feature                     | `profiler` | `mongolog`     |
|----------------------------|-------------|------------------|
| Requires `system.profile`  | ✅ Yes      | ❌ No            |
| Supports `mongos`          | ❌ No       | ✅ Yes           |
| Adds DB overhead           | ✅ Higher   | ✅ Low           |
| Uses file-based logging    | ❌ No       | ✅ Yes           |
| Durable query history      | ❌ Volatile | ✅ Disk-backed   |

---

## 🧪 Notes

- `--profile 1` or `operationProfiling.mode: slowOp` is sufficient — no need for full profiler mode.
- Metrics appear in QAN regardless of whether query source is choosen.
- Ideal for production workloads where profiler is too heavy or not available.
