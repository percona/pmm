export const Messages = {
  title: (version?: string) =>
    version ? `PMM Server was updated to ${version}` : 'PMM Server was updated',
  description:
    ': this page is still running the previous version. Reload to finish.',
  reload: 'Reload now',
  dismiss: 'Not now',
};
