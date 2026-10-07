export const Messages = {
  undetermined:
    'The PMM Extensions version could not be determined. This often means an older PMM Extensions image. Run the PMM Server and PMM Extensions images at the same version.',
  mismatch: (serverVersion: string, extensionsVersion: string) =>
    `PMM Server is ${serverVersion} and PMM Extensions is ${extensionsVersion}. Run the PMM Server and PMM Extensions images at the same version.`,
};
