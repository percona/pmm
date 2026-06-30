import { FC, useState } from 'react';
import CardContent from '@mui/material/CardContent';
import IconButton from '@mui/material/IconButton';
import Paper from '@mui/material/Paper';
import Slide from '@mui/material/Slide';
import Stack from '@mui/material/Stack';
import Tab from '@mui/material/Tab';
import Tabs from '@mui/material/Tabs';
import Tooltip from '@mui/material/Tooltip';
import KeyboardArrowDownOutlinedIcon from '@mui/icons-material/KeyboardArrowDownOutlined';
import KeyboardArrowUpOutlinedIcon from '@mui/icons-material/KeyboardArrowUpOutlined';
import { Icon } from 'components/icon';
import { useEscapeKey } from 'utils/keys.utils';
import { AlertsTableRow } from '../AlertsPage.types';
import AlertDetailsTab from './details/AlertDetailsTab';
import { Messages } from './AlertDetailsPane.messages';
import { UseDetailsPaneNavigationResult } from '@percona/percona-ui';
import RawDataTab from './raw-data/RawDataTab';

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
  const [tab, setTab] = useState<'details' | 'raw-data'>('details');

  const handleClose = () => {
    onClose();
    setTab('details');
  };

  useEscapeKey(handleClose);

  if (alert?.type === 'node') {
    return null;
  }

  return (
    <Slide in={!!alert} direction="up">
      <Paper
        data-testid="alert-details-pane"
        aria-hidden={alert ? 'false' : 'true'}
        variant="outlined"
        sx={(theme) => ({
          pb: 1,
          px: 3,
          top: -16,
          left: -16,
          right: -16,
          m: 2,
          bottom: theme.spacing(-2),
          position: 'absolute',
          overflow: 'scroll',
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
              data-testid="alert-details-pane-details-tab"
              value="details"
              label={Messages.tabs.details}
            />
            <Tab
              data-testid="alert-details-pane-raw-data-tab"
              value="raw-data"
              label={Messages.tabs.rawData}
            />
          </Tabs>
          <Stack gap={1} direction="row" alignItems="center">
            <Tooltip title={Messages.tooltips.previous} arrow>
              <IconButton
                data-testid="alert-details-pane-prev-button"
                aria-label={Messages.actions.previous}
                onClick={previous}
                disabled={isFirst}
              >
                <KeyboardArrowUpOutlinedIcon />
              </IconButton>
            </Tooltip>
            <Tooltip title={Messages.tooltips.next} arrow>
              <IconButton
                data-testid="alert-details-pane-next-button"
                aria-label={Messages.actions.next}
                onClick={next}
                disabled={isLast}
              >
                <KeyboardArrowDownOutlinedIcon />
              </IconButton>
            </Tooltip>
            <Tooltip title={Messages.tooltips.close} arrow>
              <IconButton
                data-testid="alert-details-pane-close-button"
                aria-label={Messages.actions.close}
                onClick={handleClose}
              >
                <Icon name="bottom-panel-close" />
              </IconButton>
            </Tooltip>
          </Stack>
        </Stack>
        {alert ? (
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
            {tab === 'details' && <AlertDetailsTab alert={alert} />}
            {tab === 'raw-data' && <RawDataTab alert={alert} />}
          </CardContent>
        ) : null}
      </Paper>
    </Slide>
  );
};

export default AlertDetailsPane;
