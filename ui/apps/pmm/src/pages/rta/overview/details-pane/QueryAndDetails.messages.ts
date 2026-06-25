export const Messages = {
  titles: {
    operationId: 'Operation ID',
    elapsedExecTime: 'Elapsed exec. time',
    planSummary: 'Plan summary',
    databaseName: 'Database name',
    collection: 'Collection',
    operation: 'Operation',
    username: 'User name',
    dbInstanceAddress: 'DB instance address',
    clientAppName: 'Client app name',
    operationStartTime: 'Operation start time',
    dataCaptureTime: 'Data capture time',
    clientAddress: 'Client address',
    service: 'Service',
    pid: 'PID',
    state: 'State',
    waitEvent: 'Wait event',
    lockChain: 'Lock chain',
    transactionDuration: 'Transaction duration',
  },
  tooltips: {
    operationId: "The database's internal identifier for this operation.",
    elapsedExecTime: 'How long this operation has been running for.',
    planSummary:
      'High-level summary of how the database is executing this query. For example, using an index or scanning the full collection.',
    databaseName: 'The database/schema where this operation is running.',
    collection: 'The MongoDB collection targeted by this operation',
    operation:
      'The type of action the database is performing, such as query, insert, update, or command.',
    username: 'The database user who started this operation.',
    clientAddress:
      'The IP address and port of the application sending this query.',
    service: 'The PMM service name for this database instance.',
    clientAppName:
      'The name of the application or driver that started this operation.',
    operationStartTime:
      'The exact timestamp when the database started executing this operation.',
    dataCaptureTime:
      'When PMM took this snapshot. Compare with Operation start time to calculate how long the operation has been running so far.',
    dbInstanceAddress:
      'The server hostname and port where this operation is running.',
    pid: 'PostgreSQL backend process ID.',
    state: 'Current session state in pg_stat_activity.',
    waitEvent: 'Wait event type and name when the session is waiting.',
  },
  lockChainEntry: (pid: number, lockMode: string, lockType: string) =>
    `PID ${pid} · ${lockMode} (${lockType})`,
};
