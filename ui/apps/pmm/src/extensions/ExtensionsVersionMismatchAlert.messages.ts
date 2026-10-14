export const Messages = {
  undetermined:
    'PMM Extensions did not report a usable release version. Run the PMM Server and PMM Extensions images at the same version.',
  mismatch: (serverVersion: string, extensionsVersion: string) =>
    `PMM Server is ${serverVersion} and PMM Extensions is ${extensionsVersion}. Run the PMM Server and PMM Extensions images at the same version.`,
};
