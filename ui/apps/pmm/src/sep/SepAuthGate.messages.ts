export const Messages = {
  loading: 'Authenticating with Smart Expert Platform…',
  retry: 'Try again',
  // Shown instead of the page: the exchange failed at load, so there is no work
  // in progress to preserve.
  blocked: {
    signedOutTitle: 'Not signed in',
    signedOut:
      'Smart Expert Platform could not verify your PMM session. Sign in to PMM again, then retry.',
    unreachableTitle: 'Could not reach Smart Expert Platform',
    unreachable:
      'Authenticating with Smart Expert Platform failed. This is usually temporary.',
  },
  // Shown beside a page that is already open. Never replaces it — the user may
  // be part-way through a form.
  notice: {
    signedOut:
      'Your PMM session has ended, so Smart Expert Platform can no longer be reached. Anything you submit from this page will fail. Sign in to PMM in another tab, then retry — your work here is kept.',
    unreachable:
      'Lost the connection to Smart Expert Platform. Anything you submit from this page will fail until it is back. Your work here is kept.',
  },
};
