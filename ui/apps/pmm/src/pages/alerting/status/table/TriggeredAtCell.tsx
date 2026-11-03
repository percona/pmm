import { FC } from 'react';
import { formatTriggeredAt } from './AlertStatusTable.utils';
import { useTimezone } from 'hooks/utils/useTimezone';

const TriggeredAtCell: FC<{ activeAt?: string }> = ({ activeAt }) => {
  const timezone = useTimezone();
  return <>{formatTriggeredAt(activeAt, timezone)}</>;
};

export default TriggeredAtCell;
