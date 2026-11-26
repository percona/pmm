import { SEVERITY } from 'lib/constants';
import { ADVISOR_SEVERITY_OPTIONS } from 'pages/settings/components/advisors/Advisors.constants';
import {
  type AdvisorCheck,
  type AdvisorSchedule,
  AdvisorScheduleEmailMode,
  type AdvisorScheduleInput,
  type AdvisorTargetService,
  AdvisorTechnology,
} from 'types/advisors.types';
import { type Severity } from 'types/severity.types';
import { Messages } from './AdvisorScheduler.messages';

const MINUTE_MS = 60_000;
const HOUR_MS = 60 * MINUTE_MS;
const DAY_MS = 24 * HOUR_MS;

// mirrors the name limit in api/advisors/v1/advisors.proto
export const MAX_SCHEDULE_NAME_LENGTH = 100;

export const WEEKDAYS_SHORT = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
export const WEEKDAYS_LONG = [
  'Sunday',
  'Monday',
  'Tuesday',
  'Wednesday',
  'Thursday',
  'Friday',
  'Saturday',
];
const MONTHS_SHORT = [
  'Jan',
  'Feb',
  'Mar',
  'Apr',
  'May',
  'Jun',
  'Jul',
  'Aug',
  'Sep',
  'Oct',
  'Nov',
  'Dec',
];

// day numbers as Date.getUTCDay() returns them, listed Monday first
export const WEEK_ORDER = [1, 2, 3, 4, 5, 6, 0];
export const EVERY_DAY = [0, 1, 2, 3, 4, 5, 6];
export const WEEKDAYS = [1, 2, 3, 4, 5];
export const WEEKEND = [0, 6];

export const SCHEDULE_TECHNOLOGIES = [
  AdvisorTechnology.mysql,
  AdvisorTechnology.postgresql,
  AdvisorTechnology.mongodb,
];

export const DEFAULT_SCHEDULE_INPUT: AdvisorScheduleInput = {
  name: 'All checks daily',
  enabled: true,
  daysOfWeek: EVERY_DAY,
  timeOfDay: '06:00',
  checkNames: [],
  serviceIds: [],
  emailMode: AdvisorScheduleEmailMode.settings,
  emailAddresses: [],
  emailSeverities: [],
};

const pad = (n: number) => String(n).padStart(2, '0');

const sameDays = (days: number[], other: number[]) =>
  [...new Set(days)].sort().join() === [...other].sort().join();

export const joinAnd = (items: string[]): string =>
  items.length > 1
    ? `${items.slice(0, -1).join(', ')} and ${items[items.length - 1]}`
    : (items[0] ?? '');

// "Every day", "Mon–Fri", "Mon, Wed, Fri"
export const describeDays = (days: number[]): string => {
  if (sameDays(days, EVERY_DAY)) {
    return Messages.days.everyDay;
  }
  if (sameDays(days, WEEKDAYS)) {
    return Messages.days.weekdays;
  }
  if (sameDays(days, WEEKEND)) {
    return Messages.days.weekend;
  }
  return WEEK_ORDER.filter((day) => days.includes(day))
    .map((day) => WEEKDAYS_SHORT[day])
    .join(', ');
};

// "every weekday", "every Monday and Thursday"
export const daysPhrase = (days: number[]): string => {
  if (sameDays(days, EVERY_DAY)) {
    return Messages.phrases.everyDay;
  }
  if (sameDays(days, WEEKDAYS)) {
    return Messages.phrases.everyWeekday;
  }
  if (sameDays(days, WEEKEND)) {
    return Messages.phrases.everyWeekend;
  }
  return Messages.phrases.every(
    joinAnd(
      WEEK_ORDER.filter((day) => days.includes(day)).map(
        (day) => WEEKDAYS_LONG[day]
      )
    )
  );
};

export const isValidTimeOfDay = (time: string): boolean =>
  /^([01]\d|2[0-3]):[0-5]\d$/.test(time);

// UTC occurrences of a schedule in [from, to)
export const occurrencesBetween = (
  days: number[],
  timeOfDay: string,
  from: Date,
  to: Date
): Date[] => {
  if (!days.length || !isValidTimeOfDay(timeOfDay)) {
    return [];
  }
  const [hours, minutes] = timeOfDay.split(':').map(Number);
  const offset = hours * HOUR_MS + minutes * MINUTE_MS;
  const result: Date[] = [];
  const firstDay = Date.UTC(
    from.getUTCFullYear(),
    from.getUTCMonth(),
    from.getUTCDate()
  );
  for (let day = firstDay; day < to.getTime(); day += DAY_MS) {
    const at = day + offset;
    if (
      days.includes(new Date(day).getUTCDay()) &&
      at >= from.getTime() &&
      at < to.getTime()
    ) {
      result.push(new Date(at));
    }
  }
  return result;
};

