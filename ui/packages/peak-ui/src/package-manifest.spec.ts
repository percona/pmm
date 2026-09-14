import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

const root = process.cwd();
const pkg = JSON.parse(readFileSync(resolve(root, 'package.json'), 'utf8')) as {
  files: string[];
};

// PMM-15491: the consumer-facing AGENTS.md must ship in the npm tarball.
describe('published package contents', () => {
  it('lists AGENTS.md in package.json files', () => {
    expect(pkg.files).toContain('AGENTS.md');
  });

  it('has a non-empty AGENTS.md at the package root', () => {
    const path = resolve(root, 'AGENTS.md');
    expect(existsSync(path)).toBe(true);
    expect(readFileSync(path, 'utf8')).toMatch(/^# Peak UI/);
  });
});
