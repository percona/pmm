import type { FC } from 'react';
import { useLiveTimestamp } from 'hooks/useLiveTimestamp';
import { formatAge } from './hub.time';

export interface LiveAgeProps {
  /** ISO time or epoch milliseconds. */
  at: string | number;
  /** Wraps the age, "20s ago", in its sentence. */
  format?: (age: string) => string;
}

/** An age such as "20s ago" that keeps itself current. */
export const LiveAge: FC<LiveAgeProps> = ({ at, format = (age) => age }) => {
  useLiveTimestamp();
  const iso = typeof at === 'number' ? new Date(at).toISOString() : at;

  return <>{format(formatAge(iso))}</>;
};