// the next occurrences strictly after `after`, as the server computes next_run_at
export const nextRuns = (
  days: number[],
  timeOfDay: string,
  after: Date,
  count: number
): Date[] =>
  occurrencesBetween(
    days,
    timeOfDay,
    new Date(after.getTime() + 1),
    new Date(after.getTime() + (7 * count + 1) * DAY_MS)
  ).slice(0, count);

// spreads items sorted by x over lanes, each taking the lowest lane whose last item is minGap away;
// sets the lane count of each group of colliding items and returns the largest one
export const assignLanes = <
  T extends { x: number; lane: number; lanes: number },
>(
  items: T[],
  minGap: number
): number => {
  let most = 1;
  let start = 0;
  items.forEach((item, i) => {
    const next = items[i + 1];
    if (next && next.x - item.x < minGap) {
      return;
    }
    const group = items.slice(start, i + 1);
    const ends: number[] = [];
    group.forEach((g) => {
      const free = ends.findIndex((end) => g.x - end >= minGap);
      g.lane = free === -1 ? ends.length : free;
      ends[g.lane] = g.x;
    });
    group.forEach((g) => {
      g.lanes = ends.length;
    });
    most = Math.max(most, ends.length);
    start = i + 1;
  });
  return most;
};

export const formatUtcTime = (date: Date): string =>
  `${pad(date.getUTCHours())}:${pad(date.getUTCMinutes())}`;

export const formatUtcDay = (date: Date): string =>
  `${WEEKDAYS_SHORT[date.getUTCDay()]}, ${date.getUTCDate()} ${MONTHS_SHORT[date.getUTCMonth()]}`;

// "Mon, 27 Oct, 06:00 UTC"
export const formatUtc = (date: Date): string =>
  `${formatUtcDay(date)}, ${formatUtcTime(date)} UTC`;

export const formatLocalTime = (date: Date): string =>
  `${pad(date.getHours())}:${pad(date.getMinutes())}`;

export const formatLocalDay = (date: Date): string =>
  `${WEEKDAYS_SHORT[date.getDay()]}, ${date.getDate()} ${MONTHS_SHORT[date.getMonth()]}`;

// "Mon 09:00", the viewer's weekday and time
export const formatLocalWeekdayTime = (date: Date): string =>
  `${WEEKDAYS_SHORT[date.getDay()]} ${formatLocalTime(date)}`;

// 1 when the viewer's calendar date is the day after the UTC date, -1 the day before
export const localDayShift = (date: Date): number =>
  Math.round(
    (Date.UTC(date.getFullYear(), date.getMonth(), date.getDate()) -
      Date.UTC(date.getUTCFullYear(), date.getUTCMonth(), date.getUTCDate())) /
      DAY_MS
  );

// true when the viewer's clock shows UTC at that moment, so a local time adds nothing
export const isUtcViewer = (date: Date): boolean =>
  date.getTimezoneOffset() === 0;

// "09:00 your time, next day"; empty for a viewer on UTC
export const localHint = (date?: Date | null): string => {
  if (!date || isUtcViewer(date)) {
    return '';
  }
  const shift = localDayShift(date);
  const day =
    shift > 0 ? Messages.nextDay : shift < 0 ? Messages.dayBefore : '';
  return [Messages.localTime(formatLocalTime(date)), day]
    .filter(Boolean)
    .join(', ');
};

export const offsetLabel = (at: Date): string => {
  const offset = -at.getTimezoneOffset();
  const abs = Math.abs(offset);
  return `UTC${offset >= 0 ? '+' : '-'}${pad(Math.floor(abs / 60))}:${pad(abs % 60)}`;
};

export const timeZoneLabel = (): string =>
  (Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC').replace(
    /_/g,
    ' '
  );

export const observesDaylightSaving = (year: number): boolean =>
  new Date(Date.UTC(year, 0, 1)).getTimezoneOffset() !==
  new Date(Date.UTC(year, 6, 1)).getTimezoneOffset();

// "in 5 min", "3 h ago", "in 2 days"
export const relativeTime = (date: Date, now: Date): string => {
  const minutes = Math.round((date.getTime() - now.getTime()) / MINUTE_MS);
  const abs = Math.abs(minutes);
  const text =
    abs < 60
      ? Messages.relative.minutes(abs)
      : abs < 2 * 24 * 60
        ? Messages.relative.hours(Math.round(abs / 60))
        : Messages.relative.days(Math.round(abs / (24 * 60)));
  return minutes >= 0
    ? Messages.relative.in(text)
    : Messages.relative.ago(text);
};

// how a schedule's own severities read in the Email cell; Settings severities need no mention
export const severitySummary = (severities?: Severity[]): string => {
  if (!severities?.length) {
    return '';
  }
  if (ADVISOR_SEVERITY_OPTIONS.every((s) => severities.includes(s))) {
    return Messages.allSeverities;
  }
  if (severities.length === 1) {
    return Messages.severityOnly(SEVERITY[severities[0]]);
  }
  return severities.map((s) => SEVERITY[s]).join(', ');
};

