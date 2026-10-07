/**
 * Reduce a version string to its X.Y.Z release for PMM Server / PMM Extensions
 * comparison (see PMM-15671). A leading `v` and any development or pre-release
 * suffix are ignored so `v3.10.0.dev0` and `3.10.0` match.
 *
 * Returns `null` when the string has no release triplet.
 */
export const toReleaseVersion = (version: string): string | null => {
  const match = version.trim().match(/^v?(\d+\.\d+\.\d+)/i);
  return match?.[1] ?? null;
};
