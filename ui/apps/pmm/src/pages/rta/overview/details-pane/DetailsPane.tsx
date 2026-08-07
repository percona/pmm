import { FC, useState } from 'react';
import Stack from '@mui/material/Stack';
import Box from '@mui/material/Box';
import CardContent from '@mui/material/CardContent';
import Tabs from '@mui/material/Tabs';
import Tab from '@mui/material/Tab';
import Tooltip from '@mui/material/Tooltip';
import IconButton from '@mui/material/IconButton';
import KeyboardArrowUpOutlinedIcon from '@mui/icons-material/KeyboardArrowUpOutlined';
import KeyboardArrowDownOutlinedIcon from '@mui/icons-material/KeyboardArrowDownOutlined';
import ClearOutlinedIcon from '@mui/icons-material/ClearOutlined';
import Paper from '@mui/material/Paper';
import Slide from '@mui/material/Slide';
import { QueryData } from 'types/rta.types';
import { useEscapeKey } from 'utils/keys.utils';
import { Messages } from './DetailsPane.messages';
import QueryAndDetails from './QueryAndDetails';
import { CodeBlock } from '@percona/percona-ui';

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
  const [tab, setTab] = useState<'details' | 'raw-data'>('details');

  return (
    <Slide in={!!query} direction="up" timeout={{ enter: 300, exit: 200 }}>
      <Paper
        data-testid="query-details-pane"
        aria-hidden={query ? 'false' : 'true'}
        variant="outlined"
        sx={(theme) => ({
          px: 2,
          top: 0,
          left: 0,
          right: 0,
          bottom: 0,
          mt: 2,
          mx: 2,
          // TODO: use theme.shape.borderRadiusMd (8px) once percona-ui
          // publishes the Shape tokens (percona-ui#37, not in 1.0.23)
          borderTopLeftRadius: '8px',
          borderTopRightRadius: '8px',
          borderBottomLeftRadius: 0,
          borderBottomRightRadius: 0,
          borderBottom: 'none',
          position: 'absolute',
          overflow: 'hidden',
          display: 'flex',
          flexDirection: 'column',
          zIndex: theme.zIndex.modal,
        })}
      >
        <Stack
          direction="row"
          justifyContent="space-between"
          sx={{
            borderBottom: 1,
            borderColor: 'divider',
            position: 'sticky',
            top: 0,
            zIndex: 1,
            backgroundColor: 'inherit',
          }}
        >
          <Tabs value={tab} onChange={(_, newValue) => setTab(newValue)}>
            <Tab
              data-testid="details-pane-details-tab"
              value="details"
              label={Messages.tabs.details}
            />
            <Tab
              data-testid="details-pane-raw-data-tab"
              value="raw-data"
              label={Messages.tabs.rawData}
            />
          </Tabs>
          <Stack direction="row" alignItems="center" sx={{ mr: -1.5 }}>
            <Tooltip title={Messages.tooltips.previous} arrow>
              <IconButton
                data-testid="details-pane-prev-button"
                aria-label={Messages.actions.previous}
                onClick={onPrevious}
                disabled={isFirstQuery}
              >
                <KeyboardArrowUpOutlinedIcon />
              </IconButton>
            </Tooltip>
            <Tooltip title={Messages.tooltips.next} arrow>
              <IconButton
                data-testid="details-pane-next-button"
                aria-label={Messages.actions.next}
                onClick={onNext}
                disabled={isLastQuery}
              >
                <KeyboardArrowDownOutlinedIcon />
              </IconButton>
            </Tooltip>
            <Tooltip title={Messages.tooltips.close} arrow>
              <IconButton
                data-testid="details-pane-close-button"
                aria-label={Messages.actions.close}
                onClick={onClose}
              >
                <ClearOutlinedIcon />
              </IconButton>
            </Tooltip>
          </Stack>
        </Stack>
        {query ? (
          <CardContent
            sx={{
              px: 0,
              py: 2,
              '&:last-child': { pb: 2 },
              flexGrow: 1,
              minHeight: 0,
              display: 'flex',
              flexDirection: 'column',
              overflowY: 'auto',
              overflowX: 'hidden',
            }}
          >
            {tab === 'details' && <QueryAndDetails queryData={query} />}
            {tab === 'raw-data' && (
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
            )}
          </CardContent>
        ) : null}
      </Paper>
    </Slide>
  );
};

export default DetailsPane;
