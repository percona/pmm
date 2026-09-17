import { GetUpdatesResponse } from 'types/updates.types';
import { Messages } from './Footer.messages';

/**
 * Formats date to "Month Day, Year, HH:MM UTC"
 * @param date
 * @returns formatted date
 */
export const formatCheckDate = (date: string) => {
  const dateObj = new Date(date);
  const month = dateObj.toLocaleDateString('en-US', { month: 'long' });
  const day = dateObj.getUTCDate();
  const year = dateObj.getUTCFullYear();
  const hours = String(dateObj.getUTCHours()).padStart(2, '0');
  const minutes = String(dateObj.getUTCMinutes()).padStart(2, '0');
  return `${month} ${day}, ${year}, ${hours}:${minutes} UTC`;
};

export const getCheckStatus = (
  versionInfo: GetUpdatesResponse | undefined,
  inProgress: boolean
): string | null => {
  if (inProgress) {
    return Messages.inProgress;
  }

  if (versionInfo?.lastCheck) {
    return Messages.checkedOn(formatCheckDate(versionInfo.lastCheck));
  }

  return null;
};
