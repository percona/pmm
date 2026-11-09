import { AdvisorRunIntervals } from 'types/settings.types';
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

const parseSeconds = (s: string): number => {
  const match = String(s).match(/^(\d+)s?$/);
  return match ? parseInt(match[1], 10) : 0;
};

export const convertSecondsStringToHour = (secondsStr: string): number =>
  parseSeconds(secondsStr) / 3600;

export const convertHoursStringToSeconds = (hours: string | number): number =>
  Math.round(parseFloat(String(hours)) * 3600);

export const convertCheckIntervalsToHours = (
  sttCheckIntervals: AdvisorRunIntervals | undefined
) => {
  if (!sttCheckIntervals)
    return {
      rareInterval: '24',
      standardInterval: '24',
      frequentInterval: '24',
    };
  return {
    rareInterval: `${convertSecondsStringToHour(sttCheckIntervals.rareInterval)}`,
    standardInterval: `${convertSecondsStringToHour(sttCheckIntervals.standardInterval)}`,
    frequentInterval: `${convertSecondsStringToHour(sttCheckIntervals.frequentInterval)}`,
  };
};
