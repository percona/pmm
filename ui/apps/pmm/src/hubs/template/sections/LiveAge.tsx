import type { FC } from 'react';
import { useLiveTimestamp } from 'hooks/useLiveTimestamp';
import { formatAge } from './hub.time';

export interface LiveAgeProps {
  /** ISO time or epoch milliseconds. */
  at: string | number;
  /** Wraps the age, "20s ago", in its sentence. */
  format?: (age: string) => string;
  /** Past a minute, whole minutes only: "32m ago", not "32m 8s ago". */
  coarse?: boolean;
}

const MINUTE_MS = 60_000;

/** An age such as "20s ago" that keeps itself current. */
export const LiveAge: FC<LiveAgeProps> = ({
  at,
  format = (age) => age,
  coarse = false,
}) => {
  useLiveTimestamp();
  const ms = typeof at === 'number' ? at : Date.parse(at);
  const now = Date.now();
  const age = now - ms;
  // Ages the formatter reads in whole minutes drop the seconds it would add.
  const shown =
    coarse && age >= MINUTE_MS
      ? now - Math.floor(age / MINUTE_MS) * MINUTE_MS
      : ms;

  return <>{format(formatAge(new Date(shown).toISOString(), now))}</>;
};
