import { afterAll, beforeAll, describe, expect, it } from 'vitest';
import {
  type AdvisorCheck,
  type AdvisorSchedule,
  AdvisorScheduleEmailMode,
  type AdvisorTargetService,
  AdvisorTechnology,
} from 'types/advisors.types';
import { Severity } from 'types/severity.types';
import {
  copyName,
  coveredChecks,
  daysPhrase,
  describeDays,
  formatUtc,
  localHint,
  missingTargets,
  nextRuns,
  occurrencesBetween,
  offsetLabel,
  relativeTime,
  scheduleColors,
  serviceCoverage,
  severitySummary,
  technologyCoverage,
  toScheduleInput,
} from './AdvisorScheduler.utils';

const check = (
  name: string,
  technology: AdvisorTechnology,
  extra: Partial<AdvisorCheck> = {}
): AdvisorCheck => ({
  name,
  enabled: true,
  description: '',
  summary: name,
  technology,
  category: 'Security',
  userDefined: false,
  ...extra,
});

const service = (
  serviceId: string,
  technology: AdvisorTechnology
): AdvisorTargetService => ({
  serviceId,
  serviceName: serviceId,
  technology,
});

const iso = (dates: Date[]) => dates.map((d) => d.toISOString());

describe('describeDays and daysPhrase', () => {
  it('names the common day sets', () => {
    expect(describeDays([0, 1, 2, 3, 4, 5, 6])).toBe('Every day');
    expect(describeDays([5, 1, 2, 3, 4])).toBe('Mon–Fri');
    expect(describeDays([6, 0])).toBe('Sat, Sun');
    expect(daysPhrase([1, 2, 3, 4, 5])).toBe('every weekday');
    expect(daysPhrase([0, 6])).toBe('every Saturday and Sunday');
  });

  it('lists other days Monday first', () => {
    expect(describeDays([0, 3, 1])).toBe('Mon, Wed, Sun');
    expect(daysPhrase([0, 3, 1])).toBe('every Monday, Wednesday and Sunday');
    expect(daysPhrase([4])).toBe('every Thursday');
  });
});

describe('nextRuns', () => {
  // Friday, 23 Oct 2026
  const friday = new Date('2026-10-23T07:00:00Z');

  it('skips days outside the set, across the weekend', () => {
    expect(iso(nextRuns([1, 2, 3, 4, 5], '06:00', friday, 3))).toEqual([
      '2026-10-26T06:00:00.000Z',
      '2026-10-27T06:00:00.000Z',
      '2026-10-28T06:00:00.000Z',
    ]);
  });

  it('is strictly after the given time, like the server', () => {
    expect(iso(nextRuns([5], '07:00', friday, 1))).toEqual([
      '2026-10-30T07:00:00.000Z',
    ]);
    expect(iso(nextRuns([5], '07:01', friday, 1))).toEqual([
      '2026-10-23T07:01:00.000Z',
    ]);
  });

  it('runs on a single weekday every week, across the year end', () => {
    expect(
      iso(nextRuns([0], '23:59', new Date('2026-12-31T12:00:00Z'), 2))
    ).toEqual(['2027-01-03T23:59:00.000Z', '2027-01-10T23:59:00.000Z']);
  });

  it('plans nothing without days or with a malformed time', () => {
    expect(nextRuns([], '06:00', friday, 1)).toEqual([]);
    expect(nextRuns([1], '6:00', friday, 1)).toEqual([]);
    expect(nextRuns([1], '24:00', friday, 1)).toEqual([]);
  });
});

describe('occurrencesBetween', () => {
  it('includes the start and excludes the end', () => {
    expect(
      iso(
        occurrencesBetween(
          [0, 1, 2, 3, 4, 5, 6],
          '00:00',
          new Date('2026-10-24T00:00:00Z'),
          new Date('2026-10-26T00:00:00Z')
        )
      )
    ).toEqual(['2026-10-24T00:00:00.000Z', '2026-10-25T00:00:00.000Z']);
  });
});

describe('formatting', () => {
  it('formats a UTC time with its weekday', () => {
    expect(formatUtc(new Date('2026-10-26T06:05:00Z'))).toBe(
      'Mon, 26 Oct, 06:05 UTC'
    );
  });

  it('describes a time relative to now', () => {
    const now = new Date('2026-10-24T12:00:00Z');
    expect(relativeTime(new Date('2026-10-24T12:05:00Z'), now)).toBe(
      'in 5 min'
    );
    expect(relativeTime(new Date('2026-10-24T09:00:00Z'), now)).toBe('3 h ago');
    expect(relativeTime(new Date('2026-10-27T12:00:00Z'), now)).toBe(
      'in 3 days'
    );
  });

  it('summarizes a schedule own severities', () => {
    expect(severitySummary([])).toBe('');
    expect(severitySummary([Severity.critical])).toBe('Critical only');
    expect(severitySummary([Severity.critical, Severity.error])).toBe(
      'Critical, Error'
    );
    expect(
      severitySummary([
        Severity.critical,
        Severity.error,
        Severity.warning,
        Severity.info,
      ])
    ).toBe('all severities');
  });
});

