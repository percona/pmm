export const Messages = {
  allSessions: 'All sessions',
  pause: 'Pause',
  resume: 'Resume',
  refresh: 'Refresh',
  hideCommit: 'Hide transaction control',
  hideCommitTooltip:
    'Hide transaction-control statements (COMMIT, ROLLBACK, BEGIN, START TRANSACTION) from the list.',
  blockedOnly: (count: number) =>
    count > 0 ? `Blocked only (${count})` : 'Blocked only',
  blockedOnlyTooltip:
    'Show only statements waiting for a row lock. Collection is unaffected; this filters the view.',
  export: 'Export',
};
