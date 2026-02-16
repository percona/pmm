import { FC, useState } from 'react';
import Stack from '@mui/material/Stack';
import CardContent from '@mui/material/CardContent';
import Tabs from '@mui/material/Tabs';
import Tab from '@mui/material/Tab';
import IconButton from '@mui/material/IconButton';
import KeyboardArrowUpOutlinedIcon from '@mui/icons-material/KeyboardArrowUpOutlined';
import KeyboardArrowDownOutlinedIcon from '@mui/icons-material/KeyboardArrowDownOutlined';
import { Icon } from 'components/icon';
import Paper from '@mui/material/Paper';
import Slide from '@mui/material/Slide';
import { QueryData } from 'types/rta.types';
import { useEscapeKey } from 'utils/keys.utils';
import { Messages } from './DetailsPane.messages';
import QueryAndDetails from './QueryAndDetails';
import { SyntaxHighlighter } from 'components/syntax-highlighter';

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
}) => {
  useEscapeKey(onClose);
  const [tab, setTab] = useState(0);

  return (
    <Slide in={!!query} direction="up">
      <Paper
        data-testid="query-details-pane"
        aria-hidden={query ? 'false' : 'true'}
        variant="outlined"
        sx={(theme) => ({
          pb: 1,
          px: 3,
          top: 0,
          left: 0,
          right: 0,
          m: 2,
          bottom: theme.spacing(-2),
          position: 'absolute',
          overflow: 'scroll',
          zIndex: theme.zIndex.modal,
        })}
      >
        <Stack direction="row" justifyContent="space-between" sx={{ borderBottom: 1, borderColor: 'divider', position: 'sticky', top: 0, zIndex: 1, backgroundColor: 'inherit' }}>
          <Tabs value={tab} onChange={(_, newValue) => setTab(newValue)}>
            <Tab
              data-testid="details-pane-details-tab"
              value={0}
              label={Messages.tabs.details}
            />
            <Tab
              data-testid="details-pane-raw-data-tab"
              value={1}
              label={Messages.tabs.rawData}
            />
          </Tabs>
          <Stack gap={1} direction="row" alignItems="center">
            <IconButton
              data-testid="details-pane-prev-button"
              aria-label={Messages.actions.previous}
              onClick={onPrevious}
              disabled={isFirstQuery}
            >
              <KeyboardArrowUpOutlinedIcon />
            </IconButton>
            <IconButton
              data-testid="details-pane-next-button"
              aria-label={Messages.actions.next}
              onClick={onNext}
              disabled={isLastQuery}
            >
              <KeyboardArrowDownOutlinedIcon />
            </IconButton>
            <IconButton
              data-testid="details-pane-close-button"
              aria-label={Messages.actions.close}
              onClick={onClose}
            >
              <Icon name="bottom-panel-close" />
            </IconButton>
          </Stack>
        </Stack>
        {query ? (
          <CardContent
            sx={{
              p: 0,
              pt: 3,
              flexGrow: 1,
              minHeight: 300,
              overflowY: 'auto',
              overflowX: 'hidden',
            }}
          >
            {tab === 0 && (
              <QueryAndDetails queryData={query} />
            )}
            {tab === 1 && (
              <SyntaxHighlighter language="json" content={query.queryRawJson} showCopyButton showLineNumbers maxHeight="80vh" />
            )}
          </CardContent>
        ) : null}
      </Paper>
    </Slide>
  );
};

export default DetailsPane;
