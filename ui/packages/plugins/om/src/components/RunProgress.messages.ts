/**
 * Copyright (C) 2026 Percona LLC
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <https://www.gnu.org/licenses/>.
 */

export const Messages = {
  attempt: (count: number) => `(attempt ${count})`,
  step: 'Step',
  starting: 'Starting…',
  rollback: 'Rollback',
  abort: 'Abort',
  abortFinished: "Finished installs can't be aborted.",
  abortTitle: 'Abort this deployment?',
  abortBody: (runId: string) =>
    `This stops run ${runId} and rolls back every node in it - packages, configuration, and data directories this run has already written are removed. Nodes that had already finished successfully are rolled back too, since a partial replica set is not a state to leave running.`,
  abortIrreversible: 'This cannot be undone.',
  abortFailed: 'Could not abort the install',
  abortRequested: 'Abort requested - rolling back once the current step stops.',
  cancel: 'Cancel',
  runId: (runId: string) => `Run ${runId}`,
  copied: 'Copied',
  copyRunId: 'Copy run ID',
  installSummary: 'Install summary',
  installed: (replicaSetName: string) =>
    `Replica set ${replicaSetName} is installed`,
  mongodbVersion: 'MongoDB version',
  members: 'Members',
  environment: 'Environment',
  cluster: 'Cluster',
  took: 'Took',
  securityInPlace: 'Security settings still in place',
  viewInFleet: 'View in Fleet',
  viewNodes: 'View nodes',
  rollingBack: 'Rolling back every node.',
  rollbackFailed:
    'The rollback failed. Clean up the nodes named in the error by hand.',
  nothingRolledBack:
    'Nothing had been installed, so there was nothing to roll back.',
  rolledBack: 'Every node was rolled back.',
};
