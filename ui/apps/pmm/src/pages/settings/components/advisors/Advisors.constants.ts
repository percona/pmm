import { Severity } from 'types/severity.types';
import { Messages } from '../../Settings.messages';

// whole hours: a number input steps from its min, so a fractional min would make
// every spinner value fractional too
export const MIN_ADVISOR_CHECK_INTERVAL = 1;
// 30 days, matches the pmm-managed default
export const DEFAULT_ADVISOR_RETENTION = '2592000s';

export const ADVISOR_SEVERITY_OPTIONS = [
  Severity.critical,
  Severity.error,
  Severity.warning,
  Severity.info,
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
