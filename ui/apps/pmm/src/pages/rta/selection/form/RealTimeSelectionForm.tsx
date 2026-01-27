import { FC, useState, useMemo } from 'react';
import Autocomplete from '@mui/material/Autocomplete';
import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import { enqueueSnackbar } from 'notistack';
import { useUser } from 'contexts/user';
import { Messages } from '../RealTimeSelection.messages';
import {
  ServiceOption,
  getServiceOptions,
  getClusterSelectionState,
  toggleClusterServices,
} from './RealTimeSelectionForm.utils';
import {
  ServiceInput,
  ServiceOption as ServiceOptionComponent,
  ServiceOptionTag,
} from '../components';
import { useAvailableServices, useStartSessions } from 'hooks/api/useRealTime';

interface RealTimeSelectionFormProps {
  onSuccess?: () => void;
}

export const RealTimeSelectionForm: FC<RealTimeSelectionFormProps> = ({
  onSuccess,
}) => {
  const { user } = useUser();
  const canManageRTA = user?.isEditor || user?.isPMMAdmin;

  const [selectedServices, setSelectedServices] = useState<ServiceOption[]>([]);
  const [isOpen, setIsOpen] = useState(false);

  const { availableServices, isLoading } = useAvailableServices();

  const serviceOptions = useMemo(
    () => getServiceOptions(availableServices),
    [availableServices]
  );

  const startSessions = useStartSessions();

  const handleServiceChange = (
    _event: React.SyntheticEvent,
    value: ServiceOption[]
  ) => {
    // Filter out cluster options - only keep real services
    const servicesOnly = value.filter((option) => option.type === 'service');
    setSelectedServices(servicesOnly);
  };

  const handleClusterToggle = (clusterName: string) => {
    const newSelection = toggleClusterServices(
      clusterName,
      serviceOptions,
      selectedServices
    );
    setSelectedServices(newSelection);
  };

  const handleStart = async () => {
    // Filter out cluster headers, keep only real services
    const realServices = selectedServices.filter(
      (s) => s.type === 'service' && s.serviceId
    );

    if (realServices.length === 0) {
      return;
    }

    const serviceIds = realServices.map((service) => service.serviceId!);
    await startSessions.mutateAsync(serviceIds, {
      onSuccess: () => {
        enqueueSnackbar(Messages.startSuccess, { variant: 'success' });
        setSelectedServices([]);
        onSuccess?.();
      },
      onError: (error) => {
        const message =
          error instanceof Error ? error.message : Messages.startError;
        enqueueSnackbar(message, { variant: 'error' });
      }
    });
  };

  const realServicesCount = selectedServices.filter(
    (s) => s.type === 'service' && s.serviceId
  ).length;

  return (
    <Stack gap={3} sx={{ width: '100%' }}>
      <Autocomplete
        multiple
        open={isOpen}
        onOpen={() => setIsOpen(true)}
        onClose={() => setIsOpen(false)}
        options={serviceOptions}
        value={selectedServices}
        onChange={handleServiceChange}
        getOptionLabel={(option) => option.label}
        isOptionEqualToValue={(option, value) => option.id === value.id}
        disableCloseOnSelect
        limitTags={2}
        renderInput={(params) => (
          <ServiceInput
            params={params}
            hasSelectedServices={selectedServices.length > 0}
            isOpen={isOpen}
          />
        )}
        renderTags={(value, getTagProps) =>
          value.slice(0, 2).map((option, index) => (
            <ServiceOptionTag
              key={option.id}
              option={option}
              tagProps={getTagProps({ index })}
            />
          ))
        }
        renderOption={(props, option, { selected }) => (
          <ServiceOptionComponent
            key={option.id}
            option={option}
            props={props}
            selected={selected}
            clusterSelectionState={
              option.type === 'cluster'
                ? getClusterSelectionState(
                  option.label,
                  serviceOptions,
                  selectedServices
                )
                : undefined
            }
            onClusterToggle={handleClusterToggle}
          />
        )}
        loading={isLoading}
        disabled={!canManageRTA || serviceOptions.length === 0}
      />

      <Button
        variant="contained"
        size="large"
        onClick={handleStart}
        disabled={
          realServicesCount === 0 ||
          startSessions.isPending ||
          !canManageRTA
        }
        sx={{
          borderRadius: 999,
          minHeight: 40,
          maxHeight: 40,
          textTransform: 'none',
          py: 0,
        }}
      >
        {Messages.startButton}
      </Button>
    </Stack>
  );
};
