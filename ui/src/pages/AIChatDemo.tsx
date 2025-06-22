import React, { useState } from 'react';
import { Box, Typography, Container, Paper, Button, Alert, CircularProgress } from '@mui/material';
import { AIChatWidget } from '../components/ai-chat-widget';
import { QANDataDisplay } from '../components/qan-data-display';
import { useRecentQANData } from '../hooks/api/useQAN';
import { formatQANDataForAI, formatQANError } from '../utils/qanFormatter';

/**
 * Demo page showing how to integrate the AI Chat Widget
 * This can be used as a standalone page or the widget can be embedded in other pages
 */
const AIChatDemo: React.FC = () => {
  const [shouldOpenWithSample, setShouldOpenWithSample] = useState(false);
  const [sampleMessage, setSampleMessage] = useState('');
  
  // Fetch real QAN data (auto-fetch for display, but not for chat)
  const { 
    data: qanData, 
    error: qanError, 
    isLoading: isLoadingQAN, 
    refetch: fetchQANData 
  } = useRecentQANData(24, 10, { 
    enabled: true, // Auto-fetch for display
    retry: 1 
  });

  const handleAnalyzeRealQANData = async () => {
    try {
      const result = await fetchQANData();
      
      if (result.data) {
        const formattedData = formatQANDataForAI(result.data);
        setSampleMessage(formattedData);
        setShouldOpenWithSample(true);
      } else if (result.error) {
        const errorMessage = formatQANError(result.error);
        setSampleMessage(errorMessage);
        setShouldOpenWithSample(true);
      }
    } catch (error) {
      const errorMessage = formatQANError(error);
      setSampleMessage(errorMessage);
      setShouldOpenWithSample(true);
    }
  };

  const handleAnalyzeSampleData = () => {
    const sampleQANPrompt = `Please analyze this sample QAN (Query Analytics) data from a MySQL database and provide performance recommendations:

**Query Performance Summary:**
- Total Queries: 15,847
- Time Period: Last 24 hours
- Average Query Time: 2.3 seconds
- Slowest Query Time: 45.2 seconds

**Top 5 Slowest Queries:**

1. **Query:** \`SELECT o.*, c.name as customer_name FROM orders o JOIN customers c ON o.customer_id = c.id WHERE o.created_at BETWEEN '2024-01-01' AND '2024-12-31' ORDER BY o.created_at DESC\`
   - **Execution Count:** 1,234 times
   - **Average Time:** 12.5 seconds
   - **Total Time:** 15,425 seconds
   - **Rows Examined:** 2,500,000 avg
   - **Rows Sent:** 50,000 avg

2. **Query:** \`SELECT p.*, COUNT(oi.product_id) as order_count FROM products p LEFT JOIN order_items oi ON p.id = oi.product_id GROUP BY p.id HAVING order_count > 100\`
   - **Execution Count:** 892 times
   - **Average Time:** 8.7 seconds
   - **Total Time:** 7,760 seconds
   - **Rows Examined:** 1,800,000 avg
   - **Rows Sent:** 245 avg

3. **Query:** \`UPDATE inventory SET quantity = quantity - 1 WHERE product_id IN (SELECT product_id FROM order_items WHERE order_id = ?)\`
   - **Execution Count:** 3,456 times
   - **Average Time:** 4.2 seconds
   - **Total Time:** 14,515 seconds
   - **Rows Examined:** 500,000 avg
   - **Rows Affected:** 1 avg

4. **Query:** \`SELECT DATE(created_at) as date, COUNT(*) as daily_orders, SUM(total_amount) as daily_revenue FROM orders WHERE created_at >= DATE_SUB(NOW(), INTERVAL 90 DAY) GROUP BY DATE(created_at)\`
   - **Execution Count:** 156 times
   - **Average Time:** 6.8 seconds
   - **Total Time:** 1,061 seconds
   - **Rows Examined:** 1,200,000 avg
   - **Rows Sent:** 90 avg

5. **Query:** \`SELECT u.email, u.name, COUNT(o.id) as order_count, SUM(o.total_amount) as total_spent FROM users u LEFT JOIN orders o ON u.id = o.user_id WHERE u.created_at > '2024-01-01' GROUP BY u.id ORDER BY total_spent DESC LIMIT 100\`
   - **Execution Count:** 89 times
   - **Average Time:** 9.2 seconds
   - **Total Time:** 819 seconds
   - **Rows Examined:** 800,000 avg
   - **Rows Sent:** 100 avg

**Database Schema Context:**
- orders table: ~2M rows (id, customer_id, created_at, total_amount, status)
- customers table: ~500K rows (id, name, email, created_at)
- products table: ~50K rows (id, name, price, category_id)
- order_items table: ~8M rows (id, order_id, product_id, quantity, price)
- inventory table: ~50K rows (id, product_id, quantity, warehouse_id)
- users table: ~500K rows (id, email, name, created_at)

**Current Indexes:**
- orders: PRIMARY KEY (id), INDEX (customer_id), INDEX (created_at)
- customers: PRIMARY KEY (id), UNIQUE INDEX (email)
- products: PRIMARY KEY (id), INDEX (category_id)
- order_items: PRIMARY KEY (id), INDEX (order_id), INDEX (product_id)
- inventory: PRIMARY KEY (id), INDEX (product_id)
- users: PRIMARY KEY (id), UNIQUE INDEX (email)

Please analyze these queries and provide specific recommendations for:
1. Index optimizations
2. Query rewrites for better performance
3. Schema improvements
4. General best practices

Focus on the most impactful changes that could reduce query execution time and improve overall database performance.`;

    setSampleMessage(sampleQANPrompt);
    setShouldOpenWithSample(true);
  };
  return (
    <Container maxWidth="lg">
      <Box sx={{ py: 4 }}>
        <Typography variant="h4" component="h1" gutterBottom>
          AI Chat Assistant Demo
        </Typography>
        
        <Typography variant="body1" color="textSecondary" paragraph>
          This page demonstrates the AI Chat Widget integration and QAN data visualization.
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
                QAN Data Integration
              </Typography>
              <Typography variant="body2" paragraph>
                This demo can analyze both real QAN (Query Analytics) data from your PMM instance 
                and sample data for demonstration purposes.
              </Typography>
              
              {qanError && (
                <Alert severity="info" sx={{ mb: 2 }}>
                  <Typography variant="body2">
                    <strong>QAN Data Status:</strong> Real QAN data is not available. 
                    This is normal in development environments. You can still use the sample data button 
                    to see how the AI analyzes database performance information.
                  </Typography>
                </Alert>
              )}
              
              {qanData && (
                <Alert severity="success" sx={{ mb: 2 }}>
                  <Typography variant="body2">
                    <strong>QAN Data Status:</strong> Real QAN data is available! 
                    Found {qanData.total_rows} queries in the recent data.
                  </Typography>
                </Alert>
              )}
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

              <Box sx={{ mt: 3 }}>
                <Button 
                  variant="contained" 
                  color="primary" 
                  onClick={handleAnalyzeRealQANData}
                  disabled={isLoadingQAN}
                  sx={{ mr: 2, mb: 1 }}
                  startIcon={isLoadingQAN ? <CircularProgress size={20} /> : null}
                >
                  📊 Analyze Real QAN Data
                </Button>
                <Button 
                  variant="outlined" 
                  color="secondary" 
                  onClick={handleAnalyzeSampleData}
                  sx={{ mr: 2, mb: 1 }}
                >
                  🔍 Analyze Sample Data
                </Button>
                <Box sx={{ mt: 1 }}>
                  <Typography variant="caption" color="textSecondary" display="block">
                    <strong>Real QAN Data:</strong> Fetches actual query performance data from PMM
                  </Typography>
                  <Typography variant="caption" color="textSecondary" display="block">
                    <strong>Sample Data:</strong> Uses example data for demonstration purposes
                  </Typography>
                </Box>
              </Box>
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

            {/* QAN Data Section */}
            <Paper sx={{ p: 3, mb: 4 }}>
              <Typography variant="h6" gutterBottom>
                Query Analytics Data
              </Typography>
              
              {isLoadingQAN && (
                <Box sx={{ textAlign: 'center', py: 3 }}>
                  <CircularProgress sx={{ mb: 2 }} />
                  <Typography variant="body1">
                    Loading QAN data...
                  </Typography>
                </Box>
              )}

              {qanError && (
                <Alert severity="warning" sx={{ mb: 2 }}>
                  <Typography variant="body2">
                    <strong>Unable to load QAN data:</strong> {qanError.message}
                  </Typography>
                  <Typography variant="body2" color="textSecondary" sx={{ mt: 1 }}>
                    This is normal in development environments where QAN collection may not be configured.
                    In a production PMM setup with monitored databases, this would show real query performance data.
                  </Typography>
                </Alert>
              )}

              {qanData && !isLoadingQAN && (
                <QANDataDisplay 
                  data={qanData} 
                  maxQueries={10} 
                  onAnalyzeQuery={(queryData) => {
                    setSampleMessage(queryData);
                    setShouldOpenWithSample(true);
                  }}
                />
              )}
            </Paper>
      </Box>

      {/* The AI Chat Widget - appears as floating button */}
      <AIChatWidget 
        open={shouldOpenWithSample}
        position="bottom-right"
        maxWidth={400}
        maxHeight={600}
        initialMessage={shouldOpenWithSample ? sampleMessage : undefined}
        onMessageSent={() => {
          // Don't close immediately - let user see the response
          // Just clear the sample message so it doesn't resend
          setSampleMessage('');
        }}
        onOpenChange={(open) => {
          if (!open) {
            // Reset state when widget is manually closed
            setShouldOpenWithSample(false);
            setSampleMessage('');
          }
        }}
      />
    </Container>
  );
};

export default AIChatDemo; 