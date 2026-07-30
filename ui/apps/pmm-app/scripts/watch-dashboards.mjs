#!/usr/bin/env node
// Mirrors dashboard JSON changes from the top-level `dashboards/` folder (reached through the
// `src/dashboards` symlink) into the Grafana-provisioned dashboards directory, so edits show up
// without any manual `make reload-dashboards` step. Grafana's file provisioner polls that
// directory on its own (see build/ansible/roles/grafana/files/dashboards.yml), so a plain file
// copy is enough — no Grafana restart needed.
import { fileURLToPath } from 'node:url';
import path from 'node:path';
import fs from 'node:fs/promises';
import chokidar from 'chokidar';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const SOURCE_DIR = path.resolve(__dirname, '..', 'src', 'dashboards');
const TARGET_DIR =
  process.env.PERCONA_DASHBOARDS_DIST_DIR ?? '/usr/share/percona-dashboards/panels/pmm-app/dist/dashboards';

function targetPathFor(sourcePath) {
  return path.join(TARGET_DIR, path.relative(SOURCE_DIR, sourcePath));
}

async function syncFile(sourcePath) {
  const target = targetPathFor(sourcePath);
  await fs.mkdir(path.dirname(target), { recursive: true });
  await fs.copyFile(sourcePath, target);
}

async function removeFile(sourcePath) {
  const target = targetPathFor(sourcePath);
  await fs.rm(target, { force: true });
}

async function main() {
  await fs.mkdir(TARGET_DIR, { recursive: true }).catch((err) => {
    console.error(
      `[watch-dashboards] Could not create target directory "${TARGET_DIR}": ${err.message}. ` +
        'Set PERCONA_DASHBOARDS_DIST_DIR to a writable path if you are not running inside the devcontainer.'
    );
    process.exit(1);
  });

  const watcher = chokidar.watch(`${SOURCE_DIR}/**/*.json`, { ignoreInitial: false });

  watcher
    .on('add', (file) => syncFile(file).catch((err) => console.error(`[watch-dashboards] sync failed for ${file}:`, err.message)))
    .on('change', (file) => syncFile(file).catch((err) => console.error(`[watch-dashboards] sync failed for ${file}:`, err.message)))
    .on('unlink', (file) => removeFile(file).catch((err) => console.error(`[watch-dashboards] remove failed for ${file}:`, err.message)))
    .on('ready', () => console.log(`[watch-dashboards] watching ${SOURCE_DIR} -> ${TARGET_DIR}`))
    .on('error', (err) => console.error('[watch-dashboards] watcher error:', err.message));
}

main();
