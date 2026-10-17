import type {
  FailedChecksCounts,
  FailedCheckSummary,
} from 'plugins/shared/types/advisors.types';

export const splitSeverities = (
  checks: FailedCheckSummary[] = []
): FailedChecksCounts => {
  const result: FailedChecksCounts = {
    emergency: 0,
    critical: 0,
    alert: 0,
    error: 0,
    warning: 0,
    debug: 0,
    info: 0,
    notice: 0,
  };

  checks.forEach(({ counts }) => {
    result.emergency += counts.emergency;
    result.critical += counts.critical;
    result.alert += counts.alert;
    result.error += counts.error;
    result.warning += counts.warning;
    result.debug += counts.debug;
    result.info += counts.info;
    result.notice += counts.notice;
  });

  return result;
};

export const sumSeverities = ({
  emergency,
  critical,
  alert,
  error,
  warning,
  debug,
  info,
  notice,
}: FailedChecksCounts) =>
  emergency + critical + alert + error + warning + debug + info + notice;
