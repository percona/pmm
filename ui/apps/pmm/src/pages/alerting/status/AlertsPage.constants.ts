import { TextSelectOption } from 'components/text-select/TextSelect.types';
import { AlertStatus } from 'types/alerting.types';

export const ALL_STATES_FILTER = '__all_states__';

export const STATUS_COLOR_MAP: Record<
  AlertStatus,
  'default' | 'error' | 'warning' | 'success'
> = {
  Alerting: 'error',
  Pending: 'warning',
  Normal: 'success',
  NoData: 'default',
  Error: 'error',
};

export const STATUS_LABEL_MAP: Record<AlertStatus, string> = {
  Alerting: 'Firing',
  Pending: 'Pending',
  Normal: 'Normal',
  NoData: 'No Data',
  Error: 'Error',
};

export const STATE_OPTIONS: TextSelectOption<string>[] = [
  {
    label: 'All',
    value: ALL_STATES_FILTER,
  },
  {
    label: 'Normal',
    value: 'Normal',
  },
  {
    label: 'Pending',
    value: 'Pending',
  },
  {
    label: 'Firing',
    value: 'Alerting',
  },
];
