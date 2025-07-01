import React from 'react';
import {
  Tooltip,
  Chip,
  Box,
  Typography,
  List,
  ListItem,
  ListItemText,
  Divider
} from '@mui/material';
import WarningIcon from '@mui/icons-material/Warning';
import ErrorIcon from '@mui/icons-material/Error';
import ReportProblemIcon from '@mui/icons-material/ReportProblem';
import InfoIcon from '@mui/icons-material/Info';
import { 
  AnomalyDetectionResult, 
  AnomalySeverity 
} from '../../utils/queryAnomalyDetection';

interface AnomalyWarningIconProps {
  result: AnomalyDetectionResult;
  onAnalyzeClick?: (result: AnomalyDetectionResult) => void;
  variant?: 'icon' | 'chip' | 'detailed';
}

export const AnomalyWarningIcon: React.FC<AnomalyWarningIconProps> = ({
  result,
  onAnalyzeClick,
  variant = 'icon'
}) => {
  if (!result.hasAnomalies) {
    return null;
  }

  const getSeverityConfig = (severity: AnomalySeverity) => {
    switch (severity) {
      case AnomalySeverity.CRITICAL:
        return {
          icon: <ErrorIcon fontSize="small" />,
          color: 'error' as const,
          label: 'Critical',
          bgColor: '#ffebee'
        };
      case AnomalySeverity.HIGH:
        return {
          icon: <ReportProblemIcon fontSize="small" />,
          color: 'warning' as const,
          label: 'High',
          bgColor: '#fff3e0'
        };
      case AnomalySeverity.MEDIUM:
        return {
          icon: <WarningIcon fontSize="small" />,
          color: 'warning' as const,
          label: 'Medium',
          bgColor: '#fffde7'
        };
      case AnomalySeverity.LOW:
        return {
          icon: <InfoIcon fontSize="small" />,
          color: 'info' as const,
          label: 'Low',
          bgColor: '#e3f2fd'
        };
      default:
        return {
          icon: <InfoIcon fontSize="small" />,
          color: 'default' as const,
          label: 'Unknown',
          bgColor: '#f5f5f5'
        };
    }
  };

  const config = getSeverityConfig(result.overallSeverity);

  const renderTooltipContent = () => (
    <Box sx={{ maxWidth: 400, p: 1 }}>
      <Typography variant="subtitle2" sx={{ fontWeight: 'bold', mb: 1 }}>
        Performance Anomalies Detected ({config.label} Severity)
      </Typography>
      
      <List dense sx={{ py: 0 }}>
        {result.anomalies.map((anomaly, index) => (
          <React.Fragment key={index}>
            <ListItem sx={{ px: 0, py: 0.5 }}>
              <ListItemText
                primary={
                  <Typography variant="body2" sx={{ fontWeight: 'medium' }}>
                    {anomaly.description}
                  </Typography>
                }
                secondary={
                  <Typography variant="caption" color="textSecondary">
                    💡 {anomaly.recommendation}
                  </Typography>
                }
              />
            </ListItem>
            {index < result.anomalies.length - 1 && <Divider />}
          </React.Fragment>
        ))}
      </List>

      {onAnalyzeClick && result.aiAnalysisPrompt && (
        <>
          <Divider sx={{ my: 1 }} />
          <Typography 
            variant="caption" 
            sx={{ 
              color: 'primary.main', 
              cursor: 'pointer',
              textDecoration: 'underline',
              display: 'block',
              textAlign: 'center',
              '&:hover': { fontWeight: 'bold' }
            }}
            onClick={(e) => {
              e.stopPropagation();
              onAnalyzeClick(result);
            }}
          >
            🤖 Get AI Analysis & Recommendations
          </Typography>
        </>
      )}
    </Box>
  );

  if (variant === 'chip') {
    return (
      <Tooltip title={renderTooltipContent()} arrow placement="top">
        <Chip
          icon={config.icon}
          label={`${result.anomalies.length} Issue${result.anomalies.length > 1 ? 's' : ''}`}
          color={config.color}
          size="small"
          variant="filled"
          sx={{ 
            cursor: 'pointer',
            '&:hover': { opacity: 0.8 }
          }}
        />
      </Tooltip>
    );
  }

  if (variant === 'detailed') {
    return (
      <Box 
        sx={{ 
          p: 2, 
          border: `1px solid ${config.color === 'error' ? '#f44336' : '#ff9800'}`,
          borderRadius: 1,
          backgroundColor: config.bgColor,
          mb: 2
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
          {config.icon}
          <Typography variant="subtitle2" sx={{ ml: 1, fontWeight: 'bold' }}>
            Performance Anomalies Detected ({config.label} Severity)
          </Typography>
        </Box>
        
        <List dense>
          {result.anomalies.slice(0, 3).map((anomaly, index) => (
            <ListItem key={index} sx={{ px: 0, py: 0.5 }}>
              <ListItemText
                primary={
                  <Typography variant="body2">
                    • {anomaly.description}
                  </Typography>
                }
                secondary={
                  <Typography variant="caption" color="textSecondary" sx={{ ml: 2 }}>
                    💡 {anomaly.recommendation}
                  </Typography>
                }
              />
            </ListItem>
          ))}
          {result.anomalies.length > 3 && (
            <Typography variant="caption" color="textSecondary">
              ... and {result.anomalies.length - 3} more issues
            </Typography>
          )}
        </List>

        {onAnalyzeClick && result.aiAnalysisPrompt && (
          <Box sx={{ mt: 1, textAlign: 'center' }}>
            <Typography 
              variant="body2" 
              sx={{ 
                color: 'primary.main', 
                cursor: 'pointer',
                textDecoration: 'underline',
                '&:hover': { fontWeight: 'bold' }
              }}
              onClick={() => onAnalyzeClick(result)}
            >
              🤖 Get AI Analysis & Recommendations
            </Typography>
          </Box>
        )}
      </Box>
    );
  }

  // Default: icon variant
  return (
    <Tooltip title={renderTooltipContent()} arrow placement="top">
      <Box 
        sx={{ 
          display: 'inline-flex',
          alignItems: 'center',
          cursor: 'pointer',
          color: config.color === 'error' ? '#d32f2f' : 
                 config.color === 'warning' ? '#ed6c02' : '#0288d1',
          '&:hover': { opacity: 0.7 }
        }}
      >
        {config.icon}
      </Box>
    </Tooltip>
  );
};

export default AnomalyWarningIcon; 