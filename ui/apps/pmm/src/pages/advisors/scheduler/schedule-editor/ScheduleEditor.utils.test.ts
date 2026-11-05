import { describe, expect, it } from 'vitest';
import {
  type AdvisorCheck,
  type AdvisorSchedule,
  AdvisorScheduleEmailMode,
  AdvisorTechnology,
} from 'types/advisors.types';
import { Severity } from 'types/severity.types';
import {
  emptyScheduleForm,
  parseEmails,
  type ScheduleForm,
  testEmailTarget,
  toScheduleForm,
  toScheduleInputFromForm,
  validateScheduleForm,
} from './ScheduleEditor.utils';

const form = (changes: Partial<ScheduleForm> = {}): ScheduleForm => ({
  ...emptyScheduleForm(),
  name: 'Nightly',
  ...changes,
});

describe('validateScheduleForm', () => {
  it('accepts the defaults once named', () => {
    expect(validateScheduleForm(form(), [])).toEqual({});
  });

  it('reports every invalid field', () => {
    expect(
      validateScheduleForm(
        form({
          name: '  ',
          days: [],
          time: '',
          checksMode: 'selected',
          servicesMode: 'selected',
          emailMode: AdvisorScheduleEmailMode.custom,
          emailText: '',
          severitiesMode: 'custom',
        }),
        []
      )
    ).toEqual({
      name: 'Give the schedule a name.',
      days: 'Pick at least one day.',
      time: 'Enter a time as HH:MM.',
      checks: 'Pick at least one check, or choose All checks.',
      services: 'Pick at least one service, or choose All services.',
      email: 'Enter at least one address, or choose Settings recipients.',
      severities: 'Pick at least one severity, or choose Same as Settings.',
    });
  });

  it('rejects a name taken by another schedule, ignoring case', () => {
    expect(
      validateScheduleForm(form({ name: 'nightly ' }), ['Nightly'])
    ).toEqual({
      name: 'A schedule named "nightly" already exists.',
    });
  });

  it('checks the recipients', () => {
    const custom = (emailText: string) =>
      validateScheduleForm(
        form({ emailMode: AdvisorScheduleEmailMode.custom, emailText }),
        []
      ).email;
    expect(custom('dba@example.com; bad, also bad@')).toBe(
      'Not a valid address: bad, also, bad@'
    );
    expect(
      custom(
        Array.from({ length: 21 }, (_, i) => `u${i}@example.com`).join(',')
      )
    ).toBe('Up to 20 addresses; there are 21.');
    expect(custom('dba@example.com')).toBeUndefined();
  });

  it('ignores the severities when the schedule sends no email', () => {
    expect(
      validateScheduleForm(
        form({
          emailMode: AdvisorScheduleEmailMode.off,
          severitiesMode: 'custom',
        }),
        []
      )
    ).toEqual({});
  });
});

describe('toScheduleInputFromForm', () => {
  it('sends empty lists for All and for Settings choices', () => {
    expect(
      toScheduleInputFromForm(
        form({
          days: [5, 1],
          checks: ['stale'],
          emailText: 'ignored@example.com',
          severities: [Severity.info],
        })
      )
    ).toEqual({
      name: 'Nightly',
      enabled: true,
      daysOfWeek: [1, 5],
      timeOfDay: '06:00',
      checkNames: [],
      serviceIds: [],
      emailMode: AdvisorScheduleEmailMode.settings,
      emailAddresses: [],
      emailSeverities: [],
    });
  });

  it('sends the picks, own recipients and severities most severe first', () => {
    expect(
      toScheduleInputFromForm(
        form({
          checksMode: 'selected',
          checks: ['mysql_a'],
          servicesMode: 'selected',
          services: ['s1'],
          emailMode: AdvisorScheduleEmailMode.custom,
          emailText: 'a@example.com, A@example.com b@example.com',
          severitiesMode: 'custom',
          severities: [Severity.warning, Severity.critical],
        })
      )
    ).toMatchObject({
      checkNames: ['mysql_a'],
      serviceIds: ['s1'],
      emailAddresses: ['a@example.com', 'b@example.com'],
      emailSeverities: [Severity.critical, Severity.warning],
    });
  });
});

describe('toScheduleForm', () => {
  const schedule: AdvisorSchedule = {
    scheduleId: 's',
    name: 'Prod',
    daysOfWeek: [1],
    timeOfDay: '00:30',
    checkNames: ['mysql_a', 'removed_check'],
    serviceIds: ['gone'],
    emailMode: AdvisorScheduleEmailMode.off,
    createdAt: '',
    updatedAt: '',
  };
  const checks = [
    {
      name: 'mysql_a',
      technology: AdvisorTechnology.mysql,
    } as AdvisorCheck,
  ];

  it('loads a schedule and drops picks that no longer exist', () => {
    expect(toScheduleForm(schedule, checks, [])).toEqual({
      name: 'Prod',
      // omitted from the JSON when false
      enabled: false,
      days: [1],
      time: '00:30',
      checksMode: 'selected',
      checks: ['mysql_a'],
      servicesMode: 'selected',
      services: [],
      emailMode: AdvisorScheduleEmailMode.off,
      emailText: '',
      severitiesMode: 'settings',
      severities: [],
    });
  });

  it('keeps the picks while the lists are unknown', () => {
    expect(toScheduleForm(schedule).checks).toEqual([
      'mysql_a',
      'removed_check',
    ]);
  });
});

describe('testEmailTarget', () => {
  const defaults = {
    enabled: true,
    emailAddresses: ['dba@example.com'],
    severities: [Severity.critical, Severity.error],
  };

  it('uses the Settings defaults unless the schedule sets its own', () => {
    expect(testEmailTarget(form(), defaults)).toEqual({
      to: ['dba@example.com'],
      severities: [Severity.critical, Severity.error],
    });
    expect(
      testEmailTarget(
        form({
          emailMode: AdvisorScheduleEmailMode.custom,
          emailText: 'secops@example.com',
          severitiesMode: 'custom',
          severities: [Severity.info],
        }),
        defaults
      )
    ).toEqual({ to: ['secops@example.com'], severities: [Severity.info] });
  });

  it('cannot send without recipients, with bad ones, or with no email', () => {
    expect(testEmailTarget(form(), { enabled: true })).toBeNull();
    expect(
      testEmailTarget(
        form({ emailMode: AdvisorScheduleEmailMode.custom, emailText: 'bad' }),
        defaults
      )
    ).toBeNull();
    expect(
      testEmailTarget(
        form({ emailMode: AdvisorScheduleEmailMode.off }),
        defaults
      )
    ).toBeNull();
  });
});

describe('parseEmails', () => {
  it('drops repeats, ignoring case', () => {
    expect(parseEmails('a@x.io, A@x.io;b@x.io')).toEqual(['a@x.io', 'b@x.io']);
  });
});
