import { execFileSync } from 'node:child_process';
import { mkdtempSync, readFileSync, readdirSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, relative } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { parseArgs } from 'node:util';
import { previousVersion } from './versions.mjs';

const PACKAGE = '@percona/peak-ui';
const HEADER = '# Changelog\n\n';
const CHANGELOG = 'package/CHANGELOG.md';
const MANIFEST = 'package/package.json';

/** Files that ship, relative to the package folder. */
export const SHIPPED_PATHSPEC = [
  '.',
  ':(exclude,glob)**/*.stories.*',
  ':(exclude,glob)**/*.spec.*',
  ':(exclude,glob)**/*.mdx',
  ':(exclude).storybook',
  ':(exclude)scripts',
  ':(exclude)CONTRIBUTING.md',
  ':(exclude)CLAUDE.md',
];

/**
 * @param {string} previous
 * @returns {string[]}
 */
export const gitLogArgs = (previous) => [
  'log',
  '--no-merges',
  '--cherry-pick',
  '--right-only',
  '--format=%s%x1f%b%x1e',
  `v${previous}...HEAD`,
  '--',
  ...SHIPPED_PATHSPEC,
];

/**
 * @typedef {{ subject: string; breaking: string[] }} Commit
 * @param {string} raw output of `git` run with `gitLogArgs`
 * @returns {Commit[]}
 */
export const parseGitLog = (raw) =>
  raw
    .split('\x1e')
    .map((record) => record.trim())
    .filter(Boolean)
    .map((record) => {
      const [subject, body = ''] = record.split('\x1f');
      const breaking = body.split('\n').flatMap((line) => {
        const match = /^BREAKING CHANGE:\s*(.+)$/.exec(line.trim());
        return match ? [match[1].trim()] : [];
      });
      return { subject: subject.trim(), breaking };
    });

/** @param {Map<string, Buffer>} files */
const comparable = (files) =>
  new Map(
    [...files]
      .filter(([path]) => path !== CHANGELOG)
      .map(([path, content]) => {
        if (path !== MANIFEST) {
          return [path, content];
        }
        const manifest = JSON.parse(content.toString('utf8'));
        delete manifest.version;
        return [path, Buffer.from(JSON.stringify(manifest))];
      })
  );

/**
 * Whether two packed tarballs ship different files, ignoring the version and
 * CHANGELOG.md.
 * @param {Map<string, Buffer>} candidate
 * @param {Map<string, Buffer>} previous
 * @returns {boolean}
 */
export const packagesDiffer = (candidate, previous) => {
  const a = comparable(candidate);
  const b = comparable(previous);
  if (a.size !== b.size) {
    return true;
  }
  return [...a].some(([path, content]) => {
    const other = b.get(path);
    return !other || !content.equals(other);
  });
};

/** @param {string[]} items */
const list = (items) => items.map((item) => `- ${item}`).join('\n');

/**
 * @param {{
 *   version: string;
 *   date: string;
 *   previous: string;
 *   firstRelease: boolean;
 *   changed: boolean;
 *   commits: Commit[];
 * }} entry
 * @returns {string}
 */
export const renderEntry = ({
  version,
  date,
  previous,
  firstRelease,
  changed,
  commits,
}) => {
  const heading = `## ${version} (${date})\n\n`;
  if (firstRelease) {
    return `${heading}First release from the PMM monorepo. Continues from ${previous}.\n`;
  }
  if (!changed) {
    return `${heading}No changes to Peak UI\n`;
  }
  if (commits.length === 0) {
    return `${heading}Rebuilt with updated bundled dependencies; no Peak UI source changes.\n`;
  }
  const breaking = commits.flatMap((commit) => commit.breaking);
  const sections = [];
  if (breaking.length > 0) {
    sections.push(`### Breaking changes\n\n${list(breaking)}`);
  }
  sections.push(
    `### Changes\n\n${list(commits.map((commit) => commit.subject))}`
  );
  return `${heading}${sections.join('\n\n')}\n`;
};

/**
 * @param {string | undefined} previousChangelog
 * @param {string} entry
 * @returns {string}
 */
export const prependEntry = (previousChangelog, entry) => {
  const older = previousChangelog?.startsWith(HEADER)
    ? previousChangelog.slice(HEADER.length)
    : (previousChangelog ?? '');
  return older ? `${HEADER}${entry}\n${older}` : `${HEADER}${entry}`;
};

/**
 * @param {string} file path to a .tgz
 * @returns {Map<string, Buffer>} contents keyed by tarball path (`package/…`)
 */
export const readTarball = (file) => {
  const dir = mkdtempSync(join(tmpdir(), 'peak-ui-tarball-'));
  execFileSync('tar', ['-xzf', file, '-C', dir]);
  return new Map(
    readdirSync(dir, { recursive: true, withFileTypes: true })
      .filter((entry) => entry.isFile())
      .map((entry) => {
        const full = join(entry.parentPath, entry.name);
        return [relative(dir, full), readFileSync(full)];
      })
  );
};

/**
 * @param {string} command
 * @param {string[]} args
 * @param {{ cwd?: string }} [options]
 */
const run = (command, args, options = {}) =>
  execFileSync(command, args, { encoding: 'utf8', ...options });

const main = () => {
  const { values } = parseArgs({
    options: {
      version: { type: 'string' },
      candidate: { type: 'string' },
      out: { type: 'string' },
      date: { type: 'string' },
    },
  });
  const { version, candidate, out } = values;
  if (!version || !candidate || !out) {
    throw new Error(
      'Usage: changelog.mjs --version <v> --candidate <tgz> --out <file> [--date YYYY-MM-DD]'
    );
  }
  const date = values.date ?? new Date().toISOString().slice(0, 10);

  const published = [].concat(
    JSON.parse(run('npm', ['view', PACKAGE, 'versions', '--json']))
  );
  const previous = previousVersion(published, version);
  if (!previous) {
    throw new Error(`No ${PACKAGE} release on npm below ${version}`);
  }

  const downloads = mkdtempSync(join(tmpdir(), 'peak-ui-previous-'));
  const [{ filename }] = JSON.parse(
    run('npm', [
      'pack',
      `${PACKAGE}@${previous}`,
      '--pack-destination',
      downloads,
      '--json',
    ])
  );
  const previousFiles = readTarball(join(downloads, filename));
  const previousChangelog = previousFiles.get(CHANGELOG)?.toString('utf8');
  const firstRelease = previousChangelog === undefined;
  const changed = packagesDiffer(readTarball(candidate), previousFiles);
  const commits =
    changed && !firstRelease
      ? parseGitLog(
          run('git', gitLogArgs(previous), {
            cwd: fileURLToPath(new URL('..', import.meta.url)),
          })
        )
      : [];

  const entry = renderEntry({
    version,
    date,
    previous,
    firstRelease,
    changed,
    commits,
  });
  writeFileSync(out, prependEntry(previousChangelog, entry));
  process.stdout.write(entry);
};

if (
  process.argv[1] &&
  import.meta.url === pathToFileURL(process.argv[1]).href
) {
  main();
}
