export const Messages = {
  allSessions: 'All sessions',
  pause: 'Pause',
  resume: 'Resume',
  refresh: 'Refresh',
  hideCommit: 'Hide transaction control',
  hideCommitTooltip:
    'Hide BEGIN, START TRANSACTION, COMMIT and ROLLBACK statements from the list',
  blockedOnly: (count: number) =>
    count > 0 ? `Blocked only (${count})` : 'Blocked only',
  blockedOnlyTooltip:
    "Show only statements that wait for a row or metadata lock. This filters the view and doesn't change what's collected.",
  // Distinct from "(0)": nothing is known about waiting, so no claim is made either way.
  blockedUnknown: 'Blocked unknown',
  blockedUnknownTooltip:
    "This instance doesn't report all lock information, so the list can't show which statements are waiting. To find out why, check the session status in All sessions or the agent log.",
  blockedPartialTooltip:
    "Show statements waiting for a lock. This instance doesn't report every lock type, so the list also shows statements that might be waiting, and the count includes only confirmed waits. To find out why, check All sessions or the agent log.",
  blockedUnattributedTooltip: (count: number) =>
    'Show statements waiting for a lock. The count includes only confirmed waits. ' +
    Messages.blockedUnattributedNote(count),
  blockedUnattributedNote: (count: number) =>
    count === 1
      ? "This filter shows 1 statement marked Possibly blocked but doesn't count it, because it isn't clear yet whether it's waiting. The next refresh checks it again."
      : `This filter shows ${count} statements marked Possibly blocked but doesn't count them, because it isn't clear yet whether they're waiting. The next refresh checks them again.`,
  sessionError: (serviceName: string, reason: string) =>
    `Real-Time Analytics could not start for ${serviceName}: ${reason}`,
};
