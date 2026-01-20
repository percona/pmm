import { FC, useState, useMemo } from 'react';
import Autocomplete from '@mui/material/Autocomplete';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Checkbox from '@mui/material/Checkbox';
import Chip from '@mui/material/Chip';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import CloseIcon from '@mui/icons-material/Close';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { enqueueSnackbar } from 'notistack';
import { useUser } from 'contexts/user';
import { useRunningRealtimeAgents, REALTIME_AGENTS_QUERY_KEY } from 'hooks/api/useRealtime';
import { Messages } from './RealTimeSelection.messages';
import { changeRealtimeAnalytics } from 'api/realtime';
import { listServices } from 'api/services';
import { ServiceType, UniversalService, ListServicesResponse } from 'types/services.types';

interface ServiceOption {
  type: 'cluster' | 'service';
  id: string;
  label: string;
  serviceId?: string;
  cluster?: string;
}

interface RealTimeSelectionFormProps {
  onSuccess?: () => void;
}

export const RealTimeSelectionForm: FC<RealTimeSelectionFormProps> = ({
  onSuccess,
}) => {
  const { user } = useUser();
  const queryClient = useQueryClient();

  const [selectedServices, setSelectedServices] = useState<ServiceOption[]>([]);
  const [isOpen, setIsOpen] = useState(false);

  const canManageRTA = user?.isEditor || user?.isPMMAdmin;

  const { data: servicesData, isLoading: isLoadingServices } = useQuery<ListServicesResponse>({
    queryKey: ['services', ServiceType.mongodb],
    queryFn: async (): Promise<ListServicesResponse> => {
      const response = await listServices({});

      // Filter MongoDB services from the services array
      // API returns serviceType as lowercase "mongodb", not "SERVICE_TYPE_MONGODB_SERVICE"
      const mongodbServices = (response.services || []).filter(
        (service: UniversalService) =>
          service.serviceType === 'mongodb' ||
          service.serviceType === 'SERVICE_TYPE_MONGODB_SERVICE'
      );

      return { services: mongodbServices };
    },
  });

  // Fetch running agents to filter out services with active RTA
  const { data: runningAgentsData, isLoading: isLoadingAgents } = useRunningRealtimeAgents();

  const serviceOptions = useMemo<ServiceOption[]>(() => {
    // Determine services from new or legacy API format
    const services: UniversalService[] = servicesData?.services
      ? servicesData.services
      : servicesData?.mongodb
        ? (servicesData.mongodb as UniversalService[])
        : [];

    if (services.length === 0) {
      return [];
    }

    // Filter out services that already have running RTA agents
    const runningServiceIds = new Set(
      runningAgentsData?.agents?.map(agent => agent.serviceId) || []
    );

    const filteredServices = services.filter(
      (service: UniversalService) => !runningServiceIds.has(service.serviceId)
    );

    // Group services by cluster
    const clusterMap = new Map<string, UniversalService[]>();
    const standaloneServices: UniversalService[] = [];

    filteredServices.forEach((service) => {
      if (service.cluster) {
        if (!clusterMap.has(service.cluster)) {
          clusterMap.set(service.cluster, []);
        }
        const clusterServices = clusterMap.get(service.cluster);
        if (clusterServices) {
          clusterServices.push(service);
        }
      } else {
        standaloneServices.push(service);
      }
    });

    // Build options: standalone first, then clusters with their services
    const options: ServiceOption[] = [];

    // Add standalone services
    standaloneServices.forEach((service) => {
      options.push({
        type: 'service',
        id: service.serviceId,
        label: service.serviceName,
        serviceId: service.serviceId,
      });
    });

    // Add clusters and their services
    Array.from(clusterMap.entries())
      .sort(([a], [b]) => a.localeCompare(b))
      .forEach(([clusterName, clusterServices]) => {
        // Add cluster header as a selectable option
        options.push({
          type: 'cluster',
          id: `cluster-${clusterName}`,
          label: clusterName,
          cluster: clusterName,
        });

        // Add cluster services sorted by name
        clusterServices
          .sort((a, b) => a.serviceName.localeCompare(b.serviceName))
          .forEach((service) => {
            options.push({
              type: 'service',
              id: service.serviceId,
              label: service.serviceName,
              serviceId: service.serviceId,
              cluster: clusterName,
            });
          });
      });

    return options;
  }, [servicesData, runningAgentsData]);

  const startMutation = useMutation({
    mutationFn: async () => {
      // Filter out cluster headers, keep only real services
      const realServices = selectedServices.filter(s => s.type === 'service' && s.serviceId);

      if (realServices.length === 0) {
        throw new Error('Please select at least one service');
      }

      await Promise.all(
        realServices.map((service) =>
          changeRealtimeAnalytics({
            enable: true,
            serviceId: service.serviceId!,
          })
        )
      );
    },
    onSuccess: () => {
      enqueueSnackbar(Messages.startSuccess, { variant: 'success' });
      setSelectedServices([]);
      // Invalidate running agents cache to update dropdown
      queryClient.invalidateQueries({ queryKey: [REALTIME_AGENTS_QUERY_KEY] });
      onSuccess?.();
    },
    onError: (error) => {
      const message =
        error instanceof Error ? error.message : Messages.startError;
      enqueueSnackbar(message, { variant: 'error' });
    },
  });

  const handleServiceChange = (
    _event: React.SyntheticEvent,
    value: ServiceOption[]
  ) => {
    // Filter out cluster options - only keep real services
    const servicesOnly = value.filter((option) => option.type === 'service');
    setSelectedServices(servicesOnly);
  };

  const handleStart = () => {
    startMutation.mutate();
  };

  // Handle cluster checkbox click
  const handleClusterToggle = (clusterName: string) => {
    const servicesInCluster = serviceOptions.filter(
      (option) => option.type === 'service' && option.cluster === clusterName
    );

    const allSelected = servicesInCluster.every((service) =>
      selectedServices.some((selected) => selected.id === service.id)
    );

    if (allSelected) {
      // Deselect all services in this cluster
      setSelectedServices(
        selectedServices.filter(
          (selected) =>
            !servicesInCluster.some((service) => service.id === selected.id)
        )
      );
    } else {
      // Select all services in this cluster
      const newSelections = [...selectedServices];
      servicesInCluster.forEach((service) => {
        if (!newSelections.some((selected) => selected.id === service.id)) {
          newSelections.push(service);
        }
      });
      setSelectedServices(newSelections);
    }
  };

  // Check if all services in a cluster are selected
  const isClusterFullySelected = (clusterName: string): boolean => {
    const servicesInCluster = serviceOptions.filter(
      (option) => option.type === 'service' && option.cluster === clusterName
    );
    return servicesInCluster.every((service) =>
      selectedServices.some((selected) => selected.id === service.id)
    );
  };

  // Check if some (but not all) services in a cluster are selected
  const isClusterPartiallySelected = (clusterName: string): boolean => {
    const servicesInCluster = serviceOptions.filter(
      (option) => option.type === 'service' && option.cluster === clusterName
    );
    const selectedCount = servicesInCluster.filter((service) =>
      selectedServices.some((selected) => selected.id === service.id)
    ).length;
    return selectedCount > 0 && selectedCount < servicesInCluster.length;
  };

  return (
    <Stack gap="24px" sx={{ width: '100%' }}>
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
          <TextField
            {...params}
            label={selectedServices.length > 0 || isOpen ? 'Cluster/Service' : undefined}
            placeholder={selectedServices.length === 0 ? Messages.searchPlaceholder : ''}
            variant="outlined"
            InputProps={{
              ...params.InputProps,
              sx: {
                fontFamily: 'Roboto, sans-serif',
                fontSize: '16px',
                fontWeight: 400,
                lineHeight: 1.5,
                fontVariationSettings: "'wdth' 100",
              },
            }}
            InputLabelProps={{
              sx: (theme) => ({
                color: theme.palette.text.primary,
                fontFamily: 'Roboto',
                fontSize: '16px',
                fontStyle: 'normal',
                fontWeight: 500,
                lineHeight: '100%',
                letterSpacing: '0.12px',
                fontFeatureSettings: "'liga' off, 'clig' off",
                '&.Mui-focused': {
                  color: theme.palette.text.primary,
                },
                '&.MuiInputLabel-shrink': {
                  color: theme.palette.text.primary,
                  fontSize: '16px',
                },
              }),
            }}
            sx={(theme) => ({
              '& .MuiInputLabel-root': {
                color: theme.palette.text.primary,
                '&.Mui-focused': {
                  color: theme.palette.text.primary,
                },
              },
              '& .MuiOutlinedInput-root': {
                '& fieldset': {
                  borderWidth: '2px',
                  borderRadius: '4px',
                  borderColor: theme.palette.mode === 'dark'
                    ? 'rgba(255, 255, 255, 0.25)'
                    : 'rgba(0, 0, 0, 0.23)',
                },
                '&:hover fieldset': {
                  borderColor: theme.palette.mode === 'dark'
                    ? 'rgba(255, 255, 255, 0.35)'
                    : 'rgba(0, 0, 0, 0.4)',
                },
                '&.Mui-focused fieldset': {
                  borderColor: theme.palette.primary.main,
                  borderWidth: '2px',
                },
              },
            })}
          />
        )}
        renderTags={(value, getTagProps) =>
          value.slice(0, 2).map((option, index) => {
            const { key, ...tagProps } = getTagProps({ index });
            return (
              <Chip
                key={key}
                label={option.label}
                deleteIcon={
                  <Box
                    sx={(theme) => ({
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      width: '16px',
                      height: '16px',
                      borderRadius: '50%',
                      backgroundColor: theme.palette.mode === 'dark'
                        ? 'rgba(255, 255, 255, 0.2)'
                        : 'rgba(0, 0, 0, 0.15)',
                    })}
                  >
                    <CloseIcon
                      sx={(theme) => ({
                        fontSize: '12px',
                        color: theme.palette.mode === 'dark'
                          ? 'rgba(255, 255, 255, 0.9)'
                          : 'rgba(0, 0, 0, 0.6)',
                      })}
                    />
                  </Box>
                }
                {...tagProps}
                sx={(theme) => ({
                  height: '24px',
                  borderRadius: '12px',
                  backgroundColor: theme.palette.mode === 'dark'
                    ? 'rgba(255, 255, 255, 0.16)'
                    : 'rgba(0, 0, 0, 0.08)',
                  px: '8px',
                  gap: '4px',
                  '& .MuiChip-label': {
                    fontFamily: 'Roboto, sans-serif',
                    fontSize: '14px',
                    fontWeight: 400,
                    lineHeight: 1.4,
                    px: 0,
                    py: 0,
                  },
                  '& .MuiChip-deleteIcon': {
                    m: 0,
                    p: 0,
                  },
                })}
              />
            );
          })
        }
        renderOption={(props, option, { selected }) => {
          const { key, ...otherProps } = props;
          const isCluster = option.type === 'cluster';
          const isServiceInCluster = option.type === 'service' && Boolean(option.cluster);

          // For cluster options, check if all/some services are selected
          const isFullySelected = isCluster && isClusterFullySelected(option.label);
          const isPartiallySelected = isCluster && isClusterPartiallySelected(option.label);

          return (
            <li
              key={key}
              {...otherProps}
              onClick={
                isCluster
                  ? (e) => {
                      e.stopPropagation();
                      handleClusterToggle(option.label);
                    }
                  : otherProps.onClick
              }
              style={{
                ...otherProps.style,
                backgroundColor: 'transparent',
                minHeight: '40px',
                padding: '0 8px',
                paddingLeft: isServiceInCluster ? '40px' : '8px',
                position: 'relative',
              }}
            >
              <Checkbox
                checked={isCluster ? isFullySelected : selected}
                indeterminate={isCluster ? isPartiallySelected : false}
                size="small"
                sx={{ p: '8px', mr: -0.5 }}
                onClick={
                  isCluster
                    ? (e) => {
                        e.stopPropagation();
                        handleClusterToggle(option.label);
                      }
                    : undefined
                }
              />
              <Box
                sx={{
                  fontFamily: 'Roboto, sans-serif',
                  fontSize: '16px',
                  fontWeight: 400,
                  lineHeight: 1.375,
                  fontVariationSettings: "'wdth' 100",
                  flex: 1,
                  py: '9px',
                  px: '8px',
                }}
              >
                {option.label}
              </Box>
              {isServiceInCluster && (
                <Box
                  sx={{
                    position: 'absolute',
                    left: '28px',
                    top: 0,
                    bottom: 0,
                    width: '1px',
                    borderLeft: '1px solid',
                    borderColor: 'divider',
                  }}
                />
              )}
            </li>
          );
        }}
        loading={isLoadingServices || isLoadingAgents}
        disabled={!canManageRTA || serviceOptions.length === 0}
      />

      <Button
        variant="contained"
        size="large"
        onClick={handleStart}
        disabled={
          selectedServices.filter(s => s.type === 'service' && s.serviceId).length === 0 ||
          startMutation.isPending ||
          !canManageRTA
        }
        sx={{
          borderRadius: '999px',
          minHeight: '40px',
          maxHeight: '40px',
          fontSize: '15px',
          fontFamily: 'Poppins, sans-serif',
          fontWeight: 600,
          textTransform: 'none',
          lineHeight: 1.063,
          py: 0,
        }}
      >
        {Messages.startButton}
      </Button>
    </Stack>
  );
};
