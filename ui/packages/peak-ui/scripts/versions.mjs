/** @param {string} version */
const parse = (version) => {
  const match = /^(\d+)\.(\d+)\.(\d+)(?:-(.+))?$/.exec(version);
  if (!match) {
    throw new Error(`Not a version: ${version}`);
  }
  return {
    numbers: [Number(match[1]), Number(match[2]), Number(match[3])],
    prerelease: match[4],
  };
};

/**
 * Semver precedence for MAJOR.MINOR.PATCH[-PRERELEASE]; a prerelease sorts
 * below its release.
 * @param {string} a
 * @param {string} b
 * @returns {number}
 */
export const compareVersions = (a, b) => {
  const x = parse(a);
  const y = parse(b);
  for (let i = 0; i < 3; i++) {
    if (x.numbers[i] !== y.numbers[i]) {
      return x.numbers[i] - y.numbers[i];
    }
  }
  if (x.prerelease === y.prerelease) {
    return 0;
  }
  if (x.prerelease === undefined) {
    return 1;
  }
  if (y.prerelease === undefined) {
    return -1;
  }
  return x.prerelease < y.prerelease ? -1 : 1;
};

/**
 * The highest published release below `version`, ignoring prereleases.
 * @param {string[]} published
 * @param {string} version
 * @returns {string | undefined}
 */
export const previousVersion = (published, version) =>
  published
    .filter((v) => parse(v).prerelease === undefined)
    .filter((v) => compareVersions(v, version) < 0)
    .sort(compareVersions)
    .at(-1);

/**
 * `latest` only moves forward; a late patch on an older line gets its own tag.
 * @param {string} version
 * @param {string} latest
 * @returns {string}
 */
export const distTag = (version, latest) => {
  if (compareVersions(version, latest) > 0) {
    return 'latest';
  }
  const [major, minor] = parse(version).numbers;
  return `release-${major}.${minor}`;
};
