import React, { useState, useEffect, useRef } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Typography,
  Paper,
  Box,
  CircularProgress,
  Alert,
  IconButton,
  Tooltip,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Chip,
} from '@mui/material';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import CloseIcon from '@mui/icons-material/Close';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import BuildIcon from '@mui/icons-material/Build';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import ErrorIcon from '@mui/icons-material/Error';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { oneLight } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { aiChatAPI, StreamMessage } from '../../api/aichat';
import { QANReportResponse } from '../../api/qan';
import { formatQANDataForAI } from '../../utils/qanFormatter';
import { copyToClipboard } from '../../utils/formatters';

interface ToolExecution {
  id: string;
  name: string;
  arguments: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
  result?: string;
  error?: string;
  timestamp: number;
}

interface QANOverviewAnalysisDialogProps {
  open: boolean;
  onClose: () => void;
  qanData: QANReportResponse | null;
}

export const QANOverviewAnalysisDialog: React.FC<QANOverviewAnalysisDialogProps> = ({
  open,
  onClose,
  qanData,
}) => {
  const [loading, setLoading] = useState(false);
  const [analysisResult, setAnalysisResult] = useState<string>('');
  const [error, setError] = useState<string | null>(null);
  const [toolExecutions, setToolExecutions] = useState<ToolExecution[]>([]);
  const [analysisSessionId, setAnalysisSessionId] = useState<string>('');
  const activeStreamCountRef = useRef(0);

  // Thread-safe counter operations
  const incrementStreamCount = () => {
    activeStreamCountRef.current += 1;
    console.log('🔢 Overview analysis stream count incremented:', activeStreamCountRef.current);
  };

  const decrementStreamCount = (sessionId: string) => {
    activeStreamCountRef.current = Math.max(0, activeStreamCountRef.current - 1);
    console.log('🔢 Overview analysis stream count decremented:', activeStreamCountRef.current);
    
    // Check if we should cleanup the session
    if (activeStreamCountRef.current === 0 && sessionId) {
      console.log('🧹 All overview analysis streams completed, cleaning up session:', sessionId);
      aiChatAPI.deleteSession(sessionId).catch(error => {
        console.warn('⚠️ Failed to cleanup overview analysis session:', error);
      }).then(() => {
        setAnalysisSessionId('');
        console.log('🧹 Overview analysis session cleaned up');
      });
    }
  };

  // Reset state when dialog opens/closes
  useEffect(() => {
    if (open && qanData) {
      setLoading(true);
      setError(null);
      setAnalysisResult('');
      setToolExecutions([]);
      setAnalysisSessionId('');
      activeStreamCountRef.current = 0;
      
      // Start comprehensive analysis
      handleAnalyzeAllQueries();
    }
  }, [open, qanData]);

  // Extracted message handler for analysis stream
  const handleAnalysisStreamMessage = (message: StreamMessage) => {
    // Update session ID if backend provides one
    if (message.session_id && analysisSessionId === '') {
      console.log('🔄 Backend created/provided overview session ID:', message.session_id);
      setAnalysisSessionId(message.session_id);
    }
    
    switch (message.type) {
      case 'message':
        if (message.content) {
          setAnalysisResult(prev => prev + message.content);
        }
        break;
      case 'tool_approval_request':
        if (message.tool_calls && message.request_id) {
          incrementStreamCount();
          // Add pending tool executions to state
          const newToolExecutions = message.tool_calls.map(tool => ({
            id: tool.id,
            name: tool.function.name,
            arguments: tool.function.arguments,
            status: 'pending' as const,
            timestamp: Date.now()
          }));
          setToolExecutions(prev => [...prev, ...newToolExecutions]);
          
          console.log('🔧 Auto-approving tools for overview analysis:', message.tool_calls);
          
          handleToolApproval(true, message);
        }
        break;
      case 'tool_execution':
        if (message.tool_executions) {
          message.tool_executions.forEach(execution => {
            setToolExecutions(prev => prev.map(tool => {
              if (tool.id === execution.id) {
                if (execution.result) {
                  return {
                    ...tool,
                    status: 'completed' as const,
                    result: execution.result
                  };
                } else if (execution.error) {
                  return {
                    ...tool,
                    status: 'failed' as const,
                    error: execution.error
                  };
                } else {
                  return {
                    ...tool,
                    status: 'running' as const
                  };
                }
              }
              return tool;
            }));
          });
        }
        break;
      case 'error':
        handleStreamError(message.content || 'Analysis failed');
        break;
      case 'done':
        handleStreamComplete();
        decrementStreamCount(message.session_id);
        break;
    }
  };

  const handleStreamError = (error: string) => {
    setError(error);
    setLoading(false);
    decrementStreamCount(analysisSessionId);
  };

  const handleStreamComplete = () => {
    setLoading(false);
  };

  const handleAnalyzeAllQueries = async () => {
    if (!qanData) return;

    try {
      setLoading(true);
      setError(null);

      // Generate comprehensive analysis prompt
      const analysisPrompt = formatQANDataForAI(qanData);
      
      const enhancedPrompt = `${analysisPrompt}

**Comprehensive Analysis Request:**
Please provide a detailed analysis of this QAN data including:

1. **Performance Overview**: Overall database performance assessment
2. **Query Patterns**: Common patterns and anti-patterns in the queries
3. **Resource Bottlenecks**: Identify CPU, I/O, and memory bottlenecks
4. **Optimization Priorities**: Top 3-5 optimization recommendations ranked by impact
5. **Index Recommendations**: Specific index suggestions for the slowest queries
6. **Schema Improvements**: Any schema-level improvements suggested
7. **Monitoring Alerts**: Recommended thresholds and alerts to set up

Focus on actionable insights that can immediately improve database performance.`;

      // Create unique session ID for this analysis
      const sessionId = `qan_overview_${Date.now()}`;
      setAnalysisSessionId(sessionId);
      incrementStreamCount();

      console.log('🚀 Starting comprehensive QAN analysis with session:', sessionId);

      // Start streaming analysis using new separate endpoints pattern
      await aiChatAPI.streamChatWithSeparateEndpoints(
        sessionId,
        enhancedPrompt,
        handleAnalysisStreamMessage,
        (error: string) => handleStreamError(error),
        () => handleStreamComplete()
      );

    } catch (error) {
      console.error('❌ Overview analysis failed:', error);
      setError(error instanceof Error ? error.message : 'Analysis failed');
      setLoading(false);
    }
  };

  const handleToolApproval = async (approved: boolean, message: StreamMessage) => {
    try {
      // Update tool statuses to running if approved
      if (approved) {
        setToolExecutions(prev => prev.map(tool => {
          const isPendingTool = message.tool_calls?.some(tc => tc.id === tool.id);
          return isPendingTool && tool.status === 'pending' 
            ? { ...tool, status: 'running' as const }
            : tool;
        }));
      } else {
        // Mark tools as failed if denied
        setToolExecutions(prev => prev.map(tool => {
          const isPendingTool = message.tool_calls?.some(tc => tc.id === tool.id);
          return isPendingTool && tool.status === 'pending'
            ? { ...tool, status: 'failed' as const, error: 'User denied tool execution' }
            : tool;
        }));
      }

      // Send approval/denial as a special message format
      const approvalMessage = approved 
        ? `[APPROVE_TOOLS:${message.request_id}]`
        : `[DENY_TOOLS:${message.request_id}]`;

      await aiChatAPI.streamChatWithSeparateEndpoints(
        message.session_id,
        approvalMessage,
        handleAnalysisStreamMessage,
        handleStreamError,
        handleStreamComplete
      );
    } catch (error) {
      console.error('Error handling tool approval:', error);
      setError('Failed to process tool approval');
    }
  };

  const handleClose = () => {
    onClose();
  };

  const getToolStatusIcon = (status: ToolExecution['status']) => {
    switch (status) {
      case 'pending':
        return <CircularProgress size={16} />;
      case 'running':
        return <CircularProgress size={16} />;
      case 'completed':
        return <CheckCircleIcon color="success" fontSize="small" />;
      case 'failed':
        return <ErrorIcon color="error" fontSize="small" />;
      default:
        return null;
    }
  };

  const getToolStatusColor = (status: ToolExecution['status']) => {
    switch (status) {
      case 'pending':
        return 'default';
      case 'running':
        return 'primary';
      case 'completed':
        return 'success';
      case 'failed':
        return 'error';
      default:
        return 'default';
    }
  };

  const queryCount = qanData?.rows?.filter(row => 
    row.fingerprint !== 'TOTAL' && row.dimension !== '' && row.rank > 0
  ).length || 0;

  return (
    <Dialog 
      open={open} 
      onClose={handleClose}
      maxWidth="lg"
      fullWidth
      PaperProps={{
        sx: { height: '90vh', maxHeight: '90vh' }
      }}
    >
      <DialogTitle sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', pb: 1 }}>
        <Typography variant="h6">
          📊 QAN Overview Analysis ({queryCount} queries)
        </Typography>
        <IconButton onClick={handleClose} size="small">
          <CloseIcon />
        </IconButton>
      </DialogTitle>

      <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: 2, overflow: 'hidden' }}>
        {/* Loading State */}
        {loading && (
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, p: 2 }}>
            <CircularProgress size={24} />
            <Typography variant="body2">
              Analyzing {queryCount} queries and generating comprehensive insights...
            </Typography>
          </Box>
        )}

        {/* Error State */}
        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            <Typography variant="body2">
              <strong>Analysis Error:</strong> {error}
            </Typography>
          </Alert>
        )}

        {/* Tool Executions */}
        {toolExecutions.length > 0 && (
          <Accordion defaultExpanded={false}>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <BuildIcon fontSize="small" />
                <Typography variant="subtitle2">
                  Tool Executions ({toolExecutions.length})
                </Typography>
              </Box>
            </AccordionSummary>
            <AccordionDetails>
              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                {toolExecutions.map((tool) => (
                  <Box key={tool.id} sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                    {getToolStatusIcon(tool.status)}
                    <Chip 
                      label={tool.name} 
                      size="small" 
                      color={getToolStatusColor(tool.status) as any}
                    />
                    <Typography variant="caption" color="textSecondary">
                      {new Date(tool.timestamp).toLocaleTimeString()}
                    </Typography>
                    {tool.result && (
                      <Tooltip title="Copy result">
                        <IconButton 
                          size="small" 
                          onClick={() => copyToClipboard(tool.result || '')}
                        >
                          <ContentCopyIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                    )}
                  </Box>
                ))}
              </Box>
            </AccordionDetails>
          </Accordion>
        )}

        {/* Analysis Results */}
        {analysisResult && (
          <Paper 
            sx={{ 
              flex: 1, 
              p: 2, 
              overflow: 'auto',
              backgroundColor: 'background.default'
            }}
          >
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
              <Typography variant="subtitle1" fontWeight="bold">
                📋 Analysis Results
              </Typography>
              <Tooltip title="Copy analysis">
                <IconButton 
                  size="small" 
                  onClick={() => copyToClipboard(analysisResult)}
                >
                  <ContentCopyIcon fontSize="small" />
                </IconButton>
              </Tooltip>
            </Box>
            
            <Box sx={{ 
              '& pre': { 
                backgroundColor: '#f5f5f5', 
                padding: 1, 
                borderRadius: 1,
                overflow: 'auto'
              },
              '& code': {
                backgroundColor: '#f5f5f5',
                padding: '2px 4px',
                borderRadius: '4px',
                fontSize: '0.875rem'
              },
              '& blockquote': {
                borderLeft: '4px solid #2196f3',
                paddingLeft: 2,
                margin: '16px 0',
                fontStyle: 'italic'
              }
            }}>
              <ReactMarkdown
                remarkPlugins={[remarkGfm]}
                components={{
                  code({ className, children, ...props }) {
                    const match = /language-(\w+)/.exec(className || '');
                    const isInline = !match;
                    return !isInline ? (
                      <SyntaxHighlighter
                        language={match[1]}
                        style={oneLight as any}
                        PreTag="div"
                        customStyle={{
                          margin: 0,
                          fontSize: '0.875rem',
                          lineHeight: 1.4,
                          overflow: 'auto'
                        }}
                      >
                        {String(children).replace(/\n$/, '')}
                      </SyntaxHighlighter>
                    ) : (
                      <code className={className} {...props}>
                        {children}
                      </code>
                    );
                  },
                }}
              >
                {analysisResult}
              </ReactMarkdown>
            </Box>
          </Paper>
        )}

        {/* Empty State */}
        {!loading && !analysisResult && !error && (
          <Box sx={{ textAlign: 'center', py: 4 }}>
            <Typography variant="body1" color="textSecondary">
              Ready to analyze {queryCount} queries
            </Typography>
          </Box>
        )}
      </DialogContent>

      <DialogActions sx={{ p: 2 }}>
        <Button onClick={handleClose} variant="outlined">
          Close
        </Button>
        {analysisResult && (
          <Button 
            onClick={() => copyToClipboard(analysisResult)}
            variant="contained"
            startIcon={<ContentCopyIcon />}
          >
            Copy Analysis
          </Button>
        )}
      </DialogActions>
    </Dialog>
  );
};

export default QANOverviewAnalysisDialog; 