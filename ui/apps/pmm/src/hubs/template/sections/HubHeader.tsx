import RefreshRounded from '@mui/icons-material/RefreshRounded';
import Box from '@mui/material/Box';
import IconButton from '@mui/material/IconButton';
import MenuItem from '@mui/material/MenuItem';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import { Chip, PageHeader, Tooltip } from '@percona/peak-ui';
import { type FC, useState } from 'react';
import { Messages } from '../hub.messages';
import type {
  HubIcon,
  HubMessages,
  HubParams,
  HubViewModel,
} from '../hub.types';
import { HubActions } from './HubActions';
import { HubStatusLine } from './HubStatusLine';
import { LiveAge } from './LiveAge';
import { PrototypeMenu } from './PrototypeMenu';
import { visuallyHidden } from './hub.sx';

// A Select needs a string value; "every environment" is the URL's absent param.
const ALL_ENVIRONMENTS = '__all__';

export interface HubHeaderProps {
  view: HubViewModel;
  icon: HubIcon;
  messages: HubMessages;
  params: HubParams;
  setParams: (next: Partial<HubParams>) => void;
  updatedAt: number;
  refetch: () => void;
  loading: boolean;
}

/** The page's h1, status line and actions, then its filter and freshness row. */
export const HubHeader: FC<HubHeaderProps> = ({
  view,
  icon: Icon,
  messages,
  params,
  setParams,
  updatedAt,
  refetch,
  loading,
}) => {
  const { environment } = view;
  const hasActions = !!(view.actions.addService || view.actions.install);
  // The update time when Refresh was pressed; a newer one means that refresh landed.
  const [refreshedFrom, setRefreshedFrom] = useState<number | null>(null);
  const refreshed = refreshedFrom !== null && updatedAt > refreshedFrom;

  return (
    <PageHeader
      title={messages.name}
      icon={<Icon fontSize="large" />}
      badge={
        view.scenario === 'live' ? undefined : (
          <Chip
            size="small"
            color="info"
            variant="outlined"
            label={Messages.sampleData}
          />
        )
      }
      description={
        <HubStatusLine
          status={view.status}
          name={messages.name}
          unit={messages.unit.cluster}
          canSetUpBackup={view.capabilities.setUpBackup}
          matchesNothing={environment.matchesNothing}
          loading={loading}
        />
      }
      actions={hasActions ? <HubActions actions={view.actions} /> : undefined}
      toolbar={
        <>
          {(environment.options.length > 1 ||
            environment.selected !== null) && (
            <TextField
              select
              size="small"
              label={Messages.environment.label}
              value={environment.selected ?? ALL_ENVIRONMENTS}
              onChange={(event) =>
                setParams({
                  environment:
                    event.target.value === ALL_ENVIRONMENTS
                      ? null
                      : event.target.value,
                })
              }
              sx={{ minWidth: 200, width: { xs: '100%', sm: 'auto' } }}
            >
              <MenuItem value={ALL_ENVIRONMENTS}>
                {Messages.environment.all}
              </MenuItem>
              {environment.options.map((option) => (
                <MenuItem key={option.value} value={option.value}>
                  {option.label}
                </MenuItem>
              ))}
            </TextField>
          )}
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
            <Typography variant="caption" color="text.secondary">
              <LiveAge at={updatedAt} format={Messages.updated} />
            </Typography>
            <Tooltip title={Messages.refresh}>
              <IconButton
                size="small"
                aria-label={Messages.refresh}
                onClick={() => {
                  setRefreshedFrom(updatedAt);
                  refetch();
                }}
              >
                <RefreshRounded />
              </IconButton>
            </Tooltip>
            <Box component="span" role="status" sx={visuallyHidden}>
              {refreshed ? Messages.refreshed : ''}
            </Box>
          </Box>
          <Box sx={{ ml: 'auto' }}>
            <PrototypeMenu params={params} setParams={setParams} />
          </Box>
        </>
      }
    />
  );
};
