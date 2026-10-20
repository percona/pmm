import Box from '@mui/material/Box';
import MenuItem from '@mui/material/MenuItem';
import TextField from '@mui/material/TextField';
import { PageHeader } from '@percona/peak-ui';
import { type FC, useId } from 'react';
import { Messages } from '../hub.messages';
import type {
  HubFreshness as Freshness,
  HubIcon,
  HubMessages,
  HubParams,
  HubViewModel,
} from '../hub.types';
import { HubFreshness } from './HubFreshness';
import { HubHeaderActions } from './HubHeaderActions';
import { visuallyHidden } from './hub.sx';

// A Select needs a string value; "every environment" is the URL's absent param.
const ALL_ENVIRONMENTS = '__all__';

export interface HubHeaderProps {
  view: HubViewModel;
  icon: HubIcon;
  messages: HubMessages;
  setParams: (next: Partial<HubParams>) => void;
  freshness: Freshness;
  refetch: () => void;
}

/** Where the reader is: the technology, the environment it shows, how current it is, and the actions that follow from it. */
export const HubHeader: FC<HubHeaderProps> = ({
  view,
  icon: Icon,
  messages,
  setParams,
  freshness,
  refetch,
}) => {
  const { environment } = view;
  const labelId = useId();
  const hasEnvironments =
    environment.options.length > 1 || environment.selected !== null;

  return (
    <PageHeader
      title={messages.name}
      titleVariant="h3"
      icon={<Icon sx={{ fontSize: 36 }} />}
      divider
      toolbarPlacement="inline"
      toolbar={
        <>
          {hasEnvironments && (
            <>
              <Box component="span" id={labelId} sx={visuallyHidden}>
                {Messages.environment.label}
              </Box>
              <TextField
                select
                size="small"
                value={environment.selected ?? ALL_ENVIRONMENTS}
                onChange={(event) =>
                  setParams({
                    environment:
                      event.target.value === ALL_ENVIRONMENTS
                        ? null
                        : event.target.value,
                  })
                }
                // The value alone does not say what it filters, so the hidden label names the field.
                slotProps={{ select: { labelId } }}
                sx={{ width: 240, maxWidth: '100%' }}
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
            </>
          )}
          <HubFreshness freshness={freshness} onRefresh={refetch} />
        </>
      }
      actions={
        view.actions.add.length || view.actions.more.length ? (
          <HubHeaderActions name={messages.name} actions={view.actions} />
        ) : undefined
      }
    />
  );
};
