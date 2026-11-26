export const Messages = {
  loading: 'Loading the app…',
  retry: 'Try again',
  loadFailedTitle: "This app can't be loaded",
  loadFailed:
    "The list of apps can't be loaded right now. This is usually temporary.",
  notFoundTitle: 'App not found',
  notFound: 'No app is available at this address.',
  disabledTitle: (label: string) => `${label} is disabled`,
  disabled: 'An administrator has disabled this app.',
  blockedBy: (labels: string[]) =>
    `This app needs ${labels.join(', ')}, which ${labels.length === 1 ? 'is' : 'are'} disabled.`,
};
