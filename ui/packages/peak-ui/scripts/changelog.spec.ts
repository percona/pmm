// @vitest-environment node
import { execFileSync } from 'node:child_process';
import { mkdirSync, mkdtempSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';
import {
  gitLogArgs,
  packagesDiffer,
  parseGitLog,
  prependEntry,
  readTarball,
  renderEntry,
  SHIPPED_PATHSPEC,
} from './changelog.mjs';

const tarball = (version: string, files: Record<string, string>) =>
  new Map(
    Object.entries({
      'package/package.json': JSON.stringify({
        name: '@percona/peak-ui',
        version,
        main: 'dist/index.js',
      }),
      ...files,
    }).map(([path, text]) => [path, Buffer.from(text)])
  );
const base = {
  version: '3.12.0',
  date: '2026-12-01',
  previous: '3.11.0',
  firstRelease: false,
};

describe('packagesDiffer', () => {
  it('treats packages that differ only in version and CHANGELOG.md as identical', () => {
    expect(
      packagesDiffer(
        tarball('3.12.0', { 'package/dist/index.js': 'a' }),
        tarball('3.11.0', {
          'package/dist/index.js': 'a',
          'package/CHANGELOG.md': 'old',
        })
      )
    ).toBe(false);
  });
  it('sees a changed bundle, e.g. a bundled dependency bump', () => {
    expect(
      packagesDiffer(
        tarball('3.12.0', { 'package/dist/index.js': 'b' }),
        tarball('3.11.0', { 'package/dist/index.js': 'a' })
      )
    ).toBe(true);
  });
  it('sees an added or removed file', () => {
    expect(
      packagesDiffer(
        tarball('3.12.0', {
          'package/dist/index.js': 'a',
          'package/llms.txt': 'x',
        }),
        tarball('3.11.0', { 'package/dist/index.js': 'a' })
      )
    ).toBe(true);
  });
  it('sees any other package.json change', () => {
    const changedManifest = tarball('3.12.0', {});
    changedManifest.set(
      'package/package.json',
      Buffer.from(
        JSON.stringify({
          name: '@percona/peak-ui',
          version: '3.12.0',
          main: 'dist/other.js',
        })
      )
    );
    expect(packagesDiffer(changedManifest, tarball('3.11.0', {}))).toBe(true);
  });
});

describe('gitLogArgs', () => {
  it('skips commits that reached the previous release through a cherry-pick', () => {
    expect(gitLogArgs('3.11.0')).toEqual([
      'log',
      '--no-merges',
      '--cherry-pick',
      '--right-only',
      '--format=%s%x1f%b%x1e',
      'v3.11.0...HEAD',
      '--',
      ...SHIPPED_PATHSPEC,
    ]);
  });
});

describe('parseGitLog', () => {
  it('reads subjects and BREAKING CHANGE lines', () => {
    const raw =
      'PMM-1 Drop the size prop from Chip (#10)\x1fWhy.\n\nBREAKING CHANGE: Chip no longer accepts `size`.\n\x1e\n' +
      'PMM-2 Fix the table filter row (#11)\x1f\x1e\n';
    expect(parseGitLog(raw)).toEqual([
      {
        subject: 'PMM-1 Drop the size prop from Chip (#10)',
        breaking: ['Chip no longer accepts `size`.'],
      },
      { subject: 'PMM-2 Fix the table filter row (#11)', breaking: [] },
    ]);
  });
});

describe('renderEntry', () => {
  it('says nothing changed for an identical package', () => {
    expect(renderEntry({ ...base, changed: false, commits: [] })).toBe(
      '## 3.12.0 (2026-12-01)\n\nNo changes to Peak UI\n'
    );
  });
  it('says it was rebuilt when only bundled dependencies changed', () => {
    expect(renderEntry({ ...base, changed: true, commits: [] })).toBe(
      '## 3.12.0 (2026-12-01)\n\nRebuilt with updated bundled dependencies; no Peak UI source changes.\n'
    );
  });
  it('lists breaking changes first, then commit titles', () => {
    const commits = [
      {
        subject: 'PMM-1 Drop the size prop from Chip (#10)',
        breaking: ['Chip no longer accepts `size`.'],
      },
      { subject: 'PMM-2 Fix the table filter row (#11)', breaking: [] },
    ];
    expect(renderEntry({ ...base, changed: true, commits })).toBe(
      '## 3.12.0 (2026-12-01)\n\n### Breaking changes\n\n- Chip no longer accepts `size`.\n\n### Changes\n\n- PMM-1 Drop the size prop from Chip (#10)\n- PMM-2 Fix the table filter row (#11)\n'
    );
  });
  it('omits the breaking section when there are none', () => {
    const commits = [
      { subject: 'PMM-2 Fix the table filter row (#11)', breaking: [] },
    ];
    expect(renderEntry({ ...base, changed: true, commits })).not.toContain(
      '### Breaking changes'
    );
  });
  it('writes the first-release line', () => {
    expect(
      renderEntry({
        ...base,
        version: '3.11.0',
        previous: '1.0.32',
        firstRelease: true,
        changed: true,
        commits: [],
      })
    ).toBe(
      '## 3.11.0 (2026-12-01)\n\nFirst release from the PMM monorepo. Continues from 1.0.32.\n'
    );
  });
});

describe('prependEntry', () => {
  it('starts a new file for the first release', () => {
    expect(prependEntry(undefined, '## 3.11.0 (2026-11-20)\n\nFirst.\n')).toBe(
      '# Changelog\n\n## 3.11.0 (2026-11-20)\n\nFirst.\n'
    );
  });
  it('puts the new entry above the older ones', () => {
    const previous = '# Changelog\n\n## 3.11.0 (2026-11-20)\n\nFirst.\n';
    expect(
      prependEntry(
        previous,
        '## 3.11.1 (2026-11-27)\n\nNo changes to Peak UI\n'
      )
    ).toBe(
      '# Changelog\n\n## 3.11.1 (2026-11-27)\n\nNo changes to Peak UI\n\n## 3.11.0 (2026-11-20)\n\nFirst.\n'
    );
  });
});

describe('readTarball', () => {
  it('maps every file to its tarball path and contents', () => {
    const dir = mkdtempSync(join(tmpdir(), 'peak-ui-readtarball-'));
    mkdirSync(join(dir, 'package', 'dist'), { recursive: true });
    writeFileSync(join(dir, 'package', 'package.json'), '{}');
    writeFileSync(join(dir, 'package', 'dist', 'index.js'), 'a');
    const file = join(dir, 'peak-ui.tgz');
    execFileSync('tar', ['-czf', file, '-C', dir, 'package']);

    const files = readTarball(file);

    expect([...files.keys()].sort()).toEqual([
      'package/dist/index.js',
      'package/package.json',
    ]);
    expect(files.get('package/dist/index.js')?.toString()).toBe('a');
  });
});

describe('SHIPPED_PATHSPEC', () => {
  const tracked = execFileSync('git', ['ls-files', '--', ...SHIPPED_PATHSPEC], {
    cwd: fileURLToPath(new URL('..', import.meta.url)),
    encoding: 'utf8',
  })
    .trim()
    .split('\n');

  it('counts the source and the files that ship', () => {
    expect(tracked).toEqual(
      expect.arrayContaining([
        'src/index.ts',
        'src/components/chip/chip.tsx',
        'package.json',
        'tsconfig.json',
        'rollup.config.js',
        'scripts/sourcemap-paths.mjs',
        'AGENTS.md',
        'llms.txt',
        'README.md',
        'LICENSE',
      ])
    );
  });

  it('leaves out tests, stories and their tooling', () => {
    const notShipped =
      /\.(spec|stories)\.|\.mdx$|^vitest\.|^\.storybook\/|^src\/storybook\/|^scripts\/(?!sourcemap-paths\.mjs$)|^(CONTRIBUTING|CLAUDE)\.md$/;
    expect(tracked.filter((file) => notShipped.test(file))).toEqual([]);
  });
});
