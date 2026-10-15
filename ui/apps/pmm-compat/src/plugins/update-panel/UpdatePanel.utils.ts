import { config } from '@grafana/runtime';
import { format } from 'date-fns';
import { enUS } from 'date-fns/locale';

const formatUtc = (timestamp: string, pattern: string) => {
  const date = new Date(timestamp);

  return `${format(date.valueOf() + date.getTimezoneOffset() * 60 * 1000, pattern, { locale: enUS })} UTC`;
};

export const formatDateWithTime = (timestamp: string) =>
  formatUtc(timestamp, 'MMMM dd, H:mm');

export const formatDateWithYear = (timestamp: string) =>
  formatUtc(timestamp, 'MMMM dd, yyyy');

export const isPmmNavEnabled = () => !!config.apps['pmm-compat-app']?.preload;
