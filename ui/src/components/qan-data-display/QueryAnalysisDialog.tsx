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
import RecommendIcon from '@mui/icons-material/Lightbulb';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import BuildIcon from '@mui/icons-material/Build';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import ErrorIcon from '@mui/icons-material/Error';
import FormatAlignLeftIcon from '@mui/icons-material/FormatAlignLeft';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { oneLight } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { format as formatSQL } from 'sql-formatter';
import { aiChatAPI, StreamMessage } from '../../api/aichat';
import { QANRow } from '../../api/qan';
import { generateDetailedQueryAnalysisPrompt } from '../../utils/queryAnalysisPrompts';
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

interface QueryAnalysisDialogProps {
  open: boolean;
  onClose: () => void;
  selectedQuery: QANRow | null;
  rank: number;
}

export const QueryAnalysisDialog: React.FC<QueryAnalysisDialogProps> = ({
  open,
  onClose,
  selectedQuery,
  rank,
}) => {
  const [loading, setLoading] = useState(false);
  const [analysisResult, setAnalysisResult] = useState<string>('');
  const [error, setError] = useState<string | null>(null);

  const [toolExecutions, setToolExecutions] = useState<ToolExecution[]>([]);
  const [analysisSessionId, setAnalysisSessionId] = useState<string>('');
  const [isQueryFormatted, setIsQueryFormatted] = useState<boolean>(false);
  const activeStreamsRef = useRef<Map<string, { type: string; timestamp: number }>>(new Map());

  // Stream management operations
  const addStream = (streamId: string, type: string) => {
    activeStreamsRef.current.set(streamId, { type, timestamp: Date.now() });
    console.log('🔢 Stream added:', streamId, 'type:', type, 'total:', activeStreamsRef.current.size);
    debugStreams();
  };

  const removeStream = (streamId: string, sessionId: string) => {
    const wasRemoved = activeStreamsRef.current.delete(streamId);
    console.log('🔢 Stream removed:', streamId, 'success:', wasRemoved, 'remaining:', activeStreamsRef.current.size);
    debugStreams();
    
    // Check if we should cleanup the session
    if (activeStreamsRef.current.size === 0 && sessionId) {
      console.log('🧹 All streams completed, cleaning up session:', sessionId);
      // Clean up immediately since dialog might be closing
      aiChatAPI.deleteSession(sessionId).catch(error => {
        console.warn('⚠️ Failed to cleanup analysis session:', error);
      }).then(() => {
        setAnalysisSessionId(''); // Reset session ID to let backend create a new one
        console.log('🧹 Session cleaned up');
      });
    }
  };

  const clearAllStreams = () => {
    console.log('🧹 Clearing all streams, count:', activeStreamsRef.current.size);
    activeStreamsRef.current.clear();
  };

  const debugStreams = () => {
    console.log('🔍 Current active streams:', Array.from(activeStreamsRef.current.entries()));
  };

  // Reset state when dialog opens/closes
  useEffect(() => {
    if (open && selectedQuery) {
      setLoading(true);
      setError(null);
      setAnalysisResult('');
      setToolExecutions([]);
      setAnalysisSessionId(''); // Reset session ID to let backend create a new one
      setIsQueryFormatted(false); // Reset formatting state
      clearAllStreams();
      
      // Start analysis
      handleAnalyzeQuery();
    }
    // Note: No cleanup on close - let the stream counter handle session cleanup
  }, [open, selectedQuery]);

  // Format SQL query
  const formatSQLQuery = (query: string): string => {
    try {
      return formatSQL(query, {
        language: 'mysql', // Can be 'mysql', 'postgresql', 'sql', etc.
        tabWidth: 2,
        useTabs: false,
        keywordCase: 'upper',
        linesBetweenQueries: 2,
      });
    } catch (error) {
      console.warn('Failed to format SQL query:', error);
      return query; // Return original if formatting fails
    }
  };

  // Get the query to display (formatted or original)
  const getDisplayQuery = (): string => {
    const originalQuery = selectedQuery?.fingerprint || 'N/A';
    if (isQueryFormatted && originalQuery !== 'N/A') {
      return formatSQLQuery(originalQuery);
    }
    return originalQuery;
  };

  // Toggle query formatting
  const toggleQueryFormatting = () => {
    setIsQueryFormatted(!isQueryFormatted);
  };

  // Extracted message handler for analysis stream
  const handleAnalysisStreamMessage = (message: StreamMessage, currentStreamId?: string) => {
    // Update session ID if backend provides one (for auto-created sessions)
    if (message.session_id && analysisSessionId === '') {
      console.log('🔄 Backend created/provided session ID:', message.session_id);
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
          // Add pending tool executions to state
          const newToolExecutions = message.tool_calls.map(tool => ({
            id: tool.id,
            name: tool.function.name,
            arguments: tool.function.arguments,
            status: 'pending' as const,
            timestamp: Date.now()
          }));
          setToolExecutions(prev => [...prev, ...newToolExecutions]);
          
          console.log('🔧 Auto-approving tools for popup analysis:', message.tool_calls);
          
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
        // Stream errors are now handled by stream-aware error handlers
        console.log('📊 Analysis stream error for', currentStreamId, ':', message.error);
        break;
      case 'done':
        // Stream completion is now handled by stream-aware complete handlers
        console.log('📊 Analysis stream completed for', currentStreamId);
        // But we still need to handle tool approval completion here
        if (currentStreamId && currentStreamId.startsWith('tool_approval_')) {
          removeStream(currentStreamId, message.session_id);
        }
        break;
    }
  };

  const handleAnalyzeQuery = async () => {
    if (!selectedQuery) return;

    const analysisPrompt = generateDetailedQueryAnalysisPrompt({
      selectedQuery,
      rank,
    });

    const streamId = 'main_analysis';

    // Create stream-aware handlers
    const streamAwareMessageHandler = (message: StreamMessage) => {
      handleAnalysisStreamMessage(message, streamId);
    };

    const streamAwareErrorHandler = (error: string) => {
      console.error('Stream error for', streamId, ':', error);
      setError(error);
      setLoading(false);
      removeStream(streamId, analysisSessionId);
    };

    const streamAwareCompleteHandler = () => {
      console.log('Stream completed for', streamId);
      setLoading(false);
      removeStream(streamId, analysisSessionId);
    };

    try {
      addStream(streamId, 'analysis_prompt');
      const cleanup = await aiChatAPI.streamChatWithSeparateEndpoints(
        analysisSessionId,
        analysisPrompt,
        streamAwareMessageHandler,
        streamAwareErrorHandler,
        streamAwareCompleteHandler
      );

      // Store cleanup function for potential cancellation
      return cleanup;
    } catch (error) {
      console.error('Error analyzing query:', error);
      setError('Failed to start analysis. Please try again.');
      setLoading(false);
      removeStream(streamId, analysisSessionId);
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

      const approvalStreamId = `approval_${message.request_id}`;
      addStream(approvalStreamId, 'approval_message');

      // Create stream-aware handlers for approval
      const approvalMessageHandler = (msg: StreamMessage) => {
        handleAnalysisStreamMessage(msg, approvalStreamId);
      };

      const approvalErrorHandler = (error: string) => {
        console.error('Approval stream error for', approvalStreamId, ':', error);
        setError('Failed to process tool approval');
        removeStream(approvalStreamId, message.session_id);
      };

      const approvalCompleteHandler = () => {
        console.log('Approval stream completed for', approvalStreamId);
        removeStream(approvalStreamId, message.session_id);
      };

      await aiChatAPI.streamChatWithSeparateEndpoints(
        message.session_id,
        approvalMessage,
        approvalMessageHandler,
        approvalErrorHandler,
        approvalCompleteHandler
      );
    } catch (error) {
      console.error('Error handling tool approval:', error);
      setError('Failed to process tool approval');
    }
  };

  const handleClose = () => {
    // Session cleanup is handled by the stream counter
    onClose();
  };

  const getToolStatusIcon = (status: ToolExecution['status']) => {
    switch (status) {
      case 'completed':
        return <CheckCircleIcon sx={{ color: 'success.main', fontSize: 16 }} />;
      case 'failed':
        return <ErrorIcon sx={{ color: 'error.main', fontSize: 16 }} />;
      case 'running':
        return <CircularProgress size={16} />;
      case 'pending':
        return <BuildIcon sx={{ color: 'grey.500', fontSize: 16 }} />;
      default:
        return null;
    }
  };

  const getToolStatusColor = (status: ToolExecution['status']) => {
    switch (status) {
      case 'completed':
        return 'success';
      case 'failed':
        return 'error';
      case 'running':
        return 'info';
      case 'pending':
        return 'default';
      default:
        return 'default';
    }
  };

  if (!selectedQuery) return null;

  return (
    <Dialog 
      open={open} 
      onClose={handleClose}
      maxWidth="md"
      fullWidth
    >
      <DialogTitle>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <RecommendIcon />
          <Typography variant="h6">
            AI Query Analysis & Recommendations
          </Typography>
        </Box>
      </DialogTitle>
      <DialogContent>
        {selectedQuery && (
          <Paper sx={{ p: 2, mb: 2, bgcolor: 'grey.50' }}>
            <Typography variant="subtitle2" gutterBottom>
              Query Details:
            </Typography>
            <Box sx={{ mb: 1 }}>
              <Typography variant="caption" color="textSecondary">
                Query ID:
              </Typography>
              <Typography 
                variant="body2" 
                sx={{ 
                  fontFamily: 'monospace',
                  fontSize: '0.875rem',
                  fontWeight: 'bold',
                  color: 'primary.main'
                }}
              >
                {selectedQuery.dimension}
              </Typography>
            </Box>
            <Box sx={{ mb: 1 }}>
              <Typography variant="caption" color="textSecondary">
                Database:
              </Typography>
              <Typography variant="body2">
                {selectedQuery.database || 'N/A'}
              </Typography>
            </Box>
            <Box>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}>
                <Typography variant="caption" color="textSecondary">
                  Query:
                </Typography>
                <Box sx={{ display: 'flex', gap: 0.5 }}>
                  <Tooltip title={isQueryFormatted ? "Show original query" : "Format SQL query"}>
                    <IconButton
                      onClick={toggleQueryFormatting}
                      size="small"
                      sx={{ 
                        p: 0.5,
                        color: isQueryFormatted ? 'primary.main' : 'inherit'
                      }}
                    >
                      <FormatAlignLeftIcon fontSize="small" />
                    </IconButton>
                  </Tooltip>
                  <Tooltip title="Copy query">
                    <IconButton
                      onClick={() => copyToClipboard(getDisplayQuery())}
                      size="small"
                      sx={{ p: 0.5 }}
                    >
                      <ContentCopyIcon fontSize="small" />
                    </IconButton>
                  </Tooltip>
                </Box>
              </Box>
              <Box sx={{ 
                border: '1px solid',
                borderColor: 'divider',
                borderRadius: 1,
                overflow: 'hidden'
              }}>
                <SyntaxHighlighter
                  language="sql"
                  style={oneLight}
                  customStyle={{
                    margin: 0,
                    fontSize: '0.875rem',
                    lineHeight: 1.4,
                    maxHeight: '200px',
                    overflow: 'auto'
                  }}
                  wrapLongLines
                >
                  {getDisplayQuery()}
                </SyntaxHighlighter>
              </Box>
            </Box>
          </Paper>
        )}

        {/* Tool Executions Display */}
        {toolExecutions.length > 0 && (
          <Paper sx={{ p: 2, mb: 2, bgcolor: 'grey.50', border: '1px solid', borderColor: 'divider' }}>
            <Typography variant="subtitle2" gutterBottom sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              <BuildIcon sx={{ color: 'text.secondary' }} />
              Tool Executions ({toolExecutions.length})
            </Typography>
            
            {toolExecutions.map((tool) => (
              <Accordion key={tool.id} sx={{ mb: 1, bgcolor: 'background.paper', border: '1px solid', borderColor: 'divider' }}>
                <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, width: '100%' }}>
                    {getToolStatusIcon(tool.status)}
                    <Typography variant="body2" sx={{ fontWeight: 'bold', flex: 1 }}>
                      {tool.name}
                    </Typography>
                    <Chip 
                      label={tool.status.toUpperCase()} 
                      size="small" 
                      color={getToolStatusColor(tool.status) as any}
                      variant="outlined"
                    />
                  </Box>
                </AccordionSummary>
                <AccordionDetails>
                  <Box sx={{ mb: 2 }}>
                    <Typography variant="caption" color="textSecondary">
                      Arguments:
                    </Typography>
                    <Typography 
                      variant="body2" 
                      sx={{ 
                        fontFamily: 'monospace',
                        fontSize: '0.75rem',
                        bgcolor: 'grey.100',
                        p: 1,
                        borderRadius: 1,
                        border: '1px solid',
                        borderColor: 'divider',
                        overflow: 'auto',
                        maxHeight: '100px'
                      }}
                    >
                      {tool.arguments}
                    </Typography>
                  </Box>
                  
                  {tool.result && (
                    <Box sx={{ mb: 2 }}>
                      <Typography variant="caption" color="textSecondary">
                        Result:
                      </Typography>
                      <Typography 
                        variant="body2" 
                        sx={{ 
                          fontFamily: 'monospace',
                          fontSize: '0.75rem',
                          bgcolor: 'grey.100',
                          p: 1,
                          borderRadius: 1,
                          border: '1px solid',
                          borderColor: 'success.main',
                          overflow: 'auto',
                          maxHeight: '150px'
                        }}
                      >
                        {tool.result}
                      </Typography>
                    </Box>
                  )}
                  
                  {tool.error && (
                    <Box sx={{ mb: 2 }}>
                      <Typography variant="caption" color="textSecondary">
                        Error:
                      </Typography>
                      <Typography 
                        variant="body2" 
                        sx={{ 
                          fontFamily: 'monospace',
                          fontSize: '0.75rem',
                          bgcolor: 'grey.100',
                          p: 1,
                          borderRadius: 1,
                          border: '1px solid',
                          borderColor: 'error.main',
                          overflow: 'auto',
                          maxHeight: '100px'
                        }}
                      >
                        {tool.error}
                      </Typography>
                    </Box>
                  )}
                  
                  <Typography variant="caption" color="textSecondary">
                    Started: {new Date(tool.timestamp).toLocaleTimeString()}
                  </Typography>
                </AccordionDetails>
              </Accordion>
            ))}
          </Paper>
        )}

        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}

        {loading && !analysisResult && (
          <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', py: 4 }}>
            <CircularProgress />
            <Typography variant="body2" sx={{ mt: 2 }}>
              Analyzing query performance with AI...
            </Typography>
          </Box>
        )}

        {analysisResult && (
          <Box sx={{ mt: 1 }}>
            <ReactMarkdown
              remarkPlugins={[remarkGfm]}
              components={{
                p: ({ children }) => (
                  <Typography
                    variant="body2"
                    component="p"
                    sx={{ 
                      mb: 1, 
                      '&:last-child': { mb: 0 },
                    }}
                  >
                    {children}
                  </Typography>
                ),
                code: ({ inline, className, children }: any) => {
                  const isInlineCode = inline !== false && !String(children).includes('\n');
                  const match = /language-(\w+)/.exec(className || '');
                  const language = match ? match[1] : '';
                  
                  return isInlineCode ? (
                    <code
                      style={{
                        backgroundColor: 'rgba(0, 0, 0, 0.1)',
                        padding: '1px 3px',
                        borderRadius: '3px',
                        fontFamily: 'monospace',
                        fontSize: '0.8em',
                        display: 'inline-block',
                        whiteSpace: 'pre-wrap',
                        wordBreak: 'break-all',
                        maxWidth: '100%',
                        overflowWrap: 'break-word',
                      }}
                    >
                      {children}
                    </code>
                  ) : (
                    <Box
                      sx={{
                        position: 'relative',
                        mb: 1,
                        '&:last-child': { mb: 0 },
                        '&:hover .copy-button': {
                          opacity: 1,
                        },
                      }}
                    >
                      <Box sx={{ 
                        border: '1px solid rgba(0, 0, 0, 0.1)',
                        borderRadius: 1,
                        overflow: 'hidden'
                      }}>
                        <SyntaxHighlighter
                          language={language === 'sql' ? 'sql' : language || 'text'}
                          style={oneLight}
                          customStyle={{
                            margin: 0,
                            fontSize: '0.8em',
                            lineHeight: 1.3,
                            maxHeight: '300px',
                            overflow: 'auto'
                          }}
                          wrapLongLines
                        >
                          {String(children).replace(/\n$/, '')}
                        </SyntaxHighlighter>
                      </Box>
                      
                      <Tooltip title="Copy code">
                        <IconButton
                          className="copy-button"
                          onClick={() => copyToClipboard(String(children))}
                          sx={{
                            position: 'absolute',
                            top: 8,
                            right: 8,
                            opacity: 0,
                            transition: 'opacity 0.2s',
                            backgroundColor: 'rgba(255, 255, 255, 0.9)',
                            '&:hover': {
                              backgroundColor: 'rgba(255, 255, 255, 1)',
                            },
                            width: 32,
                            height: 32,
                          }}
                          size="small"
                        >
                          <ContentCopyIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                    </Box>
                  );
                },
                ul: ({ children }) => (
                  <Box component="ul" sx={{ pl: 2, mb: 1, '&:last-child': { mb: 0 } }}>
                    {children}
                  </Box>
                ),
                ol: ({ children }) => (
                  <Box component="ol" sx={{ pl: 2, mb: 1, '&:last-child': { mb: 0 } }}>
                    {children}
                  </Box>
                ),
                li: ({ children }) => (
                  <Typography component="li" variant="body2" sx={{ mb: 0.5 }}>
                    {children}
                  </Typography>
                ),
                h1: ({ children }) => (
                  <Typography variant="h5" component="h1" sx={{ mb: 1, mt: 2, '&:first-of-type': { mt: 0 } }}>
                    {children}
                  </Typography>
                ),
                h2: ({ children }) => (
                  <Typography variant="h6" component="h2" sx={{ mb: 1, mt: 2, '&:first-of-type': { mt: 0 } }}>
                    {children}
                  </Typography>
                ),
                h3: ({ children }) => (
                  <Typography variant="subtitle1" component="h3" sx={{ mb: 1, mt: 1.5, fontWeight: 'bold' }}>
                    {children}
                  </Typography>
                ),
                blockquote: ({ children }) => (
                  <Box
                    sx={{
                      borderLeft: '4px solid',
                      borderLeftColor: 'primary.main',
                      pl: 2,
                      ml: 1,
                      mb: 1,
                      '&:last-child': { mb: 0 },
                      fontStyle: 'italic',
                      color: 'text.secondary',
                    }}
                  >
                    {children}
                  </Box>
                ),
                strong: ({ children }) => (
                  <Typography component="strong" sx={{ fontWeight: 'bold' }}>
                    {children}
                  </Typography>
                ),
                em: ({ children }) => (
                  <Typography component="em" sx={{ fontStyle: 'italic' }}>
                    {children}
                  </Typography>
                ),
              }}
            >
              {analysisResult}
            </ReactMarkdown>
            {loading && (
              <Box sx={{ display: 'flex', alignItems: 'center', mt: 2, color: 'text.secondary' }}>
                <CircularProgress size={16} sx={{ mr: 1 }} />
                <Typography variant="caption">
                  Analysis in progress...
                </Typography>
              </Box>
            )}
          </Box>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={handleClose} color="primary" variant="contained">
          Close
        </Button>
      </DialogActions>
    </Dialog>
  );
}; 