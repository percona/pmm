import { AdvisorCheckResultStatus } from 'types/advisors.types';
import { Severity } from 'types/severity.types';
import { ADVISOR_RESULT_STATUS, SEVERITY } from 'lib/constants';
import { Messages } from './AdvisorInsights.messages';

export const SEVERITY_FILTER_OPTIONS = [
  Severity.critical,
  Severity.error,
  Severity.warning,
  Severity.info,
].map((severity) => ({ label: SEVERITY[severity], value: severity }));

export const STATUS_FILTER_OPTIONS = [
  AdvisorCheckResultStatus.ok,
  AdvisorCheckResultStatus.failed,
  AdvisorCheckResultStatus.error,
].map((status) => ({ label: ADVISOR_RESULT_STATUS[status], value: status }));

export const READ_FILTER_OPTIONS = [
  { label: Messages.read, value: 'true' },
  { label: Messages.unread, value: 'false' },
];
