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
  Accordion,
  AccordionSummary,
  AccordionDetails,
} from '@mui/material';
import {
  Build as ToolIcon,
  ExpandMore as ExpandMoreIcon,
  Storage as ServerIcon,
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
  // Group tools by server
  const toolsByServer = React.useMemo(() => {
    const grouped: Record<string, MCPTool[]> = {};
    tools.forEach(tool => {
      const serverName = tool.server || 'Unknown';
      if (!grouped[serverName]) {
        grouped[serverName] = [];
      }
      grouped[serverName].push(tool);
    });
    return grouped;
  }, [tools]);

  const serverNames = Object.keys(toolsByServer).sort();

  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>
        <Box>
          <Typography variant="h6" component="div" sx={{ fontWeight: 600 }}>
            Available MCP Tools
          </Typography>
          <Typography variant="subtitle2" color="textSecondary">
            {tools.length} tool{tools.length !== 1 ? 's' : ''} connected
          </Typography>
        </Box>
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
          <Box>
            {serverNames.map((serverName) => (
              <Accordion key={serverName} defaultExpanded sx={{ mb: 1 }}>
                <AccordionSummary
                  expandIcon={<ExpandMoreIcon />}
                  sx={{
                    backgroundColor: 'primary.main',
                    color: 'primary.contrastText',
                    '&:hover': {
                      backgroundColor: 'primary.dark',
                    },
                    borderRadius: 1,
                    mb: 0.5,
                  }}
                >
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <ServerIcon />
                    <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>
                      {serverName}
                    </Typography>
                    <Chip 
                      label={`${toolsByServer[serverName].length} tools`}
                      size="small"
                      sx={{ 
                        backgroundColor: 'rgba(255, 255, 255, 0.2)',
                        color: 'inherit',
                        fontWeight: 'bold',
                      }}
                    />
                  </Box>
                </AccordionSummary>
                <AccordionDetails sx={{ p: 0 }}>
                  <List>
                    {toolsByServer[serverName].map((tool, index) => (
                      <ListItem key={`${tool.name}-${index}`} divider>
                        <ListItemIcon>
                          <ToolIcon color="primary" />
                        </ListItemIcon>
                        <ListItemText
                          primary={
                            <span style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                              <Typography variant="subtitle1" component="span">
                                {tool.name}
                              </Typography>
                              <Chip label="MCP" size="small" variant="outlined" color="primary" />
                            </span>
                          }
                          secondary={
                            <span>
                              <Typography variant="body2" color="textSecondary" component="span" sx={{ display: 'block', mb: 1 }}>
                                {tool.description}
                              </Typography>
                              {tool.input_schema && (
                                <Typography variant="caption" color="textSecondary" component="span" sx={{ display: 'block' }}>
                                  Parameters: {Object.keys(tool.input_schema.properties || {}).join(', ') || 'None'}
                                </Typography>
                              )}
                            </span>
                          }
                        />
                      </ListItem>
                    ))}
                  </List>
                </AccordionDetails>
              </Accordion>
            ))}
          </Box>
        )}
      </DialogContent>
      
      <DialogActions>
        <Button onClick={onClose}>Close</Button>
      </DialogActions>
    </Dialog>
  );
}; 