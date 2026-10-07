import { toReleaseVersion } from './toReleaseVersion';

describe('toReleaseVersion', () => {
  it('returns the X.Y.Z release unchanged', () => {
    expect(toReleaseVersion('3.10.0')).toBe('3.10.0');
  });

  it('strips a leading v', () => {
    expect(toReleaseVersion('v3.10.0')).toBe('3.10.0');
    expect(toReleaseVersion('V3.10.0')).toBe('3.10.0');
  });

  it('ignores a development or pre-release suffix', () => {
    expect(toReleaseVersion('3.10.0.dev0')).toBe('3.10.0');
    expect(toReleaseVersion('v3.10.0.dev0')).toBe('3.10.0');
    expect(toReleaseVersion('3.10.0-rc1')).toBe('3.10.0');
    expect(toReleaseVersion('v3.11.0-feature.1')).toBe('3.11.0');
  });

  it('treats prefix- or suffix-only differences as the same release', () => {
    expect(toReleaseVersion('v3.10.0.dev0')).toBe(toReleaseVersion('3.10.0'));
    expect(toReleaseVersion('3.10.0-rc1')).toBe(toReleaseVersion('v3.10.0'));
  });

  it('distinguishes different releases', () => {
    expect(toReleaseVersion('3.10.0')).not.toBe(toReleaseVersion('3.11.0'));
    expect(toReleaseVersion('v3.10.0.dev0')).not.toBe(
      toReleaseVersion('v3.11.0.dev0')
    );
  });

  it('returns null when there is no release triplet', () => {
    expect(toReleaseVersion('')).toBeNull();
    expect(toReleaseVersion('   ')).toBeNull();
    expect(toReleaseVersion('dev')).toBeNull();
    expect(toReleaseVersion('v')).toBeNull();
    expect(toReleaseVersion('3.10')).toBeNull();
  });

  it('trims surrounding whitespace', () => {
    expect(toReleaseVersion('  v3.10.0.dev0  ')).toBe('3.10.0');
  });
});
