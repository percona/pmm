export const Messages = {
  loading: 'Authenticating with the support platform…',
  retry: 'Try again',
  // Shown instead of the page: the exchange failed at load, so there is no work
  // in progress to preserve.
  blocked: {
    signedOutTitle: 'Not signed in',
    signedOut:
      "This page can't be loaded because your PMM session could not be verified. Sign in to PMM again, then retry.",
    unreachableTitle: "This page can't be loaded",
    unreachable:
      "This page can't be loaded because the support platform can't be reached right now. This is usually temporary.",
  },
  // Shown beside a page that is already open. Never replaces it — the user may
  // be part-way through a form.
  notice: {
    signedOut:
      'Your PMM session has expired and ServiceNow can no longer be reached. Sign in to PMM in a new tab, then retry. Your work on this page is saved.',
    unreachable:
      'Lost the connection to the support platform. Anything you submit from this page will fail until it is back. Your work here is kept.',
  },
};
