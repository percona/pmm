import { TextSelectOption } from 'components/text-select/TextSelect.types';
import { AlertStatus } from 'types/alerting.types';

export const ALL_STATES_FILTER = '__all_states__';
export const SILENCED_FILTER = '__silenced__';

export const STATUS_COLOR_MAP: Record<
  AlertStatus,
  'default' | 'error' | 'warning' | 'success'
> = {
  Alerting: 'error',
  Pending: 'warning',
  Recovering: 'warning',
  Normal: 'success',
  NoData: 'default',
  Error: 'error',
};

// Key order defines the state filter dropdown order.
export const STATUS_LABEL_MAP: Record<AlertStatus, string> = {
  Normal: 'Normal',
  Pending: 'Pending',
  Alerting: 'Firing',
  Recovering: 'Recovering',
  NoData: 'No Data',
  Error: 'Error',
};

export const STATE_OPTIONS: TextSelectOption<string>[] = [
  { label: 'All', value: ALL_STATES_FILTER },
  ...Object.entries(STATUS_LABEL_MAP).map(([value, label]) => ({
    value,
    label,
  })),
  { label: 'Silenced', value: SILENCED_FILTER },
];
