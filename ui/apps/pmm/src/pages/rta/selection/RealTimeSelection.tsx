import { FC, useState, useMemo } from 'react';
import {
  Autocomplete,
  Box,
  Button,
  Checkbox,
  Chip,
  Link,
  Stack,
  TextField,
  Typography,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import { useQuery, useMutation } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { enqueueSnackbar } from 'notistack';
import { Page } from 'components/page';
import { useUser } from 'contexts/user';
import { Messages } from './RealTimeSelection.messages';
import { changeRealtimeAnalytics } from 'api/realtime';
import { listServices } from 'api/services';
import { ServiceType } from 'types/services.types';
import { mockMongoDBServices } from './RealTimeSelection.mocks';

interface ServiceOption {
  type: 'cluster' | 'service';
  id: string;
  label: string;
  serviceId?: string;
  cluster?: string;
}

// Set to true to use mock data for development/testing
const USE_MOCK_DATA = import.meta.env.VITE_USE_MOCK_DATA === 'true';

export const RealTimeSelection: FC = () => {
  const { user } = useUser();
  const navigate = useNavigate();

  const [selectedServices, setSelectedServices] = useState<ServiceOption[]>([]);
  const [isOpen, setIsOpen] = useState<boolean>(false);

  const canManageRTA = user?.isEditor || user?.isPMMAdmin;

  const { data: servicesData, isLoading: isLoadingServices } = useQuery({
    queryKey: ['services', ServiceType.mongodb],
    queryFn: USE_MOCK_DATA
      ? async () => ({ mongodb: mockMongoDBServices })
      : () => listServices({ serviceType: ServiceType.mongodb }),
  });

  const serviceOptions = useMemo<ServiceOption[]>(() => {
    if (!servicesData?.mongodb) {
      return [];
    }

    // Simply map all services to options - no separate cluster headers
    return servicesData.mongodb.map((service) => ({
      type: 'service' as const,
      id: service.serviceId,
      label: service.serviceName,
      serviceId: service.serviceId,
      cluster: service.cluster,
    }));
  }, [servicesData]);

  const startMutation = useMutation({
    mutationFn: async () => {
      if (selectedServices.length === 0) {
        throw new Error('Please select at least one service');
      }

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
      // TODO: Navigate to RTA agents list page when it's implemented
      // navigate('/rta/agents');
      setSelectedServices([]);
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

  return (
    <Page footer={<></>}>
      <Stack
        gap={2}
        sx={{
          maxWidth: 392,
          mx: 'auto',
          py: 6,
          px: 2,
          alignItems: 'center',
          textAlign: 'center',
        }}
      >
        {/* Intro section */}
        <Stack gap={1} sx={{ width: '100%' }}>
          <Typography
            variant="h5"
            sx={{
              fontFamily: 'Poppins, sans-serif',
              fontWeight: 600,
              fontSize: '23px',
              lineHeight: 1.125,
              textAlign: 'center',
            }}
          >
            {Messages.title}
          </Typography>
          <Typography
            variant="body1"
            sx={{
              fontFamily: 'Roboto, sans-serif',
              fontWeight: 400,
              fontSize: '16px',
              lineHeight: 1.375,
              textAlign: 'center',
              fontVariationSettings: "'wdth' 100",
            }}
          >
            {Messages.description}
          </Typography>
        </Stack>

        {/* Form section */}
        <Stack gap={3} sx={{ width: '100%', py: 2 }}>
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
                      borderWidth: '1px',
                      borderRadius: '4px',
                      borderColor: theme.palette.mode === 'dark'
                        ? 'rgba(255, 255, 255, 0.3)'
                        : 'rgba(0, 0, 0, 0.23)',
                    },
                    '&:hover fieldset': {
                      borderColor: theme.palette.mode === 'dark'
                        ? 'rgba(255, 255, 255, 0.5)'
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
              value.map((option, index) => {
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
                          width: '18px',
                          height: '18px',
                          borderRadius: '50%',
                          backgroundColor: theme.palette.mode === 'dark'
                            ? 'rgba(255, 255, 255, 0.9)'
                            : 'rgba(0, 0, 0, 0.1)',
                        })}
                      >
                        <CloseIcon
                          sx={(theme) => ({
                            fontSize: '14px',
                            color: theme.palette.mode === 'dark'
                              ? 'rgba(0, 0, 0, 0.6)'
                              : 'rgba(0, 0, 0, 0.54)',
                          })}
                        />
                      </Box>
                    }
                    {...tagProps}
                    sx={(theme) => ({
                      height: 'auto',
                      borderRadius: '100px',
                      backgroundColor: theme.palette.mode === 'dark'
                        ? 'rgba(255, 255, 255, 0.16)'
                        : 'rgba(0, 0, 0, 0.08)',
                      px: '5px',
                      py: 0,
                      m: 0,
                      '& .MuiChip-label': {
                        fontFamily: 'Roboto, sans-serif',
                        fontSize: '12px',
                        fontWeight: 450,
                        lineHeight: 1.25,
                        letterSpacing: '0.12px',
                        fontVariationSettings: "'wdth' 100",
                        px: '5px',
                        pt: '4px',
                        pb: '5px',
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
            loading={isLoadingServices}
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
              fontSize: '15px',
              fontFamily: 'Poppins, sans-serif',
              fontWeight: 600,
              textTransform: 'none',
              lineHeight: 1.063,
            }}
          >
            {Messages.startButton}
          </Button>
        </Stack>

        {/* Footer section */}
        <Stack gap={1} sx={{ width: '100%' }}>
          <Typography
            variant="body2"
            color="text.secondary"
            sx={{
              fontFamily: 'Roboto, sans-serif',
              fontWeight: 400,
              fontSize: '14px',
              lineHeight: 1.5,
              textAlign: 'center',
              fontVariationSettings: "'wdth' 100",
            }}
          >
            {Messages.mongoOnly}
          </Typography>
          <Stack direction="row" gap={2} justifyContent="center">
            <Link
              href="https://docs.percona.com/percona-monitoring-and-management/3/get-started/query-analytics.html"
              target="_blank"
              sx={(theme) => ({
                fontFamily: 'Roboto, sans-serif',
                fontSize: '14px',
                fontWeight: 400,
                lineHeight: 1.5,
                color: theme.palette.info.light,
                textAlign: 'center',
                textDecoration: 'underline solid',
                textDecorationSkipInk: 'none',
                textUnderlinePosition: 'from-font',
                fontVariationSettings: "'wdth' 100",
                '&:hover': {
                  color: theme.palette.info.main,
                },
              })}
            >
              {Messages.documentation}
            </Link>
            <Link
              href="https://forums.percona.com/c/percona-monitoring-and-management-pmm/percona-monitoring-and-management-pmm-v3"
              target="_blank"
              sx={(theme) => ({
                fontFamily: 'Roboto, sans-serif',
                fontSize: '14px',
                fontWeight: 400,
                lineHeight: 1.5,
                color: theme.palette.info.light,
                textAlign: 'center',
                textDecoration: 'underline solid',
                textDecorationSkipInk: 'none',
                textUnderlinePosition: 'from-font',
                fontVariationSettings: "'wdth' 100",
                '&:hover': {
                  color: theme.palette.info.main,
                },
              })}
            >
              {Messages.feedback}
            </Link>
          </Stack>
        </Stack>
      </Stack>
    </Page>
  );
};
