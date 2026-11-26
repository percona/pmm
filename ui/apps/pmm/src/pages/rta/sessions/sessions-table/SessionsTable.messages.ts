export const Messages = {
  stopAll: 'Stop all',
  stopSelected: 'Stop selected',
  newSession: 'New session',
  stop: 'Stop',
  empty: 'No sessions found',
  selected: (count: number) =>
    count > 1 ? `${count} rows selected:` : `1 row selected:`,
  table: {
    columns: {
      sessionName: 'Session',
      status: 'Status',
    },
    runningFor: 'Running for',
  },
  success: {
    agentStopped: 'Agent stopped',
    agentsStopped: 'Agents stopped',
    allAgentsStopped: 'All agents stopped',
  },
};