describe('viewer time zone', () => {
  const tz = process.env.TZ;

  beforeAll(() => {
    process.env.TZ = 'Asia/Tokyo';
  });

  afterAll(() => {
    process.env.TZ = tz;
  });

  it('gives the local time and flags another local day', () => {
    expect(localHint(new Date('2026-10-26T06:00:00Z'))).toBe('15:00 your time');
    expect(localHint(new Date('2026-10-26T20:00:00Z'))).toBe(
      '05:00 your time, next day'
    );
    expect(offsetLabel(new Date('2026-10-26T06:00:00Z'))).toBe('UTC+09:00');
  });
});

describe('coverage', () => {
  const checks = [
    check('mysql_a', AdvisorTechnology.mysql),
    check('mysql_b', AdvisorTechnology.mysql, { disabledServiceIds: ['my-2'] }),
    check('mysql_off', AdvisorTechnology.mysql, { enabled: false }),
    check('mongo_a', AdvisorTechnology.mongodb),
  ];
  const services = [
    service('my-1', AdvisorTechnology.mysql),
    service('my-2', AdvisorTechnology.mysql),
    service('pg-1', AdvisorTechnology.postgresql),
  ];

  it('counts the checks each service gets, skipping disabled ones', () => {
    expect(
      serviceCoverage(checks, services).map((r) => [
        r.service.serviceId,
        r.checks,
      ])
    ).toEqual([
      ['my-1', 2],
      ['my-2', 1],
      ['pg-1', 0],
    ]);
    expect(coveredChecks(checks, services)).toBe(2);
  });

  it('sums executions per technology', () => {
    expect(technologyCoverage(checks, services)).toEqual([
      {
        technology: AdvisorTechnology.mysql,
        checks: 2,
        services: 2,
        executions: 3,
      },
      {
        technology: AdvisorTechnology.postgresql,
        checks: 0,
        services: 1,
        executions: 0,
      },
      {
        technology: AdvisorTechnology.mongodb,
        checks: 1,
        services: 0,
        executions: 0,
      },
    ]);
  });

  it('flags picks that no longer exist, only once the lists load', () => {
    const picked = { checkNames: ['gone'], serviceIds: ['gone', 'gone-2'] };
    expect(missingTargets(picked, checks, services)).toEqual({
      checks: true,
      services: true,
    });
    expect(
      missingTargets({ serviceIds: ['gone', 'my-1'] }, checks, services)
    ).toEqual({ checks: false, services: false });
    expect(missingTargets(picked)).toEqual({ checks: false, services: false });
  });
});

describe('schedules', () => {
  const schedule: AdvisorSchedule = {
    scheduleId: 'b',
    name: 'Prod security',
    enabled: true,
    daysOfWeek: [1, 2, 3, 4, 5],
    timeOfDay: '00:30',
    checkNames: ['mysql_a'],
    emailMode: AdvisorScheduleEmailMode.custom,
    emailAddresses: ['secops@example.com'],
    emailSeverities: [Severity.critical],
    createdAt: '2026-10-02T00:00:00Z',
    updatedAt: '2026-10-02T00:00:00Z',
  };

  it('names a copy after the original, avoiding taken names', () => {
    expect(copyName('Daily', ['Daily'])).toBe('Daily (copy)');
    expect(copyName('Daily', ['Daily', 'daily (COPY)'])).toBe('Daily (copy 2)');
    expect(copyName('x'.repeat(100), [])).toHaveLength(100);
  });

  it('turns a schedule into the full update payload', () => {
    expect(toScheduleInput(schedule, { enabled: false })).toEqual({
      name: 'Prod security',
      enabled: false,
      daysOfWeek: [1, 2, 3, 4, 5],
      timeOfDay: '00:30',
      checkNames: ['mysql_a'],
      // omitted from the JSON when empty
      serviceIds: [],
      emailMode: AdvisorScheduleEmailMode.custom,
      emailAddresses: ['secops@example.com'],
      emailSeverities: [Severity.critical],
    });
  });

  it('colors schedules by creation order', () => {
    const older = {
      ...schedule,
      scheduleId: 'z',
      createdAt: '2026-10-01T00:00:00Z',
    };
    const colors = scheduleColors([schedule, older], 'light');
    expect(colors.get('z')).toBe('#7056FC');
    expect(colors.get('b')).toBe('#127EB7');
  });
});
