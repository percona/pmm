import { CurrentInfo, LatestInfo } from 'types/updates.types';
import { formatTimestamp } from 'utils/formatTimestamp';

export const formatVersion = ({
  version,
  timestamp,
  tag,
}: Partial<LatestInfo & CurrentInfo>) => {
  const text =
    ` ${version}` + (timestamp ? `, ${formatTimestamp(timestamp)}` : '');

  if (version === '0.0.0' && tag) {
    return `${text}, ${tag}`;
  }

  return text;
};
