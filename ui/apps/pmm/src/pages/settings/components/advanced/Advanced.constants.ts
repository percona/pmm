import { Messages } from '../../Settings.messages';

export const SECONDS = 60;
export const MINUTES = 60;
export const HOURS = 24;
export const SECONDS_IN_DAY = SECONDS * MINUTES * HOURS;
export const MINUTES_IN_DAY = MINUTES * HOURS;
export const MIN_DAYS = 1;
export const MAX_DAYS = 3650;
export const MIN_STT_CHECK_INTERVAL = 0.1;
export const STT_CHECK_INTERVAL_STEP = 0.1;
export const DEFAULT_DATA_RETENTION = '86400s';

export const STT_CHECK_INTERVALS = [
  {
    label: Messages.advanced.sttRareIntervalLabel,
    name: 'rareInterval' as const,
  },
  {
    label: Messages.advanced.sttStandardIntervalLabel,
    name: 'standardInterval' as const,
  },
  {
    label: Messages.advanced.sttFrequentIntervalLabel,
    name: 'frequentInterval' as const,
  },
];

export const TECHNICAL_PREVIEW_DOC_URL = 'https://per.co.na/pmm-feature-status';
