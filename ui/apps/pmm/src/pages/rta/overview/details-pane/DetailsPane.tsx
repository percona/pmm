import { FC } from 'react';
import Stack from '@mui/material/Stack';
import CardContent from '@mui/material/CardContent';
import Tabs from '@mui/material/Tabs';
import Tab from '@mui/material/Tab';
import IconButton from '@mui/material/IconButton';
import KeyboardArrowUpOutlinedIcon from '@mui/icons-material/KeyboardArrowUpOutlined';
import KeyboardArrowDownOutlinedIcon from '@mui/icons-material/KeyboardArrowDownOutlined';
import { Icon } from 'components/icon';
import Paper from '@mui/material/Paper';
import { SyntaxHighlighter } from 'components/syntax-highlighter';
import Slide from '@mui/material/Slide';
import Divider from '@mui/material/Divider';
import { QueryData } from 'types/rta.types';
import Typography from '@mui/material/Typography';
import { useEscapeKey } from 'utils/keys.utils';
import { Messages } from './DetailsPane.messages';

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

  return (
    <Slide in={!!query} direction="up">
      <Paper
        variant="outlined"
        sx={(theme) => ({
          p: 1,
          px: 2,
          top: 0,
          left: 0,
          right: 0,
          m: 2,
          bottom: theme.spacing(-2),
          position: 'absolute',
          zIndex: theme.zIndex.modal,
        })}
      >
        <Stack direction="row" justifyContent="space-between">
          <Tabs value={0}>
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
        <Divider sx={{}} />
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
            <Stack gap={1} mb={1}>
              <Typography variant="h6">{query.serviceName}</Typography>
              <Typography variant="body2">{query.queryId}</Typography>
              <Typography variant="body2">{query.state}</Typography>
            </Stack>
            <SyntaxHighlighter language="mongodb" showLineNumbers={true}>
              {query.queryText}
            </SyntaxHighlighter>
          </CardContent>
        ) : null}
      </Paper>
    </Slide>
  );
};

export default DetailsPane;
