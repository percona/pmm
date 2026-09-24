import { FC } from 'react';
import { AlertSeverity as Severity } from 'types/alerting.types';
import UnavailableText from 'components/unavailable-text';
import { AlertSeverity } from 'pages/alerting/status/alert-severity';

interface Props {
  severity?: string;
}

const AlertSeverityDetail: FC<Props> = ({ severity }) => {
  if (!severity) {
    return <UnavailableText />;
  }

  return <AlertSeverity severity={severity as Severity} />;
};

export default AlertSeverityDetail;
