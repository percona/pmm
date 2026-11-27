// @vitest-environment node
import { describe, expect, it } from 'vitest';
import { compareVersions, distTag, previousVersion } from './versions.mjs';

const published = ['1.0.31', '1.0.32', '3.10.0', '3.11.0', '3.11.1'];

describe('previousVersion', () => {
  it('takes the highest version below a new release', () => {
    expect(previousVersion(published, '3.12.0')).toBe('3.11.1');
  });
  it('stays on its own line for a late patch on an older line', () => {
    expect(previousVersion(published, '3.10.1')).toBe('3.10.0');
  });
  it('finds 1.0.32 for the first release from PMM', () => {
    expect(previousVersion(['1.0.31', '1.0.32'], '3.11.0')).toBe('1.0.32');
  });
  it('places a dry-run prerelease above everything published below it', () => {
    expect(previousVersion(published, '3.99.0-dryrun')).toBe('3.11.1');
  });
  it('skips published prereleases', () => {
    expect(previousVersion(['3.11.0', '3.12.0-rc.1'], '3.12.0')).toBe('3.11.0');
  });
  it('returns undefined when nothing is lower', () => {
    expect(previousVersion(published, '1.0.0')).toBeUndefined();
  });
});

describe('compareVersions', () => {
  it('compares numerically, not as strings', () => {
    expect(compareVersions('3.10.0', '3.9.9')).toBeGreaterThan(0);
  });
  it('sorts a prerelease below its release', () => {
    expect(compareVersions('3.12.0-dryrun', '3.12.0')).toBeLessThan(0);
  });
});

describe('distTag', () => {
  it('moves latest forward', () => {
    expect(distTag('3.12.0', '3.11.1')).toBe('latest');
    expect(distTag('3.11.0', '1.0.32')).toBe('latest');
  });
  it('never moves latest backwards for a late patch', () => {
    expect(distTag('3.10.1', '3.11.1')).toBe('release-3.10');
  });
});
