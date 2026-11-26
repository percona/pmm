import { existsSync, readdirSync, readFileSync, realpathSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';
import * as peakUi from './index';

const root = process.cwd();
const read = (file: string) => readFileSync(resolve(root, file), 'utf8');
const pkg = JSON.parse(read('package.json')) as {
  name: string;
  files: string[];
  scripts: Record<string, string>;
  peerDependencies: Record<string, string>;
  publishConfig: { main: string; types: string };
  license: string;
  dependencies: Record<string, string>;
  devDependencies: Record<string, string>;
};
const publicExports = Object.keys(peakUi);

// PMM-15491: the consumer-facing AGENTS.md must ship in the npm tarball.
describe('published package contents', () => {
  // The changelog compares packages, so a story or spec edit must not change one.
  it("ships PMM's license, matching the license field", () => {
    expect(read('LICENSE')).toBe(read('../../../LICENSE'));
    expect(read('LICENSE')).toMatch(
      /GNU AFFERO GENERAL PUBLIC LICENSE\s+Version 3/
    );
    expect(pkg.license).toBe('AGPL-3.0-or-later');
  });

  it('leaves licensing to LICENSE: no source file carries a header', () => {
    const withHeader = readdirSync(resolve(root, 'src'), { recursive: true })
      .map(String)
      .filter((file) => /\.(tsx?|mdx)$/.test(file))
      .filter((file) => file !== 'package-manifest.spec.ts')
      .filter((file) =>
        /Copyright \(C\)|Licensed under the/.test(read(`src/${file}`))
      );
    expect(withHeader).toEqual([]);
  });

  // turbo's build is PMM's image build (make -C ui release); consumers read src.
  it('builds the bundle only when packing, not in turbo build', () => {
    expect(pkg.scripts.build).toBeUndefined();
    expect(pkg.scripts.prepack).toBe('pnpm build:package');
  });

  it('ships only the entry points and the guides', () => {
    const { main, types } = pkg.publishConfig;
    const allowed = [main, `${main}.map`, types, 'AGENTS.md', 'llms.txt'];
    expect([...pkg.files].sort()).toEqual([...allowed, 'CHANGELOG.md'].sort());
  });

  it('lists AGENTS.md in package.json files', () => {
    expect(pkg.files).toContain('AGENTS.md');
  });

  it('has a non-empty AGENTS.md at the package root', () => {
    expect(existsSync(resolve(root, 'AGENTS.md'))).toBe(true);
    expect(read('AGENTS.md')).toMatch(/^# Peak UI/);
  });
});

// A second copy of Peak UI or of a peer (React, Emotion, MUI, date pickers, forms,
// notistack) means a second React context: an unthemed UI or a picker without its
// LocalizationProvider.
describe('workspace consumers', () => {
  const ui = resolve(root, '../..');
  const packages = [
    ...read('../../pnpm-workspace.yaml').matchAll(/^ {2}- '([^']+)\/\*'$/gm),
  ]
    .flatMap(([, dir]) =>
      readdirSync(resolve(ui, dir), { withFileTypes: true })
        .filter((entry) => entry.isDirectory())
        .map((entry) => `${dir}/${entry.name}`)
    )
    .filter((dir) => existsSync(resolve(ui, dir, 'package.json')))
    .map((dir) => ({
      dir,
      manifest: JSON.parse(
        readFileSync(resolve(ui, dir, 'package.json'), 'utf8')
      ) as Record<string, Record<string, string> | undefined>,
    }));
  const ranges = (manifest: (typeof packages)[number]['manifest']) =>
    ['dependencies', 'devDependencies', 'peerDependencies'].flatMap((field) => {
      const range = manifest[field]?.[pkg.name];
      return range ? [{ field, range }] : [];
    });
  const consumers = packages.filter(({ manifest }) => ranges(manifest).length);

  it('every PMM package uses this package from the workspace', () => {
    expect(consumers.length).toBeGreaterThan(0);
    const pinned = consumers.flatMap(({ dir, manifest }) =>
      ranges(manifest)
        .filter(({ range }) => range !== 'workspace:*')
        .map(({ field, range }) => `${dir} ${field}: ${range}`)
    );
    expect(pinned).toEqual([]);
  });

  it('accepts the React version PMM UI runs, and nothing older', () => {
    const app = JSON.parse(read('../../apps/pmm/package.json')) as {
      dependencies: Record<string, string>;
    };
    ['react', 'react-dom'].forEach((name) =>
      expect(pkg.peerDependencies[name], name).toBe(app.dependencies[name])
    );
  });

  it('consumers resolve the same copy of every peer dependency', () => {
    const installed = (dir: string, name: string) => {
      const path = resolve(ui, dir, 'node_modules', name);
      return existsSync(path) ? realpathSync(path) : undefined;
    };
    const split = Object.keys(pkg.peerDependencies).flatMap((name) => {
      const ours = installed('packages/peak-ui', name);
      return consumers.flatMap(({ dir }) => {
        const theirs = installed(dir, name);
        return ours && theirs && theirs !== ours ? [`${dir}: ${name}`] : [];
      });
    });
    expect(split).toEqual([]);
  });
});

// PMM-15878: MUI X Pro and Premium packages are commercial and need a license key.
describe('licensing', () => {
  it('lists no MUI X Pro or Premium package', () => {
    const listed = Object.keys({
      ...pkg.dependencies,
      ...pkg.devDependencies,
      ...pkg.peerDependencies,
    });
    expect(
      listed.filter((name) => /^@mui\/x-.*-(pro|premium)$/.test(name))
    ).toEqual([]);
  });
});

// Drift guards: hand-written guides must not contradict package.json or the public entry point.
describe('guides stay in sync with the package', () => {
  it('README install command matches peerDependencies', () => {
    const block = read('README.md').match(
      /```bash\n(pnpm add [\s\S]*?)```/
    )?.[1];
    expect(block, 'README has a `pnpm add` install block').toBeDefined();
    const listed = block!
      .replace(/\\\n/g, ' ')
      .split(/\s+/)
      .slice(2)
      .filter((p) => p && p !== pkg.name);
    expect(listed.sort()).toEqual(Object.keys(pkg.peerDependencies).sort());
  });

  it('AGENTS.md only names theme options that are exported', () => {
    const names = [
      ...read('AGENTS.md').matchAll(/`([a-z]+ThemeOptions)`/g),
    ].map((m) => m[1]);
    expect(names.length).toBeGreaterThan(0);
    names.forEach((name) => expect(publicExports, name).toContain(name));
  });

  it('AGENTS.md and README examples only import names the package exports', () => {
    const names = [
      ...(read('AGENTS.md') + read('README.md')).matchAll(
        /import \{([^}]+)\} from '@percona\/peak-ui'/g
      ),
    ]
      .flatMap((m) => m[1].split(','))
      .map(
        (n) =>
          n
            .trim()
            .replace(/^type\s+/, '')
            .split(/\s+as\s+/)[0]
      )
      .filter(Boolean);
    expect(names.length).toBeGreaterThan(0);
    names.forEach((name) => expect(publicExports, name).toContain(name));
  });

  it('CONTRIBUTING.md only mentions pnpm scripts that exist', () => {
    const builtins = [
      'install',
      'add',
      'dlx',
      'exec',
      'run',
      'pack',
      'publish',
    ];
    const mentioned = [
      ...read('CONTRIBUTING.md').matchAll(/\bpnpm ([a-z][a-zA-Z0-9:_-]*)/g),
    ]
      .map((m) => m[1])
      .filter((s) => !builtins.includes(s));
    expect(mentioned.length).toBeGreaterThan(0);
    mentioned.forEach((script) =>
      expect(Object.keys(pkg.scripts), script).toContain(script)
    );
  });
});
