import { AdvisorRunIntervals } from 'types/settings.types';

const parseSeconds = (s: string): number => {
  const match = String(s).match(/^(\d+)s?$/);
  return match ? parseInt(match[1], 10) : 0;
};

export const convertSecondsStringToHour = (secondsStr: string): number =>
  parseSeconds(secondsStr) / 3600;

export const convertHoursStringToSeconds = (hours: string | number): number =>
  Math.round(parseFloat(String(hours)) * 3600);

// splitEmailAddresses turns the recipients text field into the array the API takes, tolerating
// commas, semicolons and whitespace between entries.
export const splitEmailAddresses = (value: string): string[] =>
  value
    .split(/[;,\s]+/)
    .map((address) => address.trim())
    .filter(Boolean);

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
