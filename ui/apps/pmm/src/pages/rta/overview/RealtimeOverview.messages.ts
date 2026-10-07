export const Messages = {
  allSessions: 'All sessions',
  pause: 'Pause',
  resume: 'Resume',
  refresh: 'Refresh',
  hideCommit: 'Hide BEGIN/COMMIT',
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
    'Show statements waiting for a lock. PMM could not read every kind of lock on this instance, so statements it could not judge are shown as well rather than hidden; the count is of those it could confirm. The session status on the All sessions page, or the agent log, says which lock information is missing.',
  // Every lock source answered, but some connections moved on to a later statement between
  // the statement read and the lock read. A refresh, not a configuration change, clears it.
  blockedUnattributedTooltip: (count: number) =>
    'Show statements waiting for a lock; the count is of those PMM could confirm. ' +
    Messages.blockedUnattributedNote(count),
  blockedUnattributedNote: (count: number) =>
    `${count === 1 ? '1 statement' : `${count} statements`} could not be attributed this refresh: the connection moved on to a later statement between PMM reading the statements and reading the locks. ${count === 1 ? 'It is' : 'They are'} marked Blocked: unknown and shown as well rather than hidden; the next refresh reads them again.`,
  exportTooltip: 'Export to CSV',
  sessionError: (serviceName: string, reason: string) =>
    `Real-Time Analytics could not start for ${serviceName}: ${reason}`,
};
