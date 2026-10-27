import { FC } from 'react';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import UnavailableText from 'components/unavailable-text';
import { capitalize } from 'utils/text.utils';
import {
  AlertSeverity,
  getSeverityBand,
} from 'pages/alerting/status/alert-severity';
import { Messages } from './AlertSeverityDetail.messages';

interface Props {
  severity?: string;
}

const AlertSeverityDetail: FC<Props> = ({ severity }) => {
  if (!severity) {
    return <UnavailableText />;
  }

  const band = getSeverityBand(severity);
  // The band drives the signifier; keep the original level visible when the
  // band hides it (e.g. Emergency shown as Critical).
  const showOriginal = !!band && band !== severity.trim().toLowerCase();

  return (
    <Stack>
      <AlertSeverity severity={severity} />
      {showOriginal && (
        <Typography variant="body2" color="text.secondary">
          {Messages.reportedAs(capitalize(severity.trim().toLowerCase()))}
        </Typography>
      )}
    </Stack>
  );
};

export default AlertSeverityDetail;
