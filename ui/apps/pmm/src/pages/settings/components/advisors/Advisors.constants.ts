import { Severity } from 'types/severity.types';
import { Messages } from '../../Settings.messages';

export const MIN_ADVISOR_CHECK_INTERVAL = 0.1;
// 30 days, matches the pmm-managed default
export const DEFAULT_ADVISOR_RETENTION = '2592000s';

export const ADVISOR_SEVERITY_OPTIONS = [
  Severity.emergency,
  Severity.alert,
  Severity.critical,
  Severity.error,
  Severity.warning,
  Severity.notice,
  Severity.info,
  Severity.debug,
];

export const STT_CHECK_INTERVALS = [
  {
    label: Messages.advisors.rareIntervalLabel,
    name: 'rareInterval' as const,
  },
  {
    label: Messages.advisors.standardIntervalLabel,
    name: 'standardInterval' as const,
  },
  {
    label: Messages.advisors.frequentIntervalLabel,
    name: 'frequentInterval' as const,
  },
];
