import React, { useState, useEffect, useRef, useCallback } from 'react';
import {
  Box,
  Paper,
  Typography,
  TextField,
  IconButton,
  List,
  ListItem,
  Fab,
  Drawer,
  AppBar,
  Toolbar,
  CircularProgress,
  Button,
  Chip,
  Tooltip,
} from '@mui/material';
import {
  Send as SendIcon,
  Chat as ChatIcon,
  Close as CloseIcon,
  Clear as ClearIcon,
  Refresh as RefreshIcon,
  Settings as SettingsIcon,
} from '@mui/icons-material';
import { aiChatAPI, type ChatMessage, type StreamMessage, type MCPTool, type FileAttachment, type ToolCall, type ToolApprovalResponse } from '../../api/aichat';
import { ChatMessageComponent } from './ChatMessage';
import { MCPToolsDialog } from './MCPToolsDialog';
import { FileUpload, FileUploadButton } from './FileUpload';

interface AIChatWidgetProps {
  defaultOpen?: boolean;
  position?: 'bottom-right' | 'bottom-left';
  maxWidth?: number;
  maxHeight?: number;
}

export const AIChatWidget: React.FC<AIChatWidgetProps> = ({
  defaultOpen = false,
  position = 'bottom-right',
  maxWidth = 400,
  maxHeight = 600,
}) => {
  const [isOpen, setIsOpen] = useState(defaultOpen);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [inputMessage, setInputMessage] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [sessionId] = useState(`session_${Date.now()}`);
  const [availableTools, setAvailableTools] = useState<MCPTool[]>([]);
  const [showTools, setShowTools] = useState(false);
  const [streamingMessage, setStreamingMessage] = useState('');
  const [attachments, setAttachments] = useState<FileAttachment[]>([]);

  
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const streamCleanupRef = useRef<(() => void) | null>(null);
  const streamingContentRef = useRef<string>('');
  const fileUploadTriggerRef = useRef<(() => void) | null>(null);

  // Auto-scroll to bottom when new messages arrive
  const scrollToBottom = useCallback(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, []);

  useEffect(() => {
    scrollToBottom();
  }, [messages, streamingMessage, scrollToBottom]);

  // Load chat history and available tools on mount
  useEffect(() => {
    loadChatHistory();
    loadAvailableTools();
  }, []);

  const loadChatHistory = async () => {
    try {
      const history = await aiChatAPI.getHistory(sessionId);
      console.log('🔍 LoadChatHistory: Received history:', history);
      console.log('🔍 LoadChatHistory: Messages with attachments:', 
        history.messages.filter(msg => msg.attachments && msg.attachments.length > 0));
      
      // Process tool approval messages to add approval request info
      const processedMessages = history.messages.map(message => {
        if (message.role === 'tool_approval' && message.tool_calls) {
          return {
            ...message,
            approval_request: {
              request_id: `historical_${message.id}`, // Use message ID for historical approvals
              tool_calls: message.tool_calls,
            },
          };
        }
        return message;
      });
      
      setMessages(processedMessages);
    } catch (error) {
      console.error('Failed to load chat history:', error);
    }
  };

  const loadAvailableTools = async () => {
    try {
      const toolsResponse = await aiChatAPI.getMCPTools();
      setAvailableTools(toolsResponse.tools);
    } catch (error) {
      console.error('Failed to load available tools:', error);
    }
  };

  const handleSendMessage = async () => {
    if (!inputMessage.trim() || isLoading) return;

    const userMessage = inputMessage.trim();
    const currentAttachments = [...attachments];
    
    setInputMessage('');
    setAttachments([]); // Clear attachments
    setIsLoading(true);
    setStreamingMessage('');
    streamingContentRef.current = ''; // Reset streaming content ref

    // Add user message to UI immediately
    const newUserMessage: ChatMessage = {
      id: `user_${Date.now()}`,
      role: 'user',
      content: userMessage,
      timestamp: new Date().toISOString(),
      attachments: currentAttachments,
    };
    setMessages(prev => [...prev, newUserMessage]);

    try {
      // Use file upload endpoint if attachments are present
      if (currentAttachments.length > 0) {
        // For file uploads, use non-streaming API for now
        const response = await aiChatAPI.sendMessageWithFiles({
          message: userMessage,
          session_id: sessionId,
          attachments: currentAttachments,
        });

        if (response.error) {
          throw new Error(response.error);
        }

        if (response.message) {
          setMessages(prev => [...prev, response.message!]);
        }
        
        setIsLoading(false);
      } else {
        // Start streaming response for regular messages
        const cleanup = aiChatAPI.streamChat(
          sessionId,
          userMessage,
          (streamMessage: StreamMessage) => {
            if (streamMessage.type === 'message') {
              const newContent = streamMessage.content || '';
              streamingContentRef.current += newContent; // Update ref
              setStreamingMessage(streamingContentRef.current); // Update state
            } else if (streamMessage.type === 'tool_approval_request') {
              // Handle tool approval request
              console.log('Tool approval request:', streamMessage.tool_calls, 'Request ID:', streamMessage.request_id);
              if (streamMessage.tool_calls && streamMessage.tool_calls.length > 0 && streamMessage.request_id) {
                // Add tool approval message to chat
                const approvalMessage: ChatMessage = {
                  id: `approval_${Date.now()}`,
                  role: 'tool_approval',
                  content: `🔧 The assistant wants to execute ${streamMessage.tool_calls.length} tool(s). Please approve or deny the request.`,
                  timestamp: new Date().toISOString(),
                  tool_calls: streamMessage.tool_calls,
                  approval_request: {
                    request_id: streamMessage.request_id, // Use actual request ID from backend
                    tool_calls: streamMessage.tool_calls,
                  },
                };
                
                setMessages(prev => [...prev, approvalMessage]);
                
                // Clear streaming message since we're adding the approval as a separate message
                setStreamingMessage('');
                streamingContentRef.current = '';
              }
            } else if (streamMessage.type === 'tool_execution') {
              // Handle tool execution information
              console.log('Tool executions:', streamMessage.tool_executions);
              if (streamMessage.tool_executions && streamMessage.tool_executions.length > 0) {
                // Add detailed tool execution results to streaming content
                let toolContent = '\n\n**Tool Execution Results:**\n\n';
                streamMessage.tool_executions.forEach(exec => {
                  toolContent += `🔧 **${exec.tool_name}** (${exec.duration_ms}ms)\n`;
                  if (exec.error) {
                    toolContent += `❌ Error: ${exec.error}\n\n`;
                  } else if (exec.result) {
                    const resultPreview = exec.result.length > 300 ? 
                      exec.result.substring(0, 300) + '...' : 
                      exec.result;
                    toolContent += `✅ Result:\n\`\`\`\n${resultPreview}\n\`\`\`\n\n`;
                  }
                });
                streamingContentRef.current += toolContent;
                setStreamingMessage(streamingContentRef.current);
              }
            } else if (streamMessage.type === 'error') {
              console.error('Stream error:', streamMessage.error);
              const errorMsg = 'Error: ' + streamMessage.error;
              streamingContentRef.current = errorMsg;
              setStreamingMessage(errorMsg);
            }
          },
          (error: string) => {
            console.error('Stream error:', error);
            const errorMsg = 'Connection error: ' + error;
            streamingContentRef.current = errorMsg;
            setStreamingMessage(errorMsg);
          },
          () => {
            // Stream completed - use ref value (always current)
            const finalContent = streamingContentRef.current;
            if (finalContent) {
              // Add the final streaming content as an assistant message
              const assistantMessage: ChatMessage = {
                id: `assistant_${Date.now()}`,
                role: 'assistant',
                content: finalContent,
                timestamp: new Date().toISOString(),
              };
              setMessages(prev => [...prev, assistantMessage]);
            }
            
            // Clear any processed approval requests to remove "Processing request..." status
            setMessages(prevMessages => 
              prevMessages.map(msg => 
                msg.approval_request?.processed 
                  ? { ...msg, approval_request: undefined }
                  : msg
              )
            );
            
            setStreamingMessage('');
            streamingContentRef.current = '';
            setIsLoading(false);
            streamCleanupRef.current = null;
          }
        );
        
        streamCleanupRef.current = cleanup;
      }
    } catch (error) {
      console.error('Error sending message:', error);
      setIsLoading(false);
      setStreamingMessage('');
      streamingContentRef.current = '';
    }
  };

  const handleClearHistory = async () => {
    try {
      await aiChatAPI.clearHistory(sessionId);
      setMessages([]);
      setStreamingMessage('');
      setAttachments([]);
      streamingContentRef.current = '';
    } catch (error) {
      console.error('Failed to clear history:', error);
    }
  };

  const handleKeyPress = (event: React.KeyboardEvent) => {
    if (event.key === 'Enter' && !event.shiftKey) {
      event.preventDefault();
      handleSendMessage();
    }
  };

  const handleClose = () => {
    // Clean up any active streams
    if (streamCleanupRef.current) {
      streamCleanupRef.current();
      streamCleanupRef.current = null;
    }
    setIsOpen(false);
  };

  const handleToolApproval = (requestId: string, approvedIds?: string[]) => {
    console.log('Tool approval:', requestId, approvedIds);
    
    // Mark this approval request as processed to hide buttons
    setMessages(prevMessages => 
      prevMessages.map(msg => 
        msg.approval_request?.request_id === requestId 
          ? { ...msg, approval_request: { ...msg.approval_request, processed: true } }
          : msg
      )
    );
    
    // Send approval as a special chat message
    const approvalMessage = `[APPROVE_TOOLS:${requestId}]`;
    
    setIsLoading(true);
    setStreamingMessage('');
    streamingContentRef.current = '';

    // Use the same streaming chat endpoint
    const cleanup = aiChatAPI.streamChat(
      sessionId,
      approvalMessage,
      (streamMessage: StreamMessage) => {
        if (streamMessage.type === 'message') {
          const newContent = streamMessage.content || '';
          streamingContentRef.current += newContent;
          setStreamingMessage(streamingContentRef.current);
        } else if (streamMessage.type === 'tool_execution') {
          console.log('Tool executions during approval:', streamMessage.tool_executions);
          if (streamMessage.tool_executions && streamMessage.tool_executions.length > 0) {
            // Add detailed tool execution results to streaming content
            let toolContent = '\n\n**Tool Execution Results:**\n\n';
            streamMessage.tool_executions.forEach(exec => {
              toolContent += `🔧 **${exec.tool_name}** (${exec.duration_ms}ms)\n`;
              if (exec.error) {
                toolContent += `❌ Error: ${exec.error}\n\n`;
              } else if (exec.result) {
                const resultPreview = exec.result.length > 300 ? 
                  exec.result.substring(0, 300) + '...' : 
                  exec.result;
                toolContent += `✅ Result:\n\`\`\`\n${resultPreview}\n\`\`\`\n\n`;
              }
            });
            streamingContentRef.current += toolContent;
            setStreamingMessage(streamingContentRef.current);
          }
        } else if (streamMessage.type === 'error') {
          console.error('Approval stream error:', streamMessage.error);
          const errorMsg = 'Error: ' + streamMessage.error;
          streamingContentRef.current = errorMsg;
          setStreamingMessage(errorMsg);
        }
      },
      (error: string) => {
        console.error('Approval stream error:', error);
        const errorMsg = 'Connection error: ' + error;
        streamingContentRef.current = errorMsg;
        setStreamingMessage(errorMsg);
      },
      () => {
        // Stream completed - add final content as assistant message
        const finalContent = streamingContentRef.current;
        if (finalContent) {
          const assistantMessage: ChatMessage = {
            id: `assistant_${Date.now()}`,
            role: 'assistant',
            content: finalContent,
            timestamp: new Date().toISOString(),
          };
          setMessages(prev => [...prev, assistantMessage]);
        }
        
        // Clear any processed approval requests to remove "Processing request..." status
        setMessages(prevMessages => 
          prevMessages.map(msg => 
            msg.approval_request?.processed 
              ? { ...msg, approval_request: undefined }
              : msg
          )
        );
        
        setStreamingMessage('');
        streamingContentRef.current = '';
        setIsLoading(false);
      }
    );

    // Store cleanup function
    streamCleanupRef.current = cleanup;
  };

  const handleToolDenial = (requestId: string) => {
    console.log('Tool denial:', requestId);
    
    // Mark this approval request as processed to hide buttons
    setMessages(prevMessages => 
      prevMessages.map(msg => 
        msg.approval_request?.request_id === requestId 
          ? { ...msg, approval_request: { ...msg.approval_request, processed: true } }
          : msg
      )
    );
    
    // Send denial as a special chat message
    const denialMessage = `[DENY_TOOLS:${requestId}]`;
    
    setIsLoading(true);
    setStreamingMessage('');
    streamingContentRef.current = '';

    // Use the same streaming chat endpoint
    const cleanup = aiChatAPI.streamChat(
      sessionId,
      denialMessage,
      (streamMessage: StreamMessage) => {
        if (streamMessage.type === 'message') {
          const newContent = streamMessage.content || '';
          streamingContentRef.current += newContent;
          setStreamingMessage(streamingContentRef.current);
        } else if (streamMessage.type === 'error') {
          console.error('Denial stream error:', streamMessage.error);
          const errorMsg = 'Error: ' + streamMessage.error;
          streamingContentRef.current = errorMsg;
          setStreamingMessage(errorMsg);
        }
      },
      (error: string) => {
        console.error('Denial stream error:', error);
        const errorMsg = 'Connection error: ' + error;
        streamingContentRef.current = errorMsg;
        setStreamingMessage(errorMsg);
      },
      () => {
        // Stream completed - add final content as assistant message
        const finalContent = streamingContentRef.current;
        if (finalContent) {
          const assistantMessage: ChatMessage = {
            id: `assistant_${Date.now()}`,
            role: 'assistant',
            content: finalContent,
            timestamp: new Date().toISOString(),
          };
          setMessages(prev => [...prev, assistantMessage]);
        }
        
        // Clear any processed approval requests to remove "Processing request..." status
        setMessages(prevMessages => 
          prevMessages.map(msg => 
            msg.approval_request?.processed 
              ? { ...msg, approval_request: undefined }
              : msg
          )
        );
        
        setStreamingMessage('');
        streamingContentRef.current = '';
        setIsLoading(false);
      }
    );

    streamCleanupRef.current = cleanup;
  };

  const positionStyles = {
    'bottom-right': { bottom: 16, right: 16 },
    'bottom-left': { bottom: 16, left: 16 },
  };

  // Floating action button to open chat
  if (!isOpen) {
    return (
      <Fab
        color="primary"
        aria-label="open chat"
        onClick={() => setIsOpen(true)}
        sx={{
          position: 'fixed',
          ...positionStyles[position],
          zIndex: 1000,
        }}
      >
        <ChatIcon />
      </Fab>
    );
  }

  return (
    <Paper
      elevation={8}
      sx={{
        position: 'fixed',
        ...positionStyles[position],
        width: maxWidth,
        height: maxHeight,
        display: 'flex',
        flexDirection: 'column',
        zIndex: 1000,
        overflow: 'hidden',
      }}
    >
      {/* Header */}
      <AppBar position="static" color="primary" elevation={0}>
        <Toolbar variant="dense">
          <ChatIcon sx={{ mr: 1 }} />
          <Typography variant="h6" component="div" sx={{ flexGrow: 1 }}>
            AI Assistant
          </Typography>
          {availableTools.length > 0 && (
            <Tooltip title="Available Tools">
              <IconButton
                color="inherit"
                onClick={() => setShowTools(true)}
                size="small"
              >
                <SettingsIcon />
              </IconButton>
            </Tooltip>
          )}
          <Tooltip title="Refresh Tools">
            <IconButton color="inherit" onClick={loadAvailableTools} size="small">
              <RefreshIcon />
            </IconButton>
          </Tooltip>
          <Tooltip title="Clear History">
            <IconButton color="inherit" onClick={handleClearHistory} size="small">
              <ClearIcon />
            </IconButton>
          </Tooltip>
          <IconButton color="inherit" onClick={handleClose} size="small">
            <CloseIcon />
          </IconButton>
        </Toolbar>
      </AppBar>

      {/* Tools indicator */}
      {availableTools.length > 0 && (
        <Box sx={{ p: 1, borderBottom: '1px solid #e0e0e0' }}>
          <Typography variant="caption" color="textSecondary">
            {availableTools.length} MCP tool{availableTools.length !== 1 ? 's' : ''} available
          </Typography>
        </Box>
      )}

      {/* Messages area */}
      <Box
        sx={{
          flexGrow: 1,
          overflow: 'auto',
          p: 1,
          display: 'flex',
          flexDirection: 'column',
        }}
      >
        {messages.length === 0 && !streamingMessage && (
          <Box
            sx={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
              height: '100%',
              textAlign: 'center',
              p: 2,
            }}
          >
            <ChatIcon sx={{ fontSize: 48, color: 'text.secondary', mb: 2 }} />
            <Typography variant="h6" color="textSecondary" gutterBottom>
              Welcome to AI Assistant
            </Typography>
            <Typography variant="body2" color="textSecondary">
              Ask me anything! I can help with database queries, analysis, and more.
            </Typography>
          </Box>
        )}

        <List sx={{ p: 0 }}>
          {messages.map((message) => (
            <ListItem key={message.id} sx={{ p: 0, mb: 1 }}>
              <ChatMessageComponent 
                message={message} 
                onToolApproval={handleToolApproval}
                onToolDenial={handleToolDenial}
              />
            </ListItem>
          ))}
        </List>

        {/* Streaming message */}
        {streamingMessage && (
          <ListItem sx={{ p: 0, mb: 1 }}>
            <ChatMessageComponent
              message={{
                id: 'streaming',
                role: 'assistant',
                content: streamingMessage,
                timestamp: new Date().toISOString(),
              }}
              isStreaming
            />
          </ListItem>
        )}

        {/* Loading indicator */}
        {isLoading && !streamingMessage && (
          <Box sx={{ display: 'flex', justifyContent: 'center', p: 2 }}>
            <CircularProgress size={24} />
          </Box>
        )}

        <div ref={messagesEndRef} />
      </Box>

      {/* Input area */}
      <Box
        sx={{
          p: 1,
          borderTop: '1px solid #e0e0e0',
        }}
      >
        {/* File upload component (hidden, only for file processing) */}
        <FileUpload
          attachments={attachments}
          onAttachmentsChange={setAttachments}
          triggerRef={fileUploadTriggerRef}
        />
        
        {/* Message input */}
        <Box sx={{ display: 'flex', gap: 1 }}>
          <FileUploadButton
            onClick={() => fileUploadTriggerRef.current?.()}
            disabled={attachments.length >= 5}
            hasAttachments={attachments.length > 0}
            maxFiles={5}
            maxFileSize={10 * 1024 * 1024}
          />
          <TextField
            fullWidth
            multiline
            maxRows={3}
            placeholder="Type your message..."
            value={inputMessage}
            onChange={(e) => setInputMessage(e.target.value)}
            onKeyPress={handleKeyPress}
            disabled={isLoading}
            variant="outlined"
            size="small"
          />
          <IconButton
            color="primary"
            onClick={handleSendMessage}
            disabled={(!inputMessage.trim() && attachments.length === 0) || isLoading}
            sx={{ alignSelf: 'flex-end' }}
          >
            <SendIcon />
          </IconButton>
        </Box>
      </Box>

      {/* MCP Tools Dialog */}
      <MCPToolsDialog
        open={showTools}
        onClose={() => setShowTools(false)}
        tools={availableTools}
      />


    </Paper>
  );
}; 