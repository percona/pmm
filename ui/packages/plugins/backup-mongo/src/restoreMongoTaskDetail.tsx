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
  readPbmConfigYaml,
  sectionHeadingSx,
  sectionSx,
  tableSx,
} from './pbmTaskDetailShared';

interface DerivedTaskSummary {
  name: string;
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
      typeof (entry as DerivedTaskSummary).name === 'string'
  );
}

function childTaskLabel(name: string, backupType: string): string {
  if (name.endsWith('-pbm-list')) {
    return 'PBM list';
  }
  if (name.endsWith('-pbm-force-resync')) {
    return 'Force resync';
  }
  if (backupType && name.endsWith(`-${backupType}`)) {
    return backupType === 'pbm_physical'
      ? 'Physical restore'
      : 'Logical restore';
  }
  return name;
}

function findRestoreChild(
  parentName: string,
  backupType: string,
  derived: DerivedTaskSummary[]
): DerivedTaskSummary | undefined {
  if (backupType) {
    const expected = derived.find(
      (entry) => entry.name === `${parentName}-${backupType}`
    );
    if (expected) {
      return expected;
    }
  }
  return derived.find(
    (entry) =>
      entry.name.endsWith('-pbm_logical') ||
      entry.name.endsWith('-pbm_physical')
  );
}

function findPbmListChild(
  derived: DerivedTaskSummary[]
): DerivedTaskSummary | undefined {
  return derived.find((entry) => entry.name.endsWith('-pbm-list'));
}

function findForceResyncChild(
  derived: DerivedTaskSummary[]
): DerivedTaskSummary | undefined {
  return derived.find((entry) => entry.name.endsWith('-pbm-force-resync'));
}

export function getRestoreMongoHistoryTaskNames(
  task: Record<string, unknown>
): string[] {
  const parentName = typeof task.name === 'string' ? task.name : '';
  if (!parentName) {
    return [];
  }
  const derivedNames = readDerivedTasks(task).map((entry) => entry.name);
  return [parentName, ...derivedNames];
}

export function getRestoreMongoExecuteActions(
  task: Record<string, unknown>
): TaskExecuteAction[] {
  const parentName = typeof task.name === 'string' ? task.name : '';
  if (!parentName) {
    return [];
  }

  const backupType =
    typeof task.backup_type === 'string' ? task.backup_type : '';
  const derived = readDerivedTasks(task);
  const restoreChild = findRestoreChild(parentName, backupType, derived);
  const pbmList = findPbmListChild(derived);
  const forceResync = findForceResyncChild(derived);

  const actions: TaskExecuteAction[] = [
    {
      label: 'Sync Config',
      taskName: parentName,
      testId: 'restore-mongo-sync-config',
      confirmMessage: `Are you sure you want to sync restore config for "${parentName}" now?`,
    },
  ];

  if (restoreChild?.name) {
    actions.push({
      label: 'Run Restore',
      taskName: restoreChild.name,
      testId: 'restore-mongo-run-restore',
      confirmMessage: `Are you sure you want to run restore for "${parentName}"?`,
    });
  }

  if (pbmList?.name) {
    actions.push({
      label: 'PBM List',
      taskName: pbmList.name,
      testId: 'restore-mongo-pbm-list',
      confirmMessage: `Are you sure you want to run pbm list for "${parentName}"?`,
    });
  }

  if (forceResync?.name) {
    actions.push({
      label: 'Force Resync',
      taskName: forceResync.name,
      testId: 'restore-mongo-force-resync',
      confirmMessage: `Are you sure you want to run pbm config --force-resync for "${parentName}"?`,
    });
  }

  return actions;
}

export function RestoreMongoTaskDetailExtras({
  task,
}: {
  task: Record<string, unknown>;
}) {
  const derived = readDerivedTasks(task);
  const backupType =
    typeof task.backup_type === 'string' ? task.backup_type : '';
  const configYaml = readPbmConfigYaml(task);

  if (!configYaml && derived.length === 0) {
    return null;
  }

  return (
    <>
      <PbmConfigSection task={task} />
      {derived.length > 0 && (
        <Box component="section" sx={sectionSx}>
          <Box component="h2" sx={sectionHeadingSx}>
            Child tasks
          </Box>
          <Box component="table" sx={tableSx}>
            <thead>
              <tr>
                <Box component="th" sx={cellSx}>
                  Task
                </Box>
                <Box component="th" sx={cellSx}>
                  Role
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
                    {childTaskLabel(entry.name, backupType)}
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
    </>
  );
}
