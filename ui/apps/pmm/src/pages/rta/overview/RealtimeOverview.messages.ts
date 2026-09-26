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
    'Show only statements waiting for a lock, of either kind. Collection is unaffected; this filters the view.',
  // Distinct from "(0)": nothing is known about waiting, so no claim is made either way.
  blockedUnknown: 'Blocked unknown',
  blockedUnknownTooltip:
    'PMM could not read the lock information from this instance, so it cannot tell which statements are waiting. Check that the monitoring user can read performance_schema; the agent log says why.',
  // Shown when only one of the two lock sources answered. The filter still works, but it
  // keeps the undecided rows rather than dropping statements that may well be waiting.
  blockedPartialTooltip:
    'Show statements waiting for a lock. PMM could not read every kind of lock on this instance, so statements it could not judge are shown as well rather than hidden; the count is of those it could confirm. The agent log says which lock information is missing.',
  export: 'Export',
};
