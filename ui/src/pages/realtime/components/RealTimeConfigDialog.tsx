import React, { useState } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  FormControlLabel,
  Switch,
  Stack,
  Typography,
  Box,
  Divider,
  Alert,
  CircularProgress,
} from '@mui/material';
import { useEnableRealTimeAnalytics, useDisableRealTimeAnalytics, useUpdateRealTimeConfig } from 'hooks/api/useRealtime';
import { RealTimeServiceData, RealTimeConfig } from 'types/realtime.types';

interface RealTimeConfigDialogProps {
  open: boolean;
  onClose: () => void;
  services: RealTimeServiceData[];
}

export const RealTimeConfigDialog: React.FC<RealTimeConfigDialogProps> = ({
  open,
  onClose,
  services,
}) => {
  const [selectedServiceId, setSelectedServiceId] = useState<string>('');
  const [config, setConfig] = useState<RealTimeConfig>({
    collectionIntervalSeconds: 2,
    disableExamples: false,
  });

  const enableMutation = useEnableRealTimeAnalytics();
  const disableMutation = useDisableRealTimeAnalytics();
  const updateMutation = useUpdateRealTimeConfig();

  const selectedService = services.find(s => s.serviceId === selectedServiceId);

  const handleServiceChange = (serviceId: string) => {
    setSelectedServiceId(serviceId);
    const service = services.find(s => s.serviceId === serviceId);
    if (service && service.config) {
      setConfig(service.config);
    }
  };

  const handleConfigChange = (field: keyof RealTimeConfig, value: any) => {
    setConfig(prev => ({
      ...prev,
      [field]: value,
    }));
  };

  const handleEnable = async () => {
    if (!selectedServiceId) return;
    
    try {
      await enableMutation.mutateAsync({
        serviceId: selectedServiceId,
        config,
      });
      onClose();
    } catch (error) {
      console.error('Error enabling real-time analytics:', error);
    }
  };

  const handleDisable = async () => {
    if (!selectedServiceId) return;
    
    try {
      await disableMutation.mutateAsync({
        serviceId: selectedServiceId,
      });
      onClose();
    } catch (error) {
      console.error('Error disabling real-time analytics:', error);
    }
  };

  const handleUpdate = async () => {
    if (!selectedServiceId) return;
    
    try {
      await updateMutation.mutateAsync({
        serviceId: selectedServiceId,
        config,
      });
      onClose();
    } catch (error) {
      console.error('Error updating configuration:', error);
    }
  };

  const isLoading = enableMutation.isPending || disableMutation.isPending || updateMutation.isPending;

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>Real-Time Analytics Configuration</DialogTitle>
      <DialogContent>
        <Stack spacing={3} sx={{ mt: 1 }}>
          {/* Service Selection */}
          <Box>
            <Typography variant="subtitle1" gutterBottom>
              Select Service
            </Typography>
            <TextField
              select
              fullWidth
              value={selectedServiceId}
              onChange={(e) => handleServiceChange(e.target.value)}
              SelectProps={{
                native: true,
              }}
            >
              <option value="">Choose a service...</option>
              {services.map(service => (
                <option key={service.serviceId} value={service.serviceId}>
                  {service.serviceName} ({service.serviceType})
                </option>
              ))}
            </TextField>
          </Box>

          {selectedService && (
            <>
              <Divider />
              
              {/* Service Info */}
              <Box>
                <Typography variant="subtitle2" gutterBottom>
                  Service Information
                </Typography>
                <Stack spacing={1}>
                  <Box display="flex" justifyContent="space-between">
                    <Typography variant="body2" color="text.secondary">
                      Service Name:
                    </Typography>
                    <Typography variant="body2">{selectedService.serviceName}</Typography>
                  </Box>
                  <Box display="flex" justifyContent="space-between">
                    <Typography variant="body2" color="text.secondary">
                      Service Type:
                    </Typography>
                    <Typography variant="body2">{selectedService.serviceType}</Typography>
                  </Box>
                  <Box display="flex" justifyContent="space-between">
                    <Typography variant="body2" color="text.secondary">
                      Node:
                    </Typography>
                    <Typography variant="body2">{selectedService.nodeName}</Typography>
                  </Box>
                  <Box display="flex" justifyContent="space-between">
                    <Typography variant="body2" color="text.secondary">
                      Status:
                    </Typography>
                    <Typography 
                      variant="body2" 
                      color={selectedService.isEnabled ? 'success.main' : 'error.main'}
                    >
                      {selectedService.isEnabled ? 'Enabled' : 'Disabled'}
                    </Typography>
                  </Box>
                </Stack>
              </Box>

              <Divider />

              {/* Configuration */}
              <Box>
                <Typography variant="subtitle2" gutterBottom>
                  Configuration
                </Typography>
                <Stack spacing={2}>
                  <TextField
                    label="Collection Interval (seconds)"
                    type="number"
                    value={config.collectionIntervalSeconds}
                    onChange={(e) => handleConfigChange('collectionIntervalSeconds', parseInt(e.target.value) || 2)}
                    inputProps={{ min: 1, max: 60 }}
                    helperText="How often to collect real-time query data (1-60 seconds)"
                    fullWidth
                  />
                  
                  <FormControlLabel
                    control={
                      <Switch
                        checked={config.disableExamples}
                        onChange={(e) => handleConfigChange('disableExamples', e.target.checked)}
                      />
                    }
                    label="Disable Query Examples"
                  />
                  <Typography variant="caption" color="text.secondary">
                    When enabled, query text examples will not be collected for privacy
                  </Typography>
                </Stack>
              </Box>

              {/* Error Display */}
              {(enableMutation.error || disableMutation.error || updateMutation.error) && (
                <Alert severity="error">
                  {enableMutation.error?.message || 
                   disableMutation.error?.message || 
                   updateMutation.error?.message || 
                   'An error occurred'}
                </Alert>
              )}
            </>
          )}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={isLoading}>
          Cancel
        </Button>
        {selectedService && (
          <>
            {selectedService.isEnabled ? (
              <>
                <Button
                  onClick={handleUpdate}
                  variant="contained"
                  disabled={isLoading}
                  startIcon={isLoading ? <CircularProgress size={16} /> : undefined}
                >
                  Update Configuration
                </Button>
                <Button
                  onClick={handleDisable}
                  color="error"
                  disabled={isLoading}
                  startIcon={isLoading ? <CircularProgress size={16} /> : undefined}
                >
                  Disable
                </Button>
              </>
            ) : (
              <Button
                onClick={handleEnable}
                variant="contained"
                disabled={isLoading}
                startIcon={isLoading ? <CircularProgress size={16} /> : undefined}
              >
                Enable Real-Time Analytics
              </Button>
            )}
          </>
        )}
      </DialogActions>
    </Dialog>
  );
};
