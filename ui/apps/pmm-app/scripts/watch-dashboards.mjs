#!/usr/bin/env node
// Mirrors dashboard JSON changes from the top-level `dashboards/` folder (reached through the
// `src/dashboards` symlink) into the directory Grafana provisions dashboards from, so edits show
// up without any manual reload step. Grafana's file provisioner polls that directory on its own
// (every 5s in the devcontainer, see dev/grafana-dashboards-provisioning.yml), so a plain file
// copy is enough — no Grafana restart needed.
//
// Outside the devcontainer the target directory usually does not exist; the watcher then logs a
// hint and exits cleanly so `pnpm dev` keeps running the other dev servers.
import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import chokidar from 'chokidar';

const LOG_PREFIX = '[watch-dashboards]';
const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const TARGET_DIR =
  process.env.PERCONA_DASHBOARDS_DIST_DIR ??
  '/usr/share/percona-dashboards/panels/pmm-app/dist/dashboards';

async function main() {
  // Watch the real directory so every event path is relative to the same root.
  const sourceDir = await fs.realpath(
    path.resolve(SCRIPT_DIR, '..', 'src', 'dashboards')
  );
  const targetFor = (file) =>
    path.join(TARGET_DIR, path.relative(sourceDir, file));

  try {
    await fs.mkdir(TARGET_DIR, { recursive: true });
    await fs.access(TARGET_DIR, fs.constants.W_OK);
  } catch (err) {
    console.log(
      `${LOG_PREFIX} dashboard sync disabled: ${TARGET_DIR} is not writable (${err.code}). ` +
        'Set PERCONA_DASHBOARDS_DIST_DIR to enable it outside the devcontainer.'
    );
    return;
  }

  const sync = async (file) => {
    const target = targetFor(file);
    await fs.mkdir(path.dirname(target), { recursive: true });
    await fs.copyFile(file, target);
  };
  const remove = (file) => fs.rm(targetFor(file), { force: true });
  const handle = (action, verb) => (file) => {
    if (!file.endsWith('.json')) {
      return;
    }
    action(file).catch((err) =>
      console.error(`${LOG_PREFIX} ${verb} failed for ${file}: ${err.message}`)
    );
  };

  chokidar
    .watch(sourceDir, {
      ignoreInitial: false,
      // Tooling, not dashboards.
      ignored: path.join(sourceDir, 'misc'),
    })
    .on('add', handle(sync, 'sync'))
    .on('change', handle(sync, 'sync'))
    .on('unlink', handle(remove, 'remove'))
    .on('ready', () =>
      console.log(`${LOG_PREFIX} watching ${sourceDir} -> ${TARGET_DIR}`)
    )
    .on('error', (err) =>
      console.error(`${LOG_PREFIX} watcher error: ${err.message}`)
    );
}

main().catch((err) => {
  console.error(`${LOG_PREFIX} ${err.message}`);
  process.exit(1);
});
