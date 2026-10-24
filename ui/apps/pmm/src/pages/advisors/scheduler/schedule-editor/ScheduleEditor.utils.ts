import { ADVISOR_SEVERITY_OPTIONS } from 'pages/settings/components/advisors/Advisors.constants';
import { splitEmailAddresses } from 'pages/settings/components/advisors/Advisors.utils';
import {
  type AdvisorCheck,
  type AdvisorNotificationSettings,
  type AdvisorSchedule,
  AdvisorScheduleEmailMode,
  type AdvisorScheduleInput,
  type AdvisorTargetService,
} from 'types/advisors.types';
import { type Severity } from 'types/severity.types';
import { z } from 'zod';
import { EVERY_DAY, isValidTimeOfDay } from '../AdvisorScheduler.utils';
import { Messages } from './ScheduleEditor.messages';

// mirrors maxAdvisorNotificationEmailAddresses in managed/models/settings_helpers.go
export const MAX_SCHEDULE_EMAILS = 20;

export type ScopeMode = 'all' | 'selected';
export type SeveritiesMode = 'settings' | 'custom';

export interface ScheduleForm {
  name: string;
  enabled: boolean;
  days: number[];
  time: string;
  checksMode: ScopeMode;
  checks: string[];
  servicesMode: ScopeMode;
  services: string[];
  emailMode: AdvisorScheduleEmailMode;
  emailText: string;
  severitiesMode: SeveritiesMode;
  severities: Severity[];
}

// form fields in the order they appear, for the summary and the focus on submit
export const FORM_FIELDS = [
  'name',
  'days',
  'time',
  'checks',
  'services',
  'email',
  'severities',
] as const;

export type ScheduleFormField = (typeof FORM_FIELDS)[number];
export type ScheduleFormErrors = Partial<Record<ScheduleFormField, string>>;

export const emptyScheduleForm = (): ScheduleForm => ({
  name: '',
  enabled: true,
  days: EVERY_DAY,
  time: '06:00',
  checksMode: 'all',
  checks: [],
  servicesMode: 'all',
  services: [],
  emailMode: AdvisorScheduleEmailMode.settings,
  emailText: '',
  severitiesMode: 'settings',
  severities: [],
});

// picks that no longer exist are dropped, so the form shows what a run would use
export const toScheduleForm = (
  schedule: AdvisorSchedule,
  checks?: AdvisorCheck[],
  services?: AdvisorTargetService[]
): ScheduleForm => {
  const checkNames = schedule.checkNames ?? [];
  const serviceIds = schedule.serviceIds ?? [];
  const severities = schedule.emailSeverities ?? [];
  return {
    name: schedule.name,
    enabled: !!schedule.enabled,
    days: schedule.daysOfWeek ?? [],
    time: schedule.timeOfDay,
    checksMode: checkNames.length ? 'selected' : 'all',
    checks: checks
      ? checkNames.filter((name) => checks.some((c) => c.name === name))
      : checkNames,
    servicesMode: serviceIds.length ? 'selected' : 'all',
    services: services
      ? serviceIds.filter((id) => services.some((s) => s.serviceId === id))
      : serviceIds,
    emailMode: schedule.emailMode,
    emailText: (schedule.emailAddresses ?? []).join(', '),
    severitiesMode: severities.length ? 'custom' : 'settings',
    severities,
  };
};

// splits the recipients field and drops repeats, ignoring case
export const parseEmails = (text: string): string[] => {
  const seen = new Set<string>();
  return splitEmailAddresses(text).filter((address) => {
    const key = address.toLowerCase();
    if (seen.has(key)) {
      return false;
    }
    seen.add(key);
    return true;
  });
};

const emailError = (form: ScheduleForm): string | undefined => {
  if (form.emailMode !== AdvisorScheduleEmailMode.custom) {
    return undefined;
  }
  const list = parseEmails(form.emailText);
  if (!list.length) {
    return Messages.validation.emailRequired;
  }
  const invalid = list.filter(
    (address) => !z.string().email().safeParse(address).success
  );
  if (invalid.length) {
    return Messages.validation.emailInvalid(invalid.join(', '));
  }
  if (list.length > MAX_SCHEDULE_EMAILS) {
    return Messages.validation.emailMax(MAX_SCHEDULE_EMAILS, list.length);
  }
  return undefined;
};

const severitiesError = (form: ScheduleForm): string | undefined =>
  form.emailMode !== AdvisorScheduleEmailMode.off &&
  form.severitiesMode === 'custom' &&
  !form.severities.length
    ? Messages.validation.severities
    : undefined;

// takenNames are the other schedules' names; uniqueness ignores case, as on the server
export const validateScheduleForm = (
  form: ScheduleForm,
  takenNames: string[]
): ScheduleFormErrors => {
  const name = form.name.trim();
  const errors: ScheduleFormErrors = {
    name: !name
      ? Messages.validation.name
      : takenNames.some((n) => n.toLowerCase() === name.toLowerCase())
        ? Messages.validation.nameTaken(name)
        : undefined,
    days: form.days.length ? undefined : Messages.validation.days,
    time: isValidTimeOfDay(form.time) ? undefined : Messages.validation.time,
    checks:
      form.checksMode === 'selected' && !form.checks.length
        ? Messages.validation.checks
        : undefined,
    services:
      form.servicesMode === 'selected' && !form.services.length
        ? Messages.validation.services
        : undefined,
    email: emailError(form),
    severities: severitiesError(form),
  };
  return Object.fromEntries(
    Object.entries(errors).filter(([, message]) => !!message)
  ) as ScheduleFormErrors;
};

// severities in the order Settings lists them, most severe first
export const orderSeverities = (severities: Severity[]): Severity[] =>
  ADVISOR_SEVERITY_OPTIONS.filter((s) => severities.includes(s));

export const toScheduleInputFromForm = (
  form: ScheduleForm
): AdvisorScheduleInput => ({
  name: form.name.trim(),
  enabled: form.enabled,
  daysOfWeek: [...form.days].sort(),
  timeOfDay: form.time,
  checkNames: form.checksMode === 'selected' ? form.checks : [],
  serviceIds: form.servicesMode === 'selected' ? form.services : [],
  emailMode: form.emailMode,
  emailAddresses:
    form.emailMode === AdvisorScheduleEmailMode.custom
      ? parseEmails(form.emailText)
      : [],
  emailSeverities:
    form.emailMode !== AdvisorScheduleEmailMode.off &&
    form.severitiesMode === 'custom'
      ? orderSeverities(form.severities)
      : [],
});

// recipients and severities a test email would use, or null when the form can't send one
export const testEmailTarget = (
  form: ScheduleForm,
  defaults?: AdvisorNotificationSettings
): { to: string[]; severities: Severity[] } | null => {
  if (
    form.emailMode === AdvisorScheduleEmailMode.off ||
    severitiesError(form) ||
    emailError(form)
  ) {
    return null;
  }
  const to =
    form.emailMode === AdvisorScheduleEmailMode.custom
      ? parseEmails(form.emailText)
      : (defaults?.emailAddresses ?? []);
  const severities =
    form.severitiesMode === 'custom'
      ? orderSeverities(form.severities)
      : (defaults?.severities ?? []);
  return to.length && severities.length ? { to, severities } : null;
};
