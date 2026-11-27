import { TZDate } from '@date-fns/tz';

// The date pickers edit local Date objects, so a value's local fields hold
// the wall-clock time the user means in the PMM timezone.
export const toWallClock = (date: Date, timezone: string) => {
  const zoned = new TZDate(date, timezone);

  return new Date(
    zoned.getFullYear(),
    zoned.getMonth(),
    zoned.getDate(),
    zoned.getHours(),
    zoned.getMinutes(),
    zoned.getSeconds()
  );
};

export const fromWallClock = (date: Date, timezone: string) =>
  new Date(
    new TZDate(
      date.getFullYear(),
      date.getMonth(),
      date.getDate(),
      date.getHours(),
      date.getMinutes(),
      date.getSeconds(),
      timezone
    ).getTime()
  );
