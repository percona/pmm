import { tz } from '@date-fns/tz/tz';
import { formatDistanceStrict } from 'date-fns';
import { format } from 'date-fns/format';
import { PMM_DUMP_ARCHIVE_PATH, TIME_FORMAT } from 'lib/constants';
import { Dump, DumpStatus } from 'types/dump.types';
import { DOWNLOAD_DELAY_MS } from './DumpPage.constants';
import { Messages } from './DumpPage.messages';

export const isDumpReady = ({ status }: Dump) => status === DumpStatus.Success;

export const formatDate = (value: string, timezone: string) => {
  const date = new Date(value);

  return value && !Number.isNaN(date.getTime())
    ? format(date, TIME_FORMAT, { in: tz(timezone) })
    : Messages.notAvailable;
};

export const formatTimeRange = (dump: Dump) => {
  if (!dump.startTime || !dump.endTime) {
    return Messages.notAvailable;
  }

  return formatDistanceStrict(new Date(dump.endTime), new Date(dump.startTime));
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
