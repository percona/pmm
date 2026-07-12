import { FilterType, Severity } from 'types/alert-templates.types';
import { Messages } from './CreateAlertFromTemplate.messages';

export const DEFAULT_DURATION_SECONDS = 60;
export const DEFAULT_INTERVAL_SECONDS = 60;

// Sentinel value for the folder select's "create new folder" option; the new
// folder is created on submit (not up front).
export const CREATE_FOLDER_VALUE = '__create_folder__';

export const SEVERITY_OPTIONS: { value: Severity; label: string }[] = [
  { value: Severity.EMERGENCY, label: 'Emergency' },
  { value: Severity.ALERT, label: 'Alert' },
  { value: Severity.CRITICAL, label: 'Critical' },
  { value: Severity.ERROR, label: 'Error' },
  { value: Severity.WARNING, label: 'Warning' },
  { value: Severity.NOTICE, label: 'Notice' },
  { value: Severity.INFO, label: 'Info' },
  { value: Severity.DEBUG, label: 'Debug' },
];

export const FILTER_TYPE_OPTIONS: { value: FilterType; label: string }[] = [
  { value: FilterType.MATCH, label: Messages.filterTypes.match },
  { value: FilterType.MISMATCH, label: Messages.filterTypes.mismatch },
];
