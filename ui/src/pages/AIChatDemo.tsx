import React, { useState } from 'react';
import { Box, Typography, Container, Paper, Button, Alert, CircularProgress } from '@mui/material';
import { AIChatWidget } from '../components/ai-chat-widget';
import { QANDataDisplay } from '../components/qan-data-display';
import { QANOverviewAnalysisDialog } from '../components/qan-data-display/QANOverviewAnalysisDialog';
import { useRecentQANData } from '../hooks/api/useQAN';

import { QANLabel } from '../api/qan';

/**
 * Demo page showing how to integrate the AI Chat Widget
 * This can be used as a standalone page or the widget can be embedded in other pages
 */
const AIChatDemo: React.FC = () => {
  const [shouldOpenWithSample, setShouldOpenWithSample] = useState(false);
  const [sampleMessage, setSampleMessage] = useState('');
  

  
  // Overview Analysis Dialog state
  const [overviewDialogOpen, setOverviewDialogOpen] = useState(false);
  
  // Service filter state
  const [selectedServices, setSelectedServices] = useState<string[]>([]);
  
  // Time range filter state
  const [timeRangeHours, setTimeRangeHours] = useState<number>(24); // Default to 24 hours
  
  // Sorting and pagination state
  const [orderBy, setOrderBy] = useState<string>('-load'); // Default to load descending
  const [page, setPage] = useState<number>(0);
  const [pageSize, setPageSize] = useState<number>(10);
  
  // Convert selected services to QAN labels format
  const qanFilters: QANLabel[] = selectedServices.length > 0 ? [
    {
      key: 'service_name',
      value: selectedServices
    }
  ] : [];
  
  // Calculate offset for backend pagination
  const offset = page * pageSize;
  
  // Fetch real QAN data with filters, sorting, and pagination
  const { 
    data: qanData, 
    error: qanError, 
    isLoading: isLoadingQAN, 
    refetch: fetchQANData 
  } = useRecentQANData(timeRangeHours, pageSize, qanFilters, orderBy, offset, { 
    enabled: true, // Auto-fetch for display
    retry: 1 
  });

  const handleServiceFilterChange = (services: string[]) => {
    setSelectedServices(services);
    setPage(0); // Reset to first page when filters change
  };
  
  const handleTimeRangeChange = (hours: number) => {
    setTimeRangeHours(hours);
    setPage(0); // Reset to first page when time range changes
  };

  const handleSortChange = (newOrderBy: string) => {
    setOrderBy(newOrderBy);
    setPage(0); // Reset to first page when sorting changes
  };

  const handlePageChange = (newPage: number, newPageSize: number) => {
    setPage(newPage);
    if (newPageSize !== pageSize) {
      setPageSize(newPageSize);
    }
  };

  const handleAnalyzeRealQANData = async () => {
    if (!qanData || !qanData.rows || qanData.rows.length === 0) {
      // If no QAN data, try to fetch it first
      try {
        await fetchQANData();
      } catch (err) {
        console.error('Failed to fetch QAN data:', err);
        // Optionally, set a user notification state here
        return;
      }
      return;
    }

    // Open the dedicated overview analysis dialog
    setOverviewDialogOpen(true);
  };


  return (
    <Container maxWidth="lg">
      <Box sx={{ py: 4 }}>
        <Typography variant="h4" component="h1" gutterBottom>
          AI Chat Demo
        </Typography>
        
        <Typography variant="body1" color="textSecondary" paragraph>
          This is a demonstration of how AI can be integrated into PMM Query Analytics (QAN).
          The AI assistant can analyze query performance data, suggest optimizations, and provide insights based on real QAN metrics.
        </Typography>
            <Paper sx={{ p: 3, mb: 4 }}>
              <Typography variant="h6" gutterBottom>
                AI Integration Features
              </Typography>
              <ul>
                <li>AI-powered query performance analysis</li>
                <li>Real-time chat interface for database insights</li>
                <li>Integration with PMM Query Analytics data</li>
                <li>Intelligent optimization recommendations</li>
                <li>Interactive query exploration and filtering</li>
                <li>Persistent chat sessions with context retention</li>
              </ul>
            </Paper>

            <Paper sx={{ p: 3, mb: 4 }}>
              <Typography variant="h6" gutterBottom>
                Query Analytics Integration
              </Typography>
              <Typography variant="body2" paragraph>
                The AI assistant is integrated with PMM's Query Analytics engine to provide 
                intelligent insights about database performance. It can analyze real query metrics, 
                identify bottlenecks, and suggest optimization strategies.
              </Typography>
              
              {qanError && (
                <Alert severity="info" sx={{ mb: 2 }}>
                  <Typography variant="body2">
                    <strong>QAN Data Status:</strong> Real QAN data is not available. 
                    This is normal in development environments.
                  </Typography>
                </Alert>
              )}
              
              {qanData && (
                <Alert severity="success" sx={{ mb: 2 }}>
                  <Typography variant="body2">
                    <strong>QAN Data Status:</strong> Connected to live QAN data! 
                    Found {qanData.total_rows} queries in the recent data.
                  </Typography>
                </Alert>
              )}
            </Paper>

            <Paper sx={{ p: 3, mb: 4 }}>
              <Typography variant="h6" gutterBottom>
                How to Use
              </Typography>
              <Typography variant="body2" paragraph>
                1. <strong>Open the AI Chat:</strong> Click the floating chat button in the bottom-right corner
              </Typography>
              
              <Typography variant="body2" paragraph>
                2. <strong>Analyze Query Performance:</strong> Ask the AI to analyze the current QAN data, 
                identify slow queries, or suggest optimizations
              </Typography>
              
              <Typography variant="body2" paragraph>
                3. <strong>Interactive Exploration:</strong> Use the query table below to filter and sort data, 
                then ask the AI for insights about specific queries or patterns
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
                  📊 Analyze Current QAN Data
                </Button>
                <Box sx={{ mt: 1 }}>
                  <Typography variant="caption" color="textSecondary" display="block">
                    This will open the AI analysis popup with comprehensive analysis of all current QAN data
                  </Typography>
                </Box>
              </Box>
            </Paper>

            <Paper sx={{ p: 3 }}>
              <Typography variant="h6" gutterBottom>
                AI-Powered Database Insights
              </Typography>
              <Typography variant="body2" paragraph>
                This demonstration shows how AI can be seamlessly integrated into database monitoring tools 
                to provide intelligent analysis and optimization recommendations. The AI assistant understands 
                query performance metrics and can help database administrators identify issues and improve performance.
              </Typography>
              <Typography variant="body2" paragraph>
                Key capabilities include analyzing slow queries, suggesting index optimizations, 
                identifying resource bottlenecks, and providing best practices for database performance tuning.
              </Typography>
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
                <>

                <QANDataDisplay 
                  data={qanData} 
                  selectedServices={selectedServices}
                  onServiceFilterChange={handleServiceFilterChange}
                  timeRangeHours={timeRangeHours}
                  onTimeRangeChange={handleTimeRangeChange}
                    orderBy={orderBy}
                    onSortChange={handleSortChange}
                    page={page}
                    pageSize={pageSize}
                    onPageChange={handlePageChange}
                  onAnalyzeQuery={(queryData) => {
                    setSampleMessage(queryData);
                    setShouldOpenWithSample(true);
                  }}
                    onRefresh={fetchQANData}
                    isRefreshing={isLoadingQAN}
                />
                </>
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


      
      {/* QAN Overview Analysis Dialog */}
      <QANOverviewAnalysisDialog
        open={overviewDialogOpen}
        onClose={() => setOverviewDialogOpen(false)}
        qanData={qanData || null}
      />
    </Container>
  );
};

export default AIChatDemo; 