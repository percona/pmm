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

import type { TaskExecuteAction } from '@sep/framework';
import { Box } from '@mui/material';
import {
  cellSx,
  PbmConfigSection,
  preSx,
  readPbmConfigYaml,
  sectionHeadingSx,
  sectionSx,
  tableSx,
} from './pbmTaskDetailShared';

interface DerivedTaskSummary {
  name: string;
  backup_type: string;
  status?: string | null;
}

function readDerivedTasks(task: Record<string, unknown>): DerivedTaskSummary[] {
  if (!Array.isArray(task.derived_tasks)) {
    return [];
  }
  return task.derived_tasks.filter(
    (entry): entry is DerivedTaskSummary =>
      typeof entry === 'object' &&
      entry !== null &&
      typeof (entry as DerivedTaskSummary).name === 'string' &&
      typeof (entry as DerivedTaskSummary).backup_type === 'string'
  );
}

function derivedTaskLabel(backupType: string): string {
  switch (backupType) {
    case 'pbm_logical':
      return 'Logical backup';
    case 'pbm_physical':
      return 'Physical backup';
    case 'pbm_incremental':
      return 'Incremental backup';
    case 'pbm_status':
      return 'PBM status';
    default:
      return backupType;
  }
}

export function getBackupMongoHistoryTaskNames(
  task: Record<string, unknown>
): string[] {
  const parentName = typeof task.name === 'string' ? task.name : '';
  if (!parentName) {
    return [];
  }
  const derivedNames = readDerivedTasks(task).map((entry) => entry.name);
  return [parentName, ...derivedNames];
}

export function getBackupMongoExecuteActions(
  task: Record<string, unknown>
): TaskExecuteAction[] {
  const parentName = typeof task.name === 'string' ? task.name : '';
  if (!parentName) {
    return [];
  }

  const derived = readDerivedTasks(task);
  const logical = derived.find((entry) => entry.backup_type === 'pbm_logical');
  const physical = derived.find(
    (entry) => entry.backup_type === 'pbm_physical'
  );
  const incremental = derived.find(
    (entry) => entry.backup_type === 'pbm_incremental'
  );

  const actions: TaskExecuteAction[] = [
    {
      label: 'Sync Config',
      taskName: parentName,
      testId: 'backup-mongo-sync-config',
      confirmMessage: `Are you sure you want to sync backup config for "${parentName}" now?`,
    },
  ];

  if (logical?.name) {
    actions.push({
      label: 'Run Logical Backup',
      taskName: logical.name,
      testId: 'backup-mongo-logical-backup',
      confirmMessage: `Are you sure you want to run a logical backup for "${parentName}"?`,
    });
  }

  if (physical?.name) {
    actions.push({
      label: 'Run Physical Backup',
      taskName: physical.name,
      testId: 'backup-mongo-physical-backup',
      confirmMessage: `Are you sure you want to run a physical backup for "${parentName}"?`,
    });
  }

  if (incremental?.name) {
    actions.push({
      label: 'Run Incremental Backup',
      taskName: incremental.name,
      testId: 'backup-mongo-incremental-backup',
      confirmMessage: `Are you sure you want to run an incremental backup for "${parentName}"?`,
    });
  }

  return actions;
}

export function BackupMongoTaskDetailExtras({
  task,
}: {
  task: Record<string, unknown>;
}) {
  const derived = readDerivedTasks(task);
  const configYaml = readPbmConfigYaml(task);
  const latestStatus =
    typeof task.latest_pbm_status === 'string' && task.latest_pbm_status.trim()
      ? task.latest_pbm_status
      : null;

  if (!configYaml && derived.length === 0 && !latestStatus) {
    return null;
  }

  return (
    <>
      <PbmConfigSection task={task} />
      {derived.length > 0 && (
        <Box component="section" sx={sectionSx}>
          <Box component="h2" sx={sectionHeadingSx}>
            Derived tasks
          </Box>
          <Box component="table" sx={tableSx}>
            <thead>
              <tr>
                <Box component="th" sx={cellSx}>
                  Task
                </Box>
                <Box component="th" sx={cellSx}>
                  Type
                </Box>
                <Box component="th" sx={cellSx}>
                  Latest status
                </Box>
              </tr>
            </thead>
            <tbody>
              {derived.map((entry) => (
                <tr key={entry.name}>
                  <Box component="td" sx={cellSx}>
                    {entry.name}
                  </Box>
                  <Box component="td" sx={cellSx}>
                    {derivedTaskLabel(entry.backup_type)}
                  </Box>
                  <Box component="td" sx={cellSx}>
                    {entry.status ?? '—'}
                  </Box>
                </tr>
              ))}
            </tbody>
          </Box>
        </Box>
      )}

      {latestStatus && (
        <Box component="section" sx={sectionSx}>
          <Box component="h2" sx={sectionHeadingSx}>
            Latest PBM status
          </Box>
          <Box component="pre" sx={preSx}>
            {latestStatus}
          </Box>
        </Box>
      )}
    </>
  );
}
