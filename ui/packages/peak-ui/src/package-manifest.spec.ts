import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';
import * as design from './design';

const root = process.cwd();
const read = (file: string) => readFileSync(resolve(root, file), 'utf8');
const pkg = JSON.parse(read('package.json')) as {
  name: string;
  files: string[];
  scripts: Record<string, string>;
  peerDependencies: Record<string, string>;
};

// PMM-15491: the consumer-facing AGENTS.md must ship in the npm tarball.
describe('published package contents', () => {
  it('lists AGENTS.md in package.json files', () => {
    expect(pkg.files).toContain('AGENTS.md');
  });

  it('has a non-empty AGENTS.md at the package root', () => {
    expect(existsSync(resolve(root, 'AGENTS.md'))).toBe(true);
    expect(read('AGENTS.md')).toMatch(/^# Peak UI/);
  });
});

// Drift guards: hand-written guides must not contradict package.json or the exports.
describe('guides stay in sync with the package', () => {
  it('README install command matches peerDependencies', () => {
    const block = read('README.md').match(/```bash\n(pnpm add [\s\S]*?)```/)?.[1];
    expect(block, 'README has a `pnpm add` install block').toBeDefined();
    const listed = block!
      .replace(/\\\n/g, ' ')
      .split(/\s+/)
      .slice(2)
      .filter((p) => p && p !== pkg.name);
    expect(listed.sort()).toEqual(Object.keys(pkg.peerDependencies).sort());
  });

  it('AGENTS.md only names theme options that are exported', () => {
    const names = [...read('AGENTS.md').matchAll(/`([a-z]+ThemeOptions)`/g)].map((m) => m[1]);
    expect(names.length).toBeGreaterThan(0);
    names.forEach((name) => expect(design, name).toHaveProperty(name));
  });

  it('CONTRIBUTING.md only mentions pnpm scripts that exist', () => {
    const builtins = ['install', 'add'];
    const mentioned = [...read('CONTRIBUTING.md').matchAll(/`pnpm ([a-z:]+)/g)]
      .map((m) => m[1])
      .filter((s) => !builtins.includes(s));
    expect(mentioned.length).toBeGreaterThan(0);
    mentioned.forEach((script) => expect(pkg.scripts, script).toHaveProperty(script));
  });
});
