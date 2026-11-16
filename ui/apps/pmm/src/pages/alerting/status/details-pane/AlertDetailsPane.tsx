import type { FC } from 'react';
import {
  DetailsPane,
  type UseDetailsPaneNavigationResult,
} from '@percona/peak-ui';
import type { AlertsTableRow } from '../AlertsPage.types';
import AlertDetailsTab from './details/AlertDetailsTab';
import { Messages } from './AlertDetailsPane.messages';
import RawDataTab from './raw-data/RawDataTab';
import { useAlertDetailsPane } from './AlertDetailsPane.utils';

interface Props extends UseDetailsPaneNavigationResult {
  alert?: AlertsTableRow;
  onClose: () => void;
}

const AlertDetailsPane: FC<Props> = ({
  alert,
  onClose,
  isFirst,
  isLast,
  next,
  previous,
}) => {
  const details = useAlertDetailsPane(
    alert?.type === 'alert' ? alert : undefined
  );

  if (alert?.type === 'node') {
    return null;
  }

  return (
    <DetailsPane
      data-testid="alert-details-pane"
      open={!!details}
      onClose={onClose}
      navigation={{ isFirst, isLast, next, previous }}
      labels={Messages.labels}
      previousButtonProps={{ 'data-testid': 'alert-details-pane-prev-button' }}
      nextButtonProps={{ 'data-testid': 'alert-details-pane-next-button' }}
      closeButtonProps={{ 'data-testid': 'alert-details-pane-close-button' }}
      tabs={[
        {
          value: 'details',
          label: Messages.tabs.details,
          tabProps: { 'data-testid': 'alert-details-pane-details-tab' },
          content: details && <AlertDetailsTab details={details} />,
        },
        {
          value: 'raw-data',
          label: Messages.tabs.rawData,
          tabProps: { 'data-testid': 'alert-details-pane-raw-data-tab' },
          content: details && <RawDataTab details={details} />,
        },
      ]}
    />
  );
};

export default AlertDetailsPane;
