import type { FC } from 'react';
import Box from '@mui/material/Box';
import { CodeBlock, DetailsPane as PeakDetailsPane } from '@percona/peak-ui';
import type { QueryData } from 'types/rta.types';
import { Messages } from './DetailsPane.messages';
import QueryAndDetails from './QueryAndDetails';

interface Props {
  query?: QueryData;
  isFirstQuery: boolean;
  isLastQuery: boolean;
  onClose: () => void;
  onNext: () => void;
  onPrevious: () => void;
}

const DetailsPane: FC<Props> = ({
  query,
  isFirstQuery,
  isLastQuery,
  onClose,
  onNext,
  onPrevious,
}) => (
  <PeakDetailsPane
    data-testid="query-details-pane"
    open={!!query}
    onClose={onClose}
    navigation={{
      isFirst: isFirstQuery,
      isLast: isLastQuery,
      next: onNext,
      previous: onPrevious,
    }}
    labels={Messages.labels}
    previousButtonProps={{ 'data-testid': 'details-pane-prev-button' }}
    nextButtonProps={{ 'data-testid': 'details-pane-next-button' }}
    closeButtonProps={{ 'data-testid': 'details-pane-close-button' }}
    sx={{ mt: 2, mx: 2 }}
    tabs={[
      {
        value: 'details',
        label: Messages.tabs.details,
        tabProps: { 'data-testid': 'details-pane-details-tab' },
        content: query && <QueryAndDetails queryData={query} />,
      },
      {
        value: 'raw-data',
        label: Messages.tabs.rawData,
        tabProps: { 'data-testid': 'details-pane-raw-data-tab' },
        content: query && (
          <Box
            sx={{
              flex: 1,
              minHeight: 0,
              minWidth: 0,
              position: 'relative',
              display: 'grid',
              gridTemplateRows: '1fr',
            }}
          >
            <CodeBlock
              language="json"
              content={query.queryRawJson}
              copyable
              wrap
              sx={{ position: 'absolute', inset: 0, overflowY: 'auto' }}
              data-testid="query-raw-data"
            />
          </Box>
        ),
      },
    ]}
  />
);

export default DetailsPane;
