import React from 'react';
import { Box, Typography, Container, Paper } from '@mui/material';
import { AIChatWidget } from '../components/ai-chat-widget';

/**
 * Demo page showing how to integrate the AI Chat Widget
 * This can be used as a standalone page or the widget can be embedded in other pages
 */
const AIChatDemo: React.FC = () => {
  return (
    <Container maxWidth="lg">
      <Box sx={{ py: 4 }}>
        <Typography variant="h4" component="h1" gutterBottom>
          AI Chat Assistant Demo
        </Typography>
        
        <Typography variant="body1" color="textSecondary" paragraph>
          This page demonstrates the AI Chat Widget integration. The widget provides:
        </Typography>

        <Paper sx={{ p: 3, mb: 4 }}>
          <Typography variant="h6" gutterBottom>
            Features
          </Typography>
          <ul>
            <li>Real-time chat with AI assistant</li>
            <li>MCP (Model Context Protocol) tool support</li>
            <li>Streaming responses for better UX</li>
            <li>Persistent session history</li>
            <li>Markdown rendering for rich responses</li>
            <li>Tool execution and results display</li>
          </ul>
        </Paper>

        <Paper sx={{ p: 3, mb: 4 }}>
          <Typography variant="h6" gutterBottom>
            Usage
          </Typography>
          <Typography variant="body2" paragraph>
            The AI Chat Widget appears as a floating action button in the bottom-right corner. 
            Click it to open the chat interface and start conversing with the AI assistant.
          </Typography>
          
          <Typography variant="body2" paragraph>
            If MCP tools are configured in the backend, the AI can execute various operations 
            like file system access, database queries, and more.
          </Typography>
        </Paper>

        <Paper sx={{ p: 3 }}>
          <Typography variant="h6" gutterBottom>
            Integration
          </Typography>
          <Typography variant="body2" paragraph>
            To add the widget to any page, simply import and use:
          </Typography>
          <Box
            component="pre"
            sx={{
              backgroundColor: 'grey.100',
              p: 2,
              borderRadius: 1,
              overflow: 'auto',
              fontFamily: 'monospace',
              fontSize: '0.875rem',
            }}
          >
{`import { AIChatWidget } from '../components/ai-chat-widget';

// In your component:
<AIChatWidget 
  defaultOpen={false}
  position="bottom-right"
  maxWidth={400}
  maxHeight={600}
/>`}
          </Box>
        </Paper>
      </Box>

      {/* The AI Chat Widget - appears as floating button */}
      <AIChatWidget 
        defaultOpen={false}
        position="bottom-right"
        maxWidth={400}
        maxHeight={600}
      />
    </Container>
  );
};

export default AIChatDemo; 