import { VersionInfo } from 'types/version.types';
import { formatTimestamp } from 'utils/formatTimestamp';

export const formatVersion = ({ version, timestamp }: VersionInfo) =>
  ` ${version}` + (timestamp ? `, ${formatTimestamp(timestamp)}` : '');
