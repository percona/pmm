import React from 'react';
import {
  Card,
  CardContent,
  CardActions,
  Typography,
  Chip,
  Stack,
  Box,
  IconButton,
  Tooltip,
  LinearProgress,
} from '@mui/material';
import {
  Dataset as DatabaseIcon,
  PlayArrow as PlayIcon,
  Stop as StopIcon,
  Settings as SettingsIcon,
  CheckCircle as CheckCircleIcon,
  Error as ErrorIcon,
  Schedule as ScheduleIcon,
} from '@mui/icons-material';
import { RealTimeServiceData } from 'types/realtime.types';

interface RealTimeServiceCardProps {
  service: RealTimeServiceData;
  isSelected: boolean;
  onSelect: () => void;
  onEnable?: () => void;
  onDisable?: () => void;
  onConfigure?: () => void;
}

export const RealTimeServiceCard: React.FC<RealTimeServiceCardProps> = ({
  service,
  isSelected,
  onSelect,
  onEnable,
  onDisable,
  onConfigure,
}) => {
  const getServiceIcon = (serviceType: string) => {
    switch (serviceType.toLowerCase()) {
      case 'mongodb':
        return <DatabaseIcon />;
      default:
        return <DatabaseIcon />;
    }
  };

  const getStatusColor = (isEnabled: boolean | undefined) => {
    return isEnabled ? 'success' : 'default';
  };

  const getStatusIcon = (isEnabled: boolean | undefined) => {
    return isEnabled ? <CheckCircleIcon /> : <ErrorIcon />;
  };

  const getLastSeenStatus = (lastSeen: string | undefined) => {
    if (!lastSeen) {
      return { color: 'error', text: 'Never' };
    }
    
    const lastSeenTime = new Date(lastSeen).getTime();
    const now = Date.now();
    const diffMinutes = (now - lastSeenTime) / (1000 * 60);
    
    if (diffMinutes < 1) {
      return { color: 'success', text: 'Just now' };
    } else if (diffMinutes < 5) {
      return { color: 'warning', text: `${Math.floor(diffMinutes)}m ago` };
    } else {
      return { color: 'error', text: `${Math.floor(diffMinutes)}m ago` };
    }
  };

  const lastSeenStatus = getLastSeenStatus(service.lastSeen);

  return (
    <Card
      sx={{
        cursor: 'pointer',
        border: isSelected ? 2 : 1,
        borderColor: isSelected ? 'primary.main' : 'divider',
        '&:hover': {
          boxShadow: 4,
        },
      }}
      onClick={onSelect}
    >
      <CardContent>
        <Stack spacing={2}>
          {/* Header */}
          <Box display="flex" alignItems="center" justifyContent="space-between">
            <Stack direction="row" alignItems="center" spacing={1}>
              {getServiceIcon(service.serviceType)}
              <Typography variant="h6" sx={{ ml: 1 }}>
                {service.serviceName}
              </Typography>
            </Stack>
            <Chip
              icon={getStatusIcon(service.isEnabled)}
              label={service.isEnabled ? 'Enabled' : 'Disabled'}
              color={getStatusColor(service.isEnabled)}
              size="small"
            />
          </Box>

          {/* Service Type */}
          <Box>
            <Typography variant="body2" color="text.secondary">
              Service Type
            </Typography>
            <Typography variant="body1" fontWeight="medium">
              {service.serviceType}
            </Typography>
          </Box>

          {/* Node Information */}
          <Box>
            <Typography variant="body2" color="text.secondary">
              Node
            </Typography>
            <Typography variant="body1" fontWeight="medium">
              {service.nodeName}
            </Typography>
          </Box>

          {/* Configuration */}
          {service.isEnabled && service.config && (
            <Box>
              <Typography variant="body2" color="text.secondary">
                Configuration
              </Typography>
              <Stack direction="row" spacing={1} mt={0.5}>
                <Chip
                  label={`${service.config.collectionIntervalSeconds}s interval`}
                  size="small"
                  variant="outlined"
                />
                {service.config.disableExamples && (
                  <Chip
                    label="Examples disabled"
                    size="small"
                    variant="outlined"
                  />
                )}
              </Stack>
            </Box>
          )}

          {/* Labels */}
          {Object.keys(service.labels).length > 0 && (
            <Box>
              <Typography variant="body2" color="text.secondary">
                Labels
              </Typography>
              <Stack direction="row" spacing={0.5} mt={0.5} flexWrap="wrap">
                {Object.entries(service.labels).slice(0, 3).map(([key, value]) => (
                  <Chip
                    key={`${key}-${value}`}
                    label={`${key}=${value}`}
                    size="small"
                    variant="outlined"
                  />
                ))}
                {Object.keys(service.labels).length > 3 && (
                  <Chip
                    label={`+${Object.keys(service.labels).length - 3} more`}
                    size="small"
                    variant="outlined"
                  />
                )}
              </Stack>
            </Box>
          )}

          {/* Last Seen */}
          <Box>
            <Typography variant="body2" color="text.secondary">
              Last Seen
            </Typography>
            <Stack direction="row" alignItems="center" spacing={0.5}>
              <ScheduleIcon fontSize="small" />
              <Typography
                variant="body2"
                color={`${lastSeenStatus.color}.main`}
                fontWeight="medium"
              >
                {lastSeenStatus.text}
              </Typography>
            </Stack>
          </Box>

          {/* Health Indicator */}
          <Box>
            <Typography variant="body2" color="text.secondary" gutterBottom>
              Health Status
            </Typography>
            <LinearProgress
              variant="determinate"
              value={service.isEnabled ? 100 : 0}
              color={service.isEnabled ? 'success' : 'error'}
              sx={{ height: 6, borderRadius: 3 }}
            />
          </Box>
        </Stack>
      </CardContent>

      <CardActions>
        <Stack direction="row" spacing={1} sx={{ ml: 'auto' }}>
          {service.isEnabled ? (
            <>
              <Tooltip title="Disable Real-Time Analytics">
                <IconButton
                  size="small"
                  color="error"
                  onClick={(e) => {
                    e.stopPropagation();
                    onDisable?.();
                  }}
                >
                  <StopIcon />
                </IconButton>
              </Tooltip>
              <Tooltip title="Configure">
                <IconButton
                  size="small"
                  onClick={(e) => {
                    e.stopPropagation();
                    onConfigure?.();
                  }}
                >
                  <SettingsIcon />
                </IconButton>
              </Tooltip>
            </>
          ) : (
            <Tooltip title="Enable Real-Time Analytics">
              <IconButton
                size="small"
                color="primary"
                onClick={(e) => {
                  e.stopPropagation();
                  onEnable?.();
                }}
              >
                <PlayIcon />
              </IconButton>
            </Tooltip>
          )}
        </Stack>
      </CardActions>
    </Card>
  );
};
