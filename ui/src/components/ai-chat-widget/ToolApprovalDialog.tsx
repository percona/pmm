import React from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Typography,
  List,
  ListItem,
  ListItemText,
  Chip,
  Box,
  Paper,
  Divider,
} from '@mui/material';
import { Build as ToolIcon, Warning as WarningIcon } from '@mui/icons-material';
import { ToolCall } from '../../api/aichat';

interface ToolApprovalDialogProps {
  open: boolean;
  toolCalls: ToolCall[];
  requestId: string;
  onApprove: (requestId: string, approvedIds?: string[]) => void;
  onDeny: (requestId: string) => void;
}

export const ToolApprovalDialog: React.FC<ToolApprovalDialogProps> = ({
  open,
  toolCalls,
  requestId,
  onApprove,
  onDeny,
}) => {
  const handleApproveAll = () => {
    onApprove(requestId);
  };

  const handleDeny = () => {
    onDeny(requestId);
  };

  const formatArguments = (args: string) => {
    try {
      const parsed = JSON.parse(args);
      return JSON.stringify(parsed, null, 2);
    } catch {
      return args;
    }
  };

  return (
    <Dialog 
      open={open} 
      maxWidth="md" 
      fullWidth
      PaperProps={{
        sx: { 
          borderRadius: 2,
          maxHeight: '80vh'
        }
      }}
    >
      <DialogTitle sx={{ 
        display: 'flex', 
        alignItems: 'center', 
        gap: 1,
        borderBottom: 1,
        borderColor: 'divider',
        pb: 2
      }}>
        <WarningIcon color="warning" />
        <Typography variant="h6">
          Tool Execution Request
        </Typography>
      </DialogTitle>
      
      <DialogContent sx={{ pt: 2 }}>
        <Typography variant="body1" sx={{ mb: 2 }}>
          The AI assistant wants to execute the following tool(s). Please review and approve or deny the request:
        </Typography>
        
        <List sx={{ width: '100%' }}>
          {toolCalls.map((toolCall, index) => (
            <React.Fragment key={toolCall.id}>
              {index > 0 && <Divider sx={{ my: 1 }} />}
              <ListItem sx={{ px: 0, py: 1 }}>
                <Paper 
                  variant="outlined" 
                  sx={{ 
                    width: '100%', 
                    p: 2,
                    backgroundColor: 'background.paper'
                  }}
                >
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                    <ToolIcon color="primary" fontSize="small" />
                    <Typography variant="h6" component="div">
                      {toolCall.function.name}
                    </Typography>
                    <Chip 
                      label={toolCall.type} 
                      size="small" 
                      variant="outlined"
                      color="primary"
                    />
                  </Box>
                  
                  <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
                    Arguments:
                  </Typography>
                  
                  <Paper 
                    variant="outlined" 
                    sx={{ 
                      p: 1, 
                      backgroundColor: 'grey.50',
                      border: '1px solid',
                      borderColor: 'grey.300'
                    }}
                  >
                    <Typography 
                      variant="body2" 
                      component="pre"
                      sx={{ 
                        fontFamily: 'monospace',
                        fontSize: '0.75rem',
                        whiteSpace: 'pre-wrap',
                        wordBreak: 'break-word',
                        margin: 0
                      }}
                    >
                      {formatArguments(toolCall.function.arguments)}
                    </Typography>
                  </Paper>
                </Paper>
              </ListItem>
            </React.Fragment>
          ))}
        </List>
        
        <Box sx={{ 
          mt: 2, 
          p: 2, 
          backgroundColor: 'warning.light', 
          borderRadius: 1,
          border: '1px solid',
          borderColor: 'warning.main'
        }}>
          <Typography variant="body2" color="warning.dark">
            <strong>Security Notice:</strong> Only approve tools if you trust the source and understand what they will do. 
            Tool execution may access your system, databases, or external services.
          </Typography>
        </Box>
      </DialogContent>
      
      <DialogActions sx={{ 
        px: 3, 
        pb: 2,
        borderTop: 1,
        borderColor: 'divider',
        gap: 1
      }}>
        <Button 
          onClick={handleDeny} 
          color="error"
          variant="outlined"
        >
          Deny
        </Button>
        <Button 
          onClick={handleApproveAll} 
          color="primary"
          variant="contained"
          startIcon={<ToolIcon />}
        >
          Approve All Tools
        </Button>
      </DialogActions>
    </Dialog>
  );
}; 