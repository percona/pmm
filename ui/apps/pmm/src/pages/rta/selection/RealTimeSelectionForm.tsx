import { FC, useState, useMemo } from 'react';
import {
  Autocomplete,
  Box,
  Button,
  Checkbox,
  Chip,
  Stack,
  TextField,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import { useQuery, useMutation } from '@tanstack/react-query';
import { enqueueSnackbar } from 'notistack';
import { useUser } from 'contexts/user';
import { Messages } from './RealTimeSelection.messages';
import { changeRealtimeAnalytics, listRunningRealtimeAgents } from 'api/realtime';
import { listServices } from 'api/services';
import { ServiceType, UniversalService, ListServicesResponse } from 'types/services.types';
import { mockMongoDBServices } from './RealTimeSelection.mocks';

interface ServiceOption {
  type: 'cluster' | 'service';
  id: string;
  label: string;
  serviceId?: string;
  cluster?: string;
}

interface RealTimeSelectionFormProps {
  onSuccess?: () => void;
  useMockData?: boolean;
}

export const RealTimeSelectionForm: FC<RealTimeSelectionFormProps> = ({
  onSuccess,
  useMockData = false,
}) => {
  const { user } = useUser();

  const [selectedServices, setSelectedServices] = useState<ServiceOption[]>([]);
  const [isOpen, setIsOpen] = useState<boolean>(false);

  const canManageRTA = user?.isEditor || user?.isPMMAdmin;

  const { data: servicesData, isLoading: isLoadingServices } = useQuery<ListServicesResponse>({
    queryKey: ['services', ServiceType.mongodb],
    queryFn: useMockData
      ? async (): Promise<ListServicesResponse> => ({ mongodb: mockMongoDBServices })
      : async (): Promise<ListServicesResponse> => {
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

  // Fetch running agents for viewers to filter services
  const { data: runningAgentsData, isLoading: isLoadingAgents } = useQuery({
    queryKey: ['runningRealtimeAgents'],
    queryFn: () => listRunningRealtimeAgents(),
    enabled: !canManageRTA, // Only fetch for viewers
  });

  const serviceOptions = useMemo<ServiceOption[]>(() => {
    let services: UniversalService[] = [];

    // Handle new API format (services array)
    if (servicesData?.services) {
      services = servicesData.services;
    }
    // Handle legacy format (mongodb array) - for backward compatibility
    else if (servicesData?.mongodb) {
      services = servicesData.mongodb as UniversalService[];
    } else {
      return [];
    }

    // Filter by permissions:
    // - Admins (isEditor or isPMMAdmin) see all services
    // - Viewers see only services with running RTA agents
    const filteredServices = canManageRTA
      ? services
      : services.filter((service: UniversalService) =>
          runningAgentsData?.agents?.some(agent => agent.serviceId === service.serviceId)
        );

    const options = filteredServices.map((service: UniversalService) => ({
      type: 'service' as const,
      id: service.serviceId,
      label: service.serviceName,
      serviceId: service.serviceId,
      cluster: service.cluster,
    }));

    // Sort: standalone services first (no cluster), then grouped by cluster
    return options.sort((a, b) => {
      // Standalone services first
      if (!a.cluster && b.cluster) return -1;
      if (a.cluster && !b.cluster) return 1;
      // Then sort by cluster name
      if (a.cluster !== b.cluster) {
        return (a.cluster || '').localeCompare(b.cluster || '');
      }
      // Within same cluster/standalone, sort by service name
      return a.label.localeCompare(b.label);
    });
  }, [servicesData, canManageRTA, runningAgentsData]);

  const startMutation = useMutation({
    mutationFn: async () => {
      if (selectedServices.length === 0) {
        throw new Error('Please select at least one service');
      }

      // Mock mode: simulate successful API call
      if (useMockData) {
        await new Promise(resolve => setTimeout(resolve, 1000));
        return;
      }

      // Real API calls
      await Promise.all(
        selectedServices.map((service) =>
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
      onSuccess?.();
    },
    onError: (error) => {
      const message =
        error instanceof Error ? error.message : Messages.startError;
      enqueueSnackbar(message, { variant: 'error' });
    },
  });

  const handleServiceChange = (_: unknown, value: ServiceOption[]) => {
    setSelectedServices(value);
  };

  const handleStart = () => {
    startMutation.mutate();
  };

  // Handle cluster checkbox click
  const handleClusterToggle = (clusterName: string) => {
    const servicesInCluster = serviceOptions.filter(
      (option) => option.cluster === clusterName
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
      (option) => option.cluster === clusterName
    );
    return servicesInCluster.every((service) =>
      selectedServices.some((selected) => selected.id === service.id)
    );
  };

  // Check if some (but not all) services in a cluster are selected
  const isClusterPartiallySelected = (clusterName: string): boolean => {
    const servicesInCluster = serviceOptions.filter(
      (option) => option.cluster === clusterName
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
        groupBy={(option) => option.cluster || 'Standalone Services'}
        renderInput={(params) => (
          <TextField
            {...params}
            label={selectedServices.length > 0 || isOpen ? 'Cluster/Service' : undefined}
            placeholder={Messages.searchPlaceholder}
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
        renderGroup={(params) => {
          const isCluster = params.group !== 'Standalone Services';
          const isFullySelected = isCluster && isClusterFullySelected(params.group);
          const isPartiallySelected = isCluster && isClusterPartiallySelected(params.group);

          return (
            <li key={params.key}>
              <Box
                component="div"
                onClick={
                  isCluster
                    ? (e) => {
                        e.stopPropagation();
                        handleClusterToggle(params.group);
                      }
                    : undefined
                }
                sx={(theme) => ({
                  position: 'sticky',
                  top: '-8px',
                  display: 'flex',
                  alignItems: 'center',
                  px: '16px',
                  py: '8px',
                  fontFamily: 'Roboto, sans-serif',
                  fontSize: '14px',
                  fontWeight: 500,
                  lineHeight: 1.375,
                  color: theme.palette.text.secondary,
                  backgroundColor: 'transparent',
                  borderBottom: `1px solid ${theme.palette.divider}`,
                  zIndex: 1,
                  cursor: isCluster ? 'pointer' : 'default',
                  '&:hover': isCluster
                    ? {
                        backgroundColor:
                          theme.palette.mode === 'dark'
                            ? 'rgba(255, 255, 255, 0.03)'
                            : 'rgba(0, 0, 0, 0.02)',
                      }
                    : {},
                })}
              >
                {isCluster && (
                  <Checkbox
                    checked={isFullySelected}
                    indeterminate={isPartiallySelected}
                    size="small"
                    sx={{ p: '4px', mr: 0.5 }}
                    onClick={(e) => {
                      e.stopPropagation();
                      handleClusterToggle(params.group);
                    }}
                  />
                )}
                {params.group}
              </Box>
              <ul style={{ padding: 0 }}>{params.children}</ul>
            </li>
          );
        }}
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
          const hasCluster = Boolean(option.cluster);

          return (
            <li
              key={key}
              {...otherProps}
              style={{
                ...otherProps.style,
                backgroundColor: 'transparent',
                minHeight: '40px',
                padding: '0 8px',
                paddingLeft: hasCluster ? '32px' : '8px',
                position: 'relative',
              }}
            >
              <Checkbox
                checked={selected}
                size="small"
                sx={{ p: '8px', mr: -0.5 }}
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
              {hasCluster && (
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
        disabled={selectedServices.length === 0 || startMutation.isPending || !canManageRTA}
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
