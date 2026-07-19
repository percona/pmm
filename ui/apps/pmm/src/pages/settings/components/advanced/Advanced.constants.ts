import { Messages } from '../../Settings.messages';

export const SECONDS = 60;
export const MINUTES = 60;
export const HOURS = 24;
export const SECONDS_IN_DAY = SECONDS * MINUTES * HOURS;
export const MINUTES_IN_DAY = MINUTES * HOURS;
export const MIN_DAYS = 1;
export const MAX_DAYS = 3650;
export const DEFAULT_DATA_RETENTION = '86400s';

export const TECHNICAL_PREVIEW_DOC_URL = 'https://per.co.na/pmm-feature-status';

export const FEATURE_MANAGEMENT_SETTINGS = [
  {
    name: 'updates' as const,
    label: Messages.advanced.updatesLabel,
    tooltip: Messages.advanced.updatesTooltip,
    link: Messages.advanced.updatesLink,
    testId: 'advanced-updates',
  },
  {
    name: 'alerting' as const,
    label: Messages.advanced.alertingLabel,
    tooltip: Messages.advanced.alertingTooltip,
    link: Messages.advanced.alertingLink,
    testId: 'advanced-alerting',
  },
  {
    name: 'backup' as const,
    label: Messages.advanced.backupLabel,
    tooltip: Messages.advanced.backupTooltip,
    link: Messages.advanced.backupLink,
    testId: 'advanced-backup',
  },
  {
    name: 'enableInternalPgQan' as const,
    label: Messages.advanced.enableInternalPgQanLabel,
    tooltip: Messages.advanced.enableInternalPgQanTooltip,
    link: Messages.advanced.enableInternalPgQanLink,
    testId: 'enable-internal-pg-qan',
  },
];
