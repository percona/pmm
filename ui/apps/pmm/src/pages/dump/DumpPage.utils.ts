import { tz } from '@date-fns/tz/tz';
import { formatDistanceStrict } from 'date-fns';
import { format } from 'date-fns/format';
import { PMM_DUMP_ARCHIVE_PATH, TIME_FORMAT } from 'lib/constants';
import { type Dump, DumpStatus } from 'types/dump.types';
import { DOWNLOAD_DELAY_MS } from './DumpPage.constants';
import { Messages } from './DumpPage.messages';

const DATA_WINDOW_FORMAT = 'MMM d, HH:mm';

const toDate = (value: string) => {
  const date = new Date(value);

  return value && !Number.isNaN(date.getTime()) ? date : null;
};

export const isDumpReady = ({ status }: Dump) => status === DumpStatus.Success;

export const isDumpCreating = ({ status }: Dump) =>
  status === DumpStatus.InProgress;

export const formatDate = (value: string, timezone: string) => {
  const date = toDate(value);

  return date
    ? format(date, TIME_FORMAT, { in: tz(timezone) })
    : Messages.notAvailable;
};

export const formatDataWindow = (
  { startTime, endTime }: Dump,
  timezone: string
) => {
  const start = toDate(startTime);
  const end = toDate(endTime);

  if (!start || !end) {
    return Messages.notAvailable;
  }

  const formatInZone = (date: Date) =>
    format(date, DATA_WINDOW_FORMAT, { in: tz(timezone) });

  return `${formatInZone(start)} – ${formatInZone(end)}`;
};

export const formatTimeRange = (dump: Dump) => {
  const start = toDate(dump.startTime);
  const end = toDate(dump.endTime);

  return start && end
    ? formatDistanceStrict(end, start)
    : Messages.notAvailable;
};

export const formatRelativeTime = (value: string, now = new Date()) => {
  const date = toDate(value);

  return date
    ? formatDistanceStrict(date, now, { addSuffix: true })
    : Messages.notAvailable;
};

export const getDumpArchiveUrl = (dump: Dump) =>
  `${PMM_DUMP_ARCHIVE_PATH}/${dump.dumpId}.tar.gz${dump.encrypted ? '.enc' : ''}`;

const triggerDownload = (dump: Dump) => {
  const link = document.createElement('a');
  link.href = getDumpArchiveUrl(dump);
  link.download = '';
  document.body.appendChild(link);
  link.click();
  link.remove();
};

export const downloadDumps = async (dumps: Dump[]) => {
  for (const [index, dump] of dumps.entries()) {
    if (index > 0) {
      await new Promise((resolve) => setTimeout(resolve, DOWNLOAD_DELAY_MS));
    }
    triggerDownload(dump);
  }
};
