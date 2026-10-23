// PSMDB builds read 7.0.24-13: the upstream release, then Percona's build; upstream MongoDB reads 7.0.15.
const VERSION = /^(\d+)\.(\d+)\.(\d+)(?:-(\d+))?/;

const parts = (version: string) => {
  const match = VERSION.exec(version);

  // A release without a build sorts below its Percona builds.
  return match
    ? [match[1], match[2], match[3], match[4] ?? '-1'].map(Number)
    : null;
};

/** Orders by release, then by build; a string that is no version falls back to text order. */
export const compareVersions = (a: string, b: string): number => {
  const left = parts(a);
  const right = parts(b);

  if (!left || !right) {
    return a.localeCompare(b);
  }
  const index = left.findIndex((value, at) => value !== right[at]);

  return index === -1 ? 0 : left[index] - right[index];
};

/** The version a group of members is still exposed on. */
export const lowestVersion = (versions: string[]): string | undefined =>
  [...versions].sort(compareVersions)[0];

export interface VersionGroup {
  version: string;
  /** Service names, alphabetically. */
  members: string[];
}

/** Members by the version they run, lowest version first. */
export const groupByVersion = (
  members: Array<{ name: string; version?: string }>
): VersionGroup[] => {
  const groups = new Map<string, string[]>();

  members.forEach(({ name, version }) => {
    if (version) {
      groups.set(version, [...(groups.get(version) ?? []), name]);
    }
  });

  return [...groups.entries()]
    .map(([version, names]) => ({ version, members: [...names].sort() }))
    .sort((a, b) => compareVersions(a.version, b.version));
};
