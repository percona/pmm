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

import { Tab, Tabs } from '@mui/material';
import { SchemaDrivenPlugin } from '@sep/framework';
import { Link, Navigate, Route, Routes, useLocation } from 'react-router-dom';
import {
  BackupMongoTaskDetailExtras,
  getBackupMongoExecuteActions,
  getBackupMongoHistoryTaskNames,
} from './backupMongoTaskDetail';
import { backupMongoCreateRenderField } from './backupMongoCreateForm';
import {
  getRestoreMongoExecuteActions,
  getRestoreMongoHistoryTaskNames,
  RestoreMongoTaskDetailExtras,
} from './restoreMongoTaskDetail';
import {
  restoreMongoCreateForm,
  restoreMongoCreateRenderField,
  restoreMongoEditForm,
} from './restoreMongoCreateForm';
import { BACKUP_APP_NAME, RESTORE_APP_NAME } from './routes';

const BACKUP_DETAIL_SUPPRESS_KEYS = ['derived_tasks', 'latest_pbm_status'];
const RESTORE_DETAIL_SUPPRESS_KEYS = ['derived_tasks'];

/*
 * MUI `Tabs` rather than the hand-rolled <nav> of styled <Link>s this arrived
 * with. That version drew its active underline in a hardcoded `#e74c3c` -- a red
 * that appears nowhere in PMM's palette, which is purple -- and its divider in a
 * light-mode-only `rgba(0, 0, 0, 0.12)`. `Tabs` takes both from the theme, and
 * brings the roving-tabindex keyboard behaviour the plain links did not have.
 * The links stay real anchors via `component={Link}`, so middle-click and
 * open-in-new-tab keep working.
 */
function MongoBackupTabs({ basePath }: { basePath: string }) {
  const { pathname } = useLocation();
  const isRestores = pathname.includes('/restores');

  return (
    <Tabs
      value={isRestores ? 'restores' : 'backups'}
      aria-label="MongoDB backup and restore"
      sx={{ mb: 3, borderBottom: 1, borderColor: 'divider' }}
    >
      <Tab
        label="Backups"
        value="backups"
        component={Link}
        to={`${basePath}/backups`}
      />
      <Tab
        label="Restores"
        value="restores"
        component={Link}
        to={`${basePath}/restores`}
      />
    </Tabs>
  );
}

/**
 * @param basePath Absolute mount path (below the host's router basename) this app
 *   is rendered at. Supplied by the host rather than hardcoded here so the tab
 *   links and each plugin's `routeBase` cannot drift from the route that mounts
 *   them -- a mismatch breaks the plugins' absolute nav (detail back/edit links
 *   and the related-app tab bar) in ways that look like routing bugs.
 */
export function BackupMongoApp({ basePath }: { basePath: string }) {
  return (
    <div>
      <MongoBackupTabs basePath={basePath} />
      <Routes>
        <Route index element={<Navigate to="backups" replace />} />
        <Route
          path="backups/*"
          element={
            <SchemaDrivenPlugin
              pluginName={BACKUP_APP_NAME}
              routeBase={`${basePath}/backups`}
              getTaskExecuteActions={getBackupMongoExecuteActions}
              getTaskHistoryNames={getBackupMongoHistoryTaskNames}
              suppressDetailKeys={BACKUP_DETAIL_SUPPRESS_KEYS}
              renderField={backupMongoCreateRenderField}
              renderTaskDetailChildren={({ task }) => (
                <BackupMongoTaskDetailExtras task={task} />
              )}
            />
          }
        />
        <Route
          path="restores/*"
          element={
            <SchemaDrivenPlugin
              pluginName={RESTORE_APP_NAME}
              routeBase={`${basePath}/restores`}
              getTaskExecuteActions={getRestoreMongoExecuteActions}
              getTaskHistoryNames={getRestoreMongoHistoryTaskNames}
              suppressDetailKeys={RESTORE_DETAIL_SUPPRESS_KEYS}
              renderField={restoreMongoCreateRenderField}
              renderCreateForm={restoreMongoCreateForm}
              renderEditForm={restoreMongoEditForm}
              renderTaskDetailChildren={({ task }) => (
                <RestoreMongoTaskDetailExtras task={task} />
              )}
            />
          }
        />
      </Routes>
    </div>
  );
}
