import { Settings, UpdateSettingsPayload } from 'types/settings.types';
import { AdvisorsFormValues } from './AdvisorsForm.schema';
import { SECONDS_IN_DAY } from '../advanced/Advanced.constants';
import { convertSecondsToDays } from '../advanced/Advanced.utils';
import {
  DEFAULT_ADVISOR_NOTIFICATION_SEVERITIES,
  DEFAULT_ADVISOR_RETENTION,
} from './Advisors.constants';
import {
  convertCheckIntervalsToHours,
  convertHoursStringToSeconds,
  splitEmailAddresses,
} from './Advisors.utils';

export const toFormValues = (settings: Settings): AdvisorsFormValues => ({
  stt: settings.advisorEnabled,
  ...convertCheckIntervalsToHours(settings.advisorRunIntervals),
  advisorRetention: String(
    convertSecondsToDays(
      settings.advisorHistoryRetention ?? DEFAULT_ADVISOR_RETENTION
    ) || '1'
  ),
  advisorNotifications: settings.advisorNotificationsEnabled ?? false,
  advisorSeverities: settings.advisorNotificationSeverities?.length
    ? settings.advisorNotificationSeverities
    : DEFAULT_ADVISOR_NOTIFICATION_SEVERITIES,
  advisorNotificationEmails: (
    settings.advisorNotificationEmailAddresses ?? []
  ).join(', '),
});

export const toPayload = (
  values: AdvisorsFormValues
): UpdateSettingsPayload => {
  const advisorRunIntervals = values.stt
    ? {
        rareInterval: `${convertHoursStringToSeconds(values.rareInterval)}s`,
        standardInterval: `${convertHoursStringToSeconds(values.standardInterval)}s`,
        frequentInterval: `${convertHoursStringToSeconds(values.frequentInterval)}s`,
      }
    : undefined;

  return {
    enableAdvisor: values.stt,
    advisorRunIntervals,
    advisorHistoryRetention: `${Math.round(parseFloat(values.advisorRetention)) * SECONDS_IN_DAY}s`,
    enableAdvisorNotifications: values.advisorNotifications,
    advisorNotificationSeverities: values.advisorSeverities,
    advisorNotificationEmailAddresses: {
      values: splitEmailAddresses(values.advisorNotificationEmails),
    },
  };
};
