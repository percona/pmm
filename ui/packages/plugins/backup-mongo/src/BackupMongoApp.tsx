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
import {
  ClusterScopeProvider,
  clusterScopedRenderField,
} from './clusterScopedFields';
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
import { BACKUP_APP_NAME, CONFIG_APP_NAME, RESTORE_APP_NAME } from './routes';

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
/** The tab segments, in the order an operator meets them. */
const TAB_SEGMENTS = ['config', 'backups', 'restores'] as const;

function MongoBackupTabs({ basePath }: { basePath: string }) {
  const { pathname } = useLocation();
  // Matched against the path rather than tracked in state, so a bookmarked or
  // reloaded URL selects the right tab. Falls back to backups, which is where the
  // index route sends anyone arriving at the app root.
  const current =
    TAB_SEGMENTS.find((segment) => pathname.includes(`/${segment}`)) ??
    'backups';

  return (
    <Tabs
      value={current}
      aria-label="MongoDB configuration, backup and restore"
      sx={{ mb: 3, borderBottom: 1, borderColor: 'divider' }}
    >
      <Tab
        label="Configuration"
        value="config"
        component={Link}
        to={`${basePath}/config`}
      />
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
/**
 * Overrides built once at module scope.
 *
 * Identity-stable and closing over nothing, which is what lets this component stay
 * free of hooks -- its route tree is introspected by calling it as a plain function.
 */
const CONFIG_RENDER_FIELD = clusterScopedRenderField();
const BACKUPS_RENDER_FIELD = clusterScopedRenderField(
  backupMongoCreateRenderField
);
const RESTORES_RENDER_FIELD = clusterScopedRenderField(
  restoreMongoCreateRenderField
);

export function BackupMongoApp({ basePath }: { basePath: string }) {
  return (
    <div>
      {/*
        Wraps the tabs *and* the routes: the switcher renders above the tabs, and
        the forms the routes render read the same selection through context.
      */}
      <ClusterScopeProvider>
        <MongoBackupTabs basePath={basePath} />
        <Routes>
          <Route index element={<Navigate to="backups" replace />} />
          <Route
            // PBM's cluster-wide configuration: storage, point-in-time recovery and
            // the backup options that describe the deployment rather than one run.
            // Nothing custom is passed -- the app declares update=False/delete=False,
            // so the framework derives the surface, and the form needs no field
            // overrides the way the backups and restores forms do.
            path="config/*"
            element={
              <SchemaDrivenPlugin
                pluginName={CONFIG_APP_NAME}
                routeBase={`${basePath}/config`}
                renderField={CONFIG_RENDER_FIELD}
              />
            }
          />
          <Route
            path="backups/*"
            element={
              <SchemaDrivenPlugin
                pluginName={BACKUP_APP_NAME}
                routeBase={`${basePath}/backups`}
                getTaskExecuteActions={getBackupMongoExecuteActions}
                getTaskHistoryNames={getBackupMongoHistoryTaskNames}
                suppressDetailKeys={BACKUP_DETAIL_SUPPRESS_KEYS}
                renderField={BACKUPS_RENDER_FIELD}
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
                renderField={RESTORES_RENDER_FIELD}
                renderCreateForm={restoreMongoCreateForm}
                renderEditForm={restoreMongoEditForm}
                renderTaskDetailChildren={({ task }) => (
                  <RestoreMongoTaskDetailExtras task={task} />
                )}
              />
            }
          />
        </Routes>
      </ClusterScopeProvider>
    </div>
  );
}
