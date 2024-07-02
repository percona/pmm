import { VersionInfo } from 'types/updates.types';
import { formatTimestamp } from 'utils/formatTimestamp';

export const formatVersion = ({ version, timestamp, tag }: VersionInfo) => {
  const text =
    ` ${version}` + (timestamp ? `, ${formatTimestamp(timestamp)}` : '');

  if (version === '0.0.0') {
    return `${text}, ${tag}`;
  }

  return text;
};
