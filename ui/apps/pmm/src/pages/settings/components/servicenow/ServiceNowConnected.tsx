import { FC } from 'react';
import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Messages } from '../../Settings.messages';
import { MAX_LABEL_WIDTH } from '../../Settings.constants';
import { StoredDeliveryInputs } from './ServiceNowConnection.types';
import { ServiceNowDisconnect } from './ServiceNowDisconnect';

interface Props {
  stored: StoredDeliveryInputs;
  onRenew: () => void;
}

const ConnectionDetail: FC<{
  label: string;
  value: string;
  testId: string;
}> = ({ label, value, testId }) => (
  <Stack>
    <Typography variant="caption" color="text.secondary">
      {label}
    </Typography>
    <Typography variant="body1" data-testid={testId}>
      {value}
    </Typography>
  </Stack>
);

/**
 * The connection as it stands, in place of the form that produced it.
 *
 * Only the endpoint is shown. PMM stores no author or timestamp for a saved
 * setting — SEP answers whether an override exists, not who wrote it — so the
 * rest of the design's detail row waits on a SEP endpoint that can answer it.
 * A blank stored endpoint means SEP is using the receiver its image bakes in.
 * PMM does not know which one that is, so the row names the receiver Percona
 * ships rather than leaving itself empty.
 */
export const ServiceNowConnected: FC<Props> = ({ stored, onRenew }) => {
  const { serviceNow } = Messages;

  return (
    <Stack
      gap={3}
      maxWidth={MAX_LABEL_WIDTH}
      data-testid="servicenow-connected"
    >
      <Typography variant="h6">{serviceNow.connectedTitle}</Typography>

      <ConnectionDetail
        label={serviceNow.endpointDetailLabel}
        value={stored.endpoint || serviceNow.defaultEndpoint}
        testId="servicenow-connected-endpoint"
      />

      <Stack
        direction="row"
        alignItems="center"
        justifyContent="space-between"
        gap={2}
      >
        <Button
          variant="contained"
          onClick={onRenew}
          data-testid="servicenow-renew"
        >
          {serviceNow.renew}
        </Button>
        <ServiceNowDisconnect />
      </Stack>
    </Stack>
  );
};
