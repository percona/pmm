import { FC, useEffect, useMemo, useState } from 'react';
import CloseIcon from '@mui/icons-material/Close';
import RestoreOutlinedIcon from '@mui/icons-material/RestoreOutlined';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Drawer from '@mui/material/Drawer';
import IconButton from '@mui/material/IconButton';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import ListItemText from '@mui/material/ListItemText';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { enqueueSnackbar } from 'notistack';
import {
  DRAWER_CLOSED_WIDTH,
  DRAWER_WIDTH,
} from 'components/sidebar/drawer/Drawer.constants';
import { useNavigation } from 'contexts/navigation/navigation.hooks';
import { useChangeAdvisorChecks } from 'hooks/api/useAdvisors';
import { useServices } from 'hooks/api/useServices';
import { ServicesAutocompleteInput } from 'pages/rta/components/services-autocomplete-input';
import { AdvisorCheckRow } from 'types/advisors.types';
import { VersionedService } from 'types/services.types';
import { ADVISOR_FAMILY_SERVICE_TYPE } from 'utils/advisors.utils';
import { Messages } from './DisableServicesDrawer.messages';

interface DisableServicesDrawerProps {
  check: AdvisorCheckRow | null;
  onClose: () => void;
}

export const DisableServicesDrawer: FC<DisableServicesDrawerProps> = ({
  check,
  onClose,
}) => {
  const { navOpen } = useNavigation();
  // the overlay never covers the main navigation
  const sidebarWidth = navOpen ? DRAWER_WIDTH : DRAWER_CLOSED_WIDTH;

  const [selectedIds, setSelectedIds] = useState<string[]>([]);

  // drop a stale selection when the drawer is opened for another check
  useEffect(() => {
    setSelectedIds([]);
  }, [check?.checkName]);

  const serviceType = check
    ? ADVISOR_FAMILY_SERVICE_TYPE[check.family]
    : undefined;
  // the picker offers only services of the check's target type
  const { data: servicesResponse } = useServices(
    { serviceType },
    { enabled: !!check && !!serviceType }
  );
  const { mutate: changeChecks, isPending } = useChangeAdvisorChecks();

  const services = useMemo(
    () => Object.values(servicesResponse ?? {}).flat() as VersionedService[],
    [servicesResponse]
  );
  const serviceNames = useMemo(
    () => new Map(services.map((s) => [s.serviceId, s.serviceName])),
    [services]
  );

  const disabledIds = useMemo(
    () => check?.disabledServiceIds ?? [],
    [check?.disabledServiceIds]
  );
  const availableServices = useMemo(
    () => services.filter((s) => !disabledIds.includes(s.serviceId)),
    [services, disabledIds]
  );

  // per-service disabling is unavailable while the check is disabled globally;
  // existing entries are kept and re-apply once the check is enabled again
  const canDisable = !!check?.enabled;

  const handleDisable = () => {
    if (!check || !selectedIds.length) {
      return;
    }
    const count = selectedIds.length;
    changeChecks(
      [{ name: check.checkName, serviceIds: selectedIds, enable: false }],
      {
        onSuccess: () => {
          enqueueSnackbar(Messages.success.disabled(count), {
            variant: 'success',
          });
          // keep the drawer open so more services can be disabled one after another
          setSelectedIds([]);
        },
      }
    );
  };

  const handleEnable = (serviceId: string) => {
    if (!check) {
      return;
    }
    changeChecks(
      [{ name: check.checkName, serviceIds: [serviceId], enable: true }],
      {
        onSuccess: () =>
          enqueueSnackbar(
            Messages.success.enabled(serviceNames.get(serviceId) ?? serviceId),
            { variant: 'success' }
          ),
      }
    );
  };

  return (
    <Drawer
      anchor="bottom"
      open={!!check}
      onClose={onClose}
      slotProps={{
        paper: {
          // @ts-expect-error data-testid is passed through to the DOM
          'data-testid': 'disable-services-drawer',
          sx: {
            // a stable height keeps the picker in the upper half of the
            // viewport, so its popup opens downward inside the drawer
            // instead of flipping up over the page content
            height: '80vh',
            left: { xs: 0, md: sidebarWidth },
            p: 2,
            display: 'flex',
            flexDirection: 'column',
            gap: 2,
            overflowY: 'auto',
          },
        },
      }}
    >
      <Stack direction="row" justifyContent="space-between" alignItems="center">
        <Typography variant="h6">
          {check ? Messages.title(check.summary || check.checkName) : ''}
        </Typography>
        <IconButton
          size="small"
          aria-label={Messages.close}
          onClick={onClose}
          data-testid="disable-services-close"
        >
          <CloseIcon fontSize="small" />
        </IconButton>
      </Stack>

      <Typography variant="body2">{Messages.description}</Typography>

      {check && !check.enabled && (
        <Alert severity="info" data-testid="disable-services-globally-disabled">
          {Messages.disabledGlobally}
        </Alert>
      )}

      <Stack direction="row" gap={2} alignItems="flex-start">
        {/* the picker needs an explicit width: as a row-flex child the
            Autocomplete otherwise collapses to its intrinsic (tiny) size */}
        <Box sx={{ flex: 1, maxWidth: 560 }}>
          <ServicesAutocompleteInput
            services={availableServices}
            serviceIds={selectedIds}
            onServiceIdsChange={setSelectedIds}
            disabled={!canDisable || isPending}
            data-testid="disable-services-picker"
          />
        </Box>
        <Button
          variant="contained"
          disabled={!canDisable || isPending || !selectedIds.length}
          onClick={handleDisable}
          // keep the button on the input's first line even when selected
          // tags make the input grow taller
          sx={{ mt: 1.25 }}
          data-testid="disable-services-submit"
        >
          {Messages.disable}
        </Button>
      </Stack>

      <Typography variant="subtitle2">{Messages.currentlyDisabled}</Typography>
      {disabledIds.length ? (
        <List
          dense
          disablePadding
          // match the picker column width so the re-enable action stays
          // next to the service name instead of the drawer's right edge
          sx={{ maxWidth: 560 }}
          data-testid="disable-services-list"
        >
          {disabledIds.map((serviceId) => (
            <ListItem
              key={serviceId}
              disableGutters
              secondaryAction={
                <Tooltip title={Messages.enable} arrow>
                  <IconButton
                    edge="end"
                    size="small"
                    disabled={isPending}
                    aria-label={Messages.enable}
                    onClick={() => handleEnable(serviceId)}
                    data-testid={`disable-services-enable-${serviceId}`}
                  >
                    <RestoreOutlinedIcon fontSize="small" />
                  </IconButton>
                </Tooltip>
              }
            >
              <ListItemText
                primary={
                  serviceNames.get(serviceId) ??
                  Messages.removedService(serviceId)
                }
              />
            </ListItem>
          ))}
        </List>
      ) : (
        <Typography
          variant="body2"
          color="text.secondary"
          data-testid="disable-services-empty"
        >
          {Messages.noDisabledServices}
        </Typography>
      )}
    </Drawer>
  );
};
