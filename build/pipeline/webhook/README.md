# webhook

A small GitHub webhook listener that turns PR comments like
`@pmm-bot build client` into `make` invocations on the host. The listener
parses events, verifies HMAC signatures, dispatches to `make <target>`, and
posts the result back as a PR comment.

Zero third-party dependencies; Go standard library only.

## How it works

1. GitHub delivers `issue_comment.created` events to `POST /webhook`.
2. The listener verifies `X-Hub-Signature-256` with `GITHUB_WEBHOOK_SECRET`.
3. It matches `@<BOT_NAME> <verb> <target...>` in the comment body.
4. It checks `author_association` against `ALLOWED_ASSOCIATIONS`.
   - If `ALLOWED_USERS` is also set, the commenter's login must additionally
     appear in that list. Off-list commenters get a 👎 reaction on their
     comment and the request is dropped.
5. It validates the verb (`build` or `run` — both map to `make`) and lowercases
   + hyphenates the rest into a make target. Examples:
   - `@pmm-bot build client` → `make client`
   - `@pmm-bot build all` → `make all`
   - `@pmm-bot run API tests` → `make api-tests`
   - `@pmm-bot run unit tests` → `make unit-tests`
6. The job goes on a single-worker queue (one build at a time).
7. `make <target>` runs in `MAKE_DIR` (or the listener's cwd if unset) with
   PR context in env vars. Targets may write reply markdown to `$REPLY_FILE`.
8. On completion, the listener posts `$REPLY_FILE` (or a default success/
   failure message) as a PR comment.

## Configuration (environment variables)

| Var | Required | Default | Meaning |
|---|---|---|---|
| `GITHUB_WEBHOOK_SECRET` | yes | — | HMAC secret configured on the GitHub webhook |
| `GITHUB_TOKEN` | yes | — | PAT or App token with `issues:write` and `pull_requests:read` |
| `LISTEN_ADDR` | no | `:7799` | Address to bind |
| `TLS_CERT_FILE` | no | — | PEM cert chain; enables HTTPS when set with `TLS_KEY_FILE` |
| `TLS_KEY_FILE` | no | — | PEM private key paired with `TLS_CERT_FILE` |
| `BOT_NAME` | no | `pmm-bot` | Mention handle (without the `@`) |
| `MAKE_DIR` | no | listener cwd | Directory containing the Makefile (`cmd.Dir` for `make`) |
| `WORK_DIR` | no | `/var/lib/webhook` | Per-job scratch (run.log, reply.md) |
| `ALLOWED_ASSOCIATIONS` | no | `MEMBER,OWNER,COLLABORATOR` | Allowed `author_association` values |
| `ALLOWED_USERS` | no | — | Comma-separated GitHub logins (case-insensitive). When set, only these users can trigger builds; others get a 👎 reaction |
| `JOB_TIMEOUT` | no | `2h` | Go duration; the make process is killed after this |

## Env vars passed to the make process

Make targets read these as `$(PR_HEAD_SHA)` etc. (or `$$PR_HEAD_SHA` inside
recipe lines).

| Var | Example |
|---|---|
| `PR_NUMBER` | `4320` |
| `PR_HEAD_SHA` | `f5ea8c2a…` |
| `PR_HEAD_SHORT_SHA` | `f5ea8c2` |
| `PR_HEAD_REF` | `my-feature-branch` |
| `PR_HEAD_REPO` | `alice/pmm` (fork PRs) or `percona/pmm` |
| `PR_HEAD_CLONE_URL` | `https://github.com/alice/pmm.git` |
| `PR_BASE_REF` | `v3` |
| `PR_REPO_OWNER` | `percona` |
| `PR_REPO_NAME` | `pmm` |
| `PR_TAG` | `PR-4320-f5ea8c2` |
| `COMMENTER` | `alice` |
| `COMMAND` | raw command string (`build client`) |
| `REPLY_FILE` | optional: write markdown here to override the default reply |
| `JOB_DIR` | per-job scratch dir (stdout/stderr are tee'd to `JOB_DIR/run.log`) |
| `LOG_UUID` | the GitHub delivery UUID — same value used by `GET /logs/<uuid>/run.log` and the `pmm-log-viewer` container |

## Writing make targets

Targets follow the normal Makefile pattern. The recipe receives PR context
through the environment, so it can check out the right commit, tag images, etc.

```make
.PHONY: client
client:
	git fetch origin $(PR_HEAD_SHA)
	git checkout $(PR_HEAD_SHA)
	docker build -t perconalab/pmm-client-fb:$(PR_TAG) -f Dockerfile.client .
	docker push perconalab/pmm-client-fb:$(PR_TAG)
	echo "Built **\`perconalab/pmm-client-fb:$(PR_TAG)\`**" > $(REPLY_FILE)
```

Then `@pmm-bot build client` triggers `make client`.

## Running locally

```sh
make build
GITHUB_WEBHOOK_SECRET=... GITHUB_TOKEN=ghp_... \
MAKE_DIR=/path/to/your/Makefile/dir \
WORK_DIR=$(pwd)/work \
./webhook
```

Expose the listener with a tunnel (e.g. `cloudflared tunnel` or `ngrok`) and
point a GitHub webhook at `https://<tunnel>/webhook`:

- **Payload URL:** `https://<host>/webhook`
- **Content type:** `application/json`
- **Secret:** the value of `GITHUB_WEBHOOK_SECRET`
- **Events:** just *Issue comments*

## Security notes

- Only `author_association` is checked. Fork contributors will be `NONE`
  unless already added as collaborators. For stronger authorization, switch
  to a GitHub App with team-based checks.
- `make` runs with the listener's UID. Don't give it root. Scope its token,
  S3 credentials, and registry login to least privilege.
- The PR commit the make target operates on is attacker-controlled. Treat
  that code as untrusted and run the build inside an isolated runner.
- Only the verbs `build` and `run` are accepted; targets must match
  `^[a-z0-9][a-z0-9._-]*$` after lowercase + hyphenation. This blocks shell
  metachars and path traversal but does not gate which Makefile targets
  exist — keep destructive targets (e.g. `clean`, `prune`) out of the
  Makefile served to the webhook, or front-gate them inside the recipe.

## Tests

```sh
make test
```
