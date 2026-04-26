# pmm-log-viewer

Throwaway Docker image for viewing a PMM build log in a browser. Takes a
`LOG_UUID` from the bot's PR comment, downloads the log once from the webhook
host, and renders it through [Monaco Editor](https://microsoft.github.io/monaco-editor/)
(VS Code's engine) in read-only mode.

## How it works

1. Container starts with `LOG_UUID` and `WEBHOOK_URL` in env.
2. The entrypoint `curl`s `${WEBHOOK_URL}/logs/${LOG_UUID}` to
   `/usr/share/nginx/html/log.txt`.
3. Nginx serves:
   - `/` → tiny `index.html` that boots Monaco
   - `/log.txt` → the downloaded log
   - `/monaco/...` → Monaco's prebuilt AMD bundle, baked into the image
4. The browser loads Monaco, fetches `/log.txt`, and shows the editor in
   read-only mode with VS Code keybindings (`Ctrl-F`, `Ctrl-G`, etc.).

If the download fails, the container still starts; the editor shows a
diagnostic message in place of the log.

## Build

```sh
docker build -t pmm-log-viewer:dev build/pipeline/log-viewer
```

Pin a Monaco version (default is whatever the Dockerfile's `ARG MONACO_VERSION`
points at — bump it as you like):

```sh
docker build --build-arg MONACO_VERSION=0.52.2 -t pmm-log-viewer:dev build/pipeline/log-viewer
```

## View a build's log

Copy the UUID from the bot's PR comment, then:

```sh
docker run --rm -it -p 8080:8080 \
    -e LOG_UUID=<uuid-from-pr-comment> \
    -e WEBHOOK_URL=https://builds.example.com \
    pmm-log-viewer:dev
```

Open <http://localhost:8080>. Stop with Ctrl-C.

### Env vars

| Var | Required | Default | Meaning |
|---|---|---|---|
| `LOG_UUID` | yes | — | Build UUID from the PR comment |
| `WEBHOOK_URL` | yes | — | Base URL of the webhook host (e.g. `https://builds.example.com`) |
| `CURL_OPTS` | no | — | Extra flags for `curl`. Useful for self-signed TLS (`-k`) or DNS pinning (`--resolve …`). |

### Self-signed certs

```sh
docker run --rm -p 8080:8080 \
    -e LOG_UUID=... -e WEBHOOK_URL=https://builds.lan \
    -e CURL_OPTS='-k' \
    pmm-log-viewer:dev
```

## Notes

- The log is downloaded once at container start. Restart the container to
  pick up updates from a still-running build.
- The Monaco bundle is ~6 MB; final image is ~50 MB.
- Workers, syntax highlighting, and minimap are intentionally disabled —
  build logs are plaintext, the minimap adds noise, and disabling the worker
  drops the editor's idle CPU to zero.
