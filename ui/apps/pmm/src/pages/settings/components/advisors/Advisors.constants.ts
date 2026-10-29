import { Severity } from 'types/severity.types';

// 30 days, matches the pmm-managed default
export const DEFAULT_ADVISOR_RETENTION = '2592000s';

export const ADVISOR_SEVERITY_OPTIONS = [
  Severity.critical,
  Severity.error,
  Severity.warning,
  Severity.info,
];

// mirrors DefaultAdvisorNotificationSeverities in managed/models/settings.go
export const DEFAULT_ADVISOR_NOTIFICATION_SEVERITIES = [
  Severity.critical,
  Severity.error,
];
