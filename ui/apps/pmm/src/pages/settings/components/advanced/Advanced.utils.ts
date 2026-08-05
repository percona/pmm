import { HOURS, MINUTES_IN_DAY, SECONDS_IN_DAY } from './Advanced.constants';

export const convertSecondsToDays = (dataRetention: string): number | '' => {
  if (!dataRetention) return '';
  const value = parseFloat(dataRetention.replace(/[^\d.-]/g, ''));
  const units = dataRetention.slice(-1).toLowerCase();

  switch (units) {
    case 'h':
      return value / HOURS;
    case 'm':
      return value / MINUTES_IN_DAY;
    case 's':
      return value / SECONDS_IN_DAY;
    case 'd':
      return value;
    default:
      return '';
  }
};
