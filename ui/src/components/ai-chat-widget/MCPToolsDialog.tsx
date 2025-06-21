import React from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  Typography,
  Chip,
  Box,
} from '@mui/material';
import {
  Build as ToolIcon,
} from '@mui/icons-material';
import { MCPTool } from '../../api/aichat';

interface MCPToolsDialogProps {
  open: boolean;
  onClose: () => void;
  tools: MCPTool[];
}

export const MCPToolsDialog: React.FC<MCPToolsDialogProps> = ({
  open,
  onClose,
  tools,
}) => {
  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>
        Available MCP Tools
        <Typography variant="subtitle2" color="textSecondary">
          {tools.length} tool{tools.length !== 1 ? 's' : ''} connected
        </Typography>
      </DialogTitle>
      
      <DialogContent>
        {tools.length === 0 ? (
          <Box sx={{ textAlign: 'center', py: 4 }}>
            <ToolIcon sx={{ fontSize: 48, color: 'text.secondary', mb: 2 }} />
            <Typography variant="h6" color="textSecondary" gutterBottom>
              No MCP Tools Available
            </Typography>
            <Typography variant="body2" color="textSecondary">
              Connect MCP servers to enable tool functionality
            </Typography>
          </Box>
        ) : (
          <List>
            {tools.map((tool, index) => (
              <ListItem key={`${tool.name}-${index}`} divider>
                <ListItemIcon>
                  <ToolIcon />
                </ListItemIcon>
                <ListItemText
                  primary={
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <Typography variant="subtitle1" component="span">
                        {tool.name}
                      </Typography>
                      <Chip label="MCP" size="small" variant="outlined" />
                    </Box>
                  }
                  secondary={
                    <Box>
                      <Typography variant="body2" color="textSecondary" sx={{ mb: 1 }}>
                        {tool.description}
                      </Typography>
                      {tool.input_schema && (
                        <Typography variant="caption" color="textSecondary">
                          Parameters: {Object.keys(tool.input_schema.properties || {}).join(', ') || 'None'}
                        </Typography>
                      )}
                    </Box>
                  }
                />
              </ListItem>
            ))}
          </List>
        )}
      </DialogContent>
      
      <DialogActions>
        <Button onClick={onClose}>Close</Button>
      </DialogActions>
    </Dialog>
  );
}; 