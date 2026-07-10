import { formatDistanceStrict } from 'date-fns';
import { Dump, DumpStatus } from 'types/dump.types';
import { DOWNLOAD_DELAY_MS } from './DumpPage.constants';
import { Messages } from './DumpPage.messages';

export const getStatusLabel = (status: DumpStatus) => {
  switch (status) {
    case DumpStatus.InProgress:
      return Messages.status.inProgress;
    case DumpStatus.Success:
      return Messages.status.success;
    case DumpStatus.Error:
      return Messages.status.error;
    default:
      return Messages.status.unspecified;
  }
};

export const formatDate = (value: string) =>
  value
    ? new Intl.DateTimeFormat(undefined, {
        dateStyle: 'medium',
        timeStyle: 'medium',
      }).format(new Date(value))
    : Messages.notAvailable;

export const formatTimeRange = (dump: Dump) => {
  if (!dump.startTime || !dump.endTime) {
    return Messages.notAvailable;
  }

  return formatDistanceStrict(new Date(dump.endTime), new Date(dump.startTime));
};

export const getDumpArchiveUrl = (dump: Dump) =>
  `/dump/${dump.dumpId}.tar.gz${dump.encrypted ? '.enc' : ''}`;

const triggerDownload = (dump: Dump) => {
  const link = document.createElement('a');
  link.href = getDumpArchiveUrl(dump);
  link.download = '';
  document.body.appendChild(link);
  link.click();
  link.remove();
};

export const downloadDumps = async (dumps: Dump[]) => {
  const downloadable = dumps.filter(
    ({ status }) => status === DumpStatus.Success
  );

  for (const [index, dump] of downloadable.entries()) {
    if (index > 0) {
      await new Promise((resolve) => setTimeout(resolve, DOWNLOAD_DELAY_MS));
    }
    triggerDownload(dump);
  }
};