// a check runs on a service of its technology unless it is disabled, globally or for the service
export const checkApplies = (
  check: AdvisorCheck,
  service: AdvisorTargetService
): boolean =>
  check.technology === service.technology &&
  check.enabled &&
  !(check.disabledServiceIds ?? []).includes(service.serviceId);

export const scopeChecks = (
  checks: AdvisorCheck[],
  names?: string[]
): AdvisorCheck[] =>
  names?.length ? checks.filter((c) => names.includes(c.name)) : checks;

export const scopeServices = (
  services: AdvisorTargetService[],
  ids?: string[]
): AdvisorTargetService[] =>
  ids?.length ? services.filter((s) => ids.includes(s.serviceId)) : services;

export interface ServiceCoverage {
  service: AdvisorTargetService;
  checks: number;
}

export const serviceCoverage = (
  checks: AdvisorCheck[],
  services: AdvisorTargetService[]
): ServiceCoverage[] =>
  services.map((service) => ({
    service,
    checks: checks.filter((check) => checkApplies(check, service)).length,
  }));

// distinct checks that run on at least one of the services
export const coveredChecks = (
  checks: AdvisorCheck[],
  services: AdvisorTargetService[]
): number =>
  checks.filter((check) => services.some((s) => checkApplies(check, s))).length;

export interface TechnologyCoverage {
  technology: AdvisorTechnology;
  // enabled checks of the technology
  checks: number;
  services: number;
  executions: number;
}

export const technologyCoverage = (
  checks: AdvisorCheck[],
  services: AdvisorTargetService[]
): TechnologyCoverage[] =>
  SCHEDULE_TECHNOLOGIES.map((technology) => {
    const techServices = services.filter((s) => s.technology === technology);
    return {
      technology,
      checks: checks.filter((c) => c.technology === technology && c.enabled)
        .length,
      services: techServices.length,
      executions: techServices.reduce(
        (sum, service) =>
          sum + checks.filter((check) => checkApplies(check, service)).length,
        0
      ),
    };
  });

// a schedule whose picked checks or services are all gone has nothing to run;
// unknown while a list has not loaded
export const missingTargets = (
  schedule: Pick<AdvisorSchedule, 'checkNames' | 'serviceIds'>,
  checks?: AdvisorCheck[],
  services?: AdvisorTargetService[]
) => ({
  checks:
    !!checks &&
    !!schedule.checkNames?.length &&
    !schedule.checkNames.some((name) => checks.some((c) => c.name === name)),
  services:
    !!services &&
    !!schedule.serviceIds?.length &&
    !schedule.serviceIds.some((id) => services.some((s) => s.serviceId === id)),
});

// "Name (copy)", then "Name (copy 2)" and on, within the name length limit
export const copyName = (name: string, taken: string[]): string => {
  const used = new Set(taken.map((n) => n.toLowerCase()));
  for (let i = 1; ; i++) {
    const suffix = i === 1 ? ' (copy)' : ` (copy ${i})`;
    const candidate =
      name.slice(0, MAX_SCHEDULE_NAME_LENGTH - suffix.length) + suffix;
    if (!used.has(candidate.toLowerCase())) {
      return candidate;
    }
  }
};

// the full schedule an update needs, since it replaces the stored one
export const toScheduleInput = (
  schedule: AdvisorSchedule,
  changes: Partial<AdvisorScheduleInput> = {}
): AdvisorScheduleInput => ({
  name: schedule.name,
  enabled: !!schedule.enabled,
  daysOfWeek: schedule.daysOfWeek ?? [],
  timeOfDay: schedule.timeOfDay,
  checkNames: schedule.checkNames ?? [],
  serviceIds: schedule.serviceIds ?? [],
  emailMode: schedule.emailMode,
  emailAddresses: schedule.emailAddresses ?? [],
  emailSeverities: schedule.emailSeverities ?? [],
  ...changes,
});

const SCHEDULE_COLORS = {
  light: [
    '#7056FC',
    '#127EB7',
    '#7D5A08',
    '#B4237D',
    '#1B7A2F',
    '#C2410C',
    '#0F766E',
    '#5B5F6B',
  ],
  dark: [
    '#9387FE',
    '#48B6E8',
    '#FED520',
    '#F07BC1',
    '#52D66D',
    '#F49143',
    '#2DD4BF',
    '#A1A1AA',
  ],
};

// colors follow creation order, so adding a schedule never recolors the others
export const scheduleColors = (
  schedules: AdvisorSchedule[],
  mode: 'light' | 'dark'
): Map<string, string> => {
  const palette = SCHEDULE_COLORS[mode];
  const byAge = [...schedules].sort(
    (a, b) =>
      a.createdAt.localeCompare(b.createdAt) ||
      a.scheduleId.localeCompare(b.scheduleId)
  );
  return new Map(
    byAge.map((s, i) => [s.scheduleId, palette[i % palette.length]])
  );
};
