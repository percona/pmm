/**
 * The Backups tab: what PBM holds first, what was asked for second.
 *
 * Mirrors `ConfigTab`, for the same reason. The framework's task table answers "what
 * did someone submit", and that is worth keeping -- it is the audit trail, and it is
 * where a backup gets scheduled. It is not what the tab is named after. A restore
 * starts from a backup that exists, and whether one exists is PBM's answer, not the
 * task log's.
 */
import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Typography,
} from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { SchemaDrivenPlugin, type RenderFieldOverride } from '@sep/framework';
import { useLocation } from 'react-router-dom';

import { PbmBackupsPanel } from './PbmBackupsPanel';
import { showsList } from './ConfigTab';
import { useClusterScope } from './clusterScopedFields';
import { BACKUP_APP_NAME } from './routes';
import {
  BackupMongoTaskDetailExtras,
  getBackupMongoExecuteActions,
  getBackupMongoHistoryTaskNames,
} from './backupMongoTaskDetail';

/**
 * Detail keys the framework should not print raw.
 *
 * `derived_tasks` is the cascade's internal fan-out and `latest_pbm_status` is
 * rendered properly by `BackupMongoTaskDetailExtras`.
 */
const BACKUP_DETAIL_SUPPRESS_KEYS = ['derived_tasks', 'latest_pbm_status'];

/**
 * @param basePath The app's mount path, from which the backups route is derived.
 * @param renderField The cluster-scoped field override for the create form.
 */
export function BackupsTab({
  basePath,
  renderField,
}: {
  basePath: string;
  renderField: RenderFieldOverride;
}) {
  const { pathname } = useLocation();
  const { cluster } = useClusterScope();
  const routeBase = `${basePath}/backups`;

  const plugin = (
    <SchemaDrivenPlugin
      pluginName={BACKUP_APP_NAME}
      routeBase={routeBase}
      // The tabs above already offer Configuration, Backups and Restores; the
      // bar this plugin derives from `related_apps` is a second control to the
      // same three places, rendered below the switcher that scopes them.
      hideRelatedAppTabs
      getTaskExecuteActions={getBackupMongoExecuteActions}
      getTaskHistoryNames={getBackupMongoHistoryTaskNames}
      suppressDetailKeys={BACKUP_DETAIL_SUPPRESS_KEYS}
      renderField={renderField}
      renderTaskDetailChildren={({ task }) => (
        <BackupMongoTaskDetailExtras task={task} />
      )}
    />
  );

  // Only the list folds away. Collapsing unconditionally would hide the create
  // form and the task detail pages behind a closed section.
  if (!showsList(pathname, routeBase)) {
    return (
      <>
        <PbmBackupsPanel cluster={cluster} />
        {plugin}
      </>
    );
  }

  return (
    <>
      <PbmBackupsPanel cluster={cluster} />
      <Accordion defaultExpanded={false} disableGutters>
        <AccordionSummary expandIcon={<ExpandMoreIcon />}>
          <Typography variant="subtitle2">
            Backup tasks — what was scheduled or submitted, and by whom
          </Typography>
        </AccordionSummary>
        <AccordionDetails>{plugin}</AccordionDetails>
      </Accordion>
    </>
  );
}
