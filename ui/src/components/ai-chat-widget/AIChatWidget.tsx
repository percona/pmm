import React, { useState, useEffect, useRef, useCallback, forwardRef, useImperativeHandle } from 'react';
import {
  Box,
  Paper,
  Typography,
  TextField,
  IconButton,
  List,
  ListItem,
  Fab,
  AppBar,
  Toolbar,
  CircularProgress,
  Tooltip,
} from '@mui/material';
import {
  Send as SendIcon,
  Chat as ChatIcon,
  Close as CloseIcon,
  DeleteSweep as ClearIcon,
  Refresh as RefreshIcon,
  SmartToy as AIIcon,
  List as ListIcon,
  History as HistoryIcon,
} from '@mui/icons-material';
import { aiChatAPI, type ChatMessage, type StreamMessage, type MCPTool, type FileAttachment, type ToolExecution } from '../../api/aichat';
import { ChatMessageComponent } from './ChatMessage';
import { MCPToolsDialog } from './MCPToolsDialog';
import { FileUpload, FileUploadButton } from './FileUpload';
import { SessionSelector } from './SessionSelector';

interface AIChatWidgetProps {
  defaultOpen?: boolean;
  open?: boolean; // Controlled open state
  position?: 'bottom-right' | 'bottom-left';
  maxWidth?: number;
  maxHeight?: number;
  initialMessage?: string;
  onMessageSent?: () => void;
  onOpenChange?: (open: boolean) => void;
}

export interface AIChatWidgetRef {
  openAndSendMessage: (message: string) => void;
}

export const AIChatWidget = forwardRef<AIChatWidgetRef, AIChatWidgetProps>(({
  defaultOpen = false,
  open,
  position = 'bottom-right',
  maxWidth = 400,
  maxHeight = 600,
  initialMessage,
  onMessageSent,
  onOpenChange,
}, ref) => {
  const [isOpen, setIsOpen] = useState(defaultOpen);

  // Handle controlled open state
  useEffect(() => {
    if (open !== undefined) {
      setIsOpen(open);
    }
  }, [open]);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [inputMessage, setInputMessage] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [sessionId, setSessionId] = useState<string>('');
  const [availableTools, setAvailableTools] = useState<MCPTool[]>([]);
  const [showTools, setShowTools] = useState(false);
  const [streamingMessage, setStreamingMessage] = useState('');
  const [attachments, setAttachments] = useState<FileAttachment[]>([]);
  const [toolsLoading, setToolsLoading] = useState(true);
  const [showSessionSelector, setShowSessionSelector] = useState(false);
  const [sessionInitialized, setSessionInitialized] = useState(false);

  
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

  // Load available tools on component mount (no session creation)
  useEffect(() => {
    loadAvailableTools();
  }, []); // Load tools immediately, no session needed

  // Load chat history when session is selected/changed
  useEffect(() => {
    if (sessionInitialized && sessionId) {
      loadChatHistory();
    }
  }, [sessionId, sessionInitialized]); // Dependencies: sessionId and sessionInitialized

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
              request_id: `${message.id}`, // Use message ID for historical approvals
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

  const loadAvailableTools = async (forceRefresh: boolean = false) => {
    try {
      setToolsLoading(true);
      console.log(`🔧 Loading MCP tools${forceRefresh ? ' (force refresh)' : ''}...`);
      
      // Log the exact URL being called
      const url = forceRefresh ? '/v1/chat/mcp/tools?force_refresh=true' : '/v1/chat/mcp/tools';
      console.log(`🌐 Calling API endpoint: ${url}`);
      
      const toolsResponse = await aiChatAPI.getMCPTools(forceRefresh);
      console.log(`🔧 MCP Tools response:`, toolsResponse);
      setAvailableTools(toolsResponse.tools);
      console.log(`✅ ${toolsResponse.tools.length} MCP tools available:`, toolsResponse.tools.map(t => t.name));
    } catch (error) {
      console.error('❌ Failed to load MCP tools:', error);
      console.error('❌ Error details:', {
        message: error instanceof Error ? error.message : 'Unknown error',
        stack: error instanceof Error ? error.stack : undefined,
      });
      // Set empty array on error to ensure UI updates
      setAvailableTools([]);
    } finally {
      setToolsLoading(false);
    }
  };

  const handleRefreshTools = () => {
    loadAvailableTools(true);
  };

  // Create session on first message (removed automatic initialization)
  const createSessionOnFirstMessage = async (): Promise<string> => {
    console.log('🆕 Creating new session for first message...');
    
    // No need to create session explicitly - backend will create it automatically
    // Just return empty string and let backend handle session creation
    const newSessionId = ''; // Empty session ID triggers backend session creation
    setSessionId(newSessionId);
    setSessionInitialized(true);
    console.log('✅ Session will be created automatically by backend on first message');
    return newSessionId;
  };

  // Session management handlers
  const handleSessionSelect = async (newSessionId: string) => {
    console.log('🔄 Session select starting:', { newSessionId, currentSessionId: sessionId });
    
    setSessionId(newSessionId);
    setMessages([]);
    setStreamingMessage('');
    setAttachments([]);
    streamingContentRef.current = '';
    setShowSessionSelector(false);
    setSessionInitialized(true); // Mark as initialized when selecting existing session
    
    console.log('🔄 Session state cleared, loading history...');
    
    // Load history for the new session
    try {
      const history = await aiChatAPI.getHistory(newSessionId);
      console.log('🔄 Session history loaded:', { sessionId: newSessionId, messageCount: history.messages.length });
      
      const processedMessages = history.messages.map(message => {
        if (message.role === 'tool_approval' && message.tool_calls) {
          return {
            ...message,
            approval_request: {
              request_id: `historical_${message.id}`,
              tool_calls: message.tool_calls,
            },
          };
        }
        return message;
      });
      
      console.log('🔄 Setting processed messages:', processedMessages.length);
      setMessages(processedMessages);
    } catch (error) {
      console.error('❌ Failed to load session history:', error);
    }
  };

  // Unified function to send a message with streaming support
  const sendMessageWithStreaming = useCallback(async (
    messageText: string, 
    messageAttachments: FileAttachment[] = [],
    onComplete?: () => void
  ) => {
    if (!messageText.trim() || isLoading) return;

    const userMessage = messageText.trim();
    
    // If no session exists, create one automatically (session creation on first message)
    let currentSessionId = sessionId;
    if (!sessionInitialized || !currentSessionId) {
      console.log('🆕 No session exists, creating session on first message...');
      currentSessionId = await createSessionOnFirstMessage();
    }
    
    setIsLoading(true);
    setStreamingMessage('');
    streamingContentRef.current = '';

    // Add user message to UI immediately
    const newUserMessage: ChatMessage = {
      id: `user_${Date.now()}`,
      role: 'user',
      content: userMessage,
      timestamp: new Date().toISOString(),
      attachments: messageAttachments,
    };
    setMessages(prev => [...prev, newUserMessage]);

    try {
      // Use file upload endpoint if attachments are present
      if (messageAttachments.length > 0) {
        const response = await aiChatAPI.sendMessageWithFiles({
          message: userMessage,
          session_id: currentSessionId,
          attachments: messageAttachments,
        });

        if (response.error) {
          throw new Error(response.error);
        }

        if (response.message) {
          setMessages(prev => [...prev, response.message!]);
          
          // Update session ID if backend created a new one
          if (response.session_id && response.session_id !== currentSessionId) {
            console.log('🔄 Backend created new session:', response.session_id);
            setSessionId(response.session_id);
            currentSessionId = response.session_id;
          }
        }
        
        setIsLoading(false);
        onComplete?.();
      } else {
        // Start streaming response for regular messages
        const cleanup = await aiChatAPI.streamChatWithSeparateEndpoints(
          currentSessionId,
          userMessage,
          (streamMessage: StreamMessage) => {
            // Update session ID if backend provides a new one in stream messages
            if (streamMessage.session_id && streamMessage.session_id !== currentSessionId) {
              console.log('🔄 Backend created new session via stream:', streamMessage.session_id);
              setSessionId(streamMessage.session_id);
              currentSessionId = streamMessage.session_id;
            }
            
            if (streamMessage.type === 'message') {
              const newContent = streamMessage.content || '';
              streamingContentRef.current += newContent;
              setStreamingMessage(streamingContentRef.current);
            } else if (streamMessage.type === 'tool_approval_request') {
              console.log('Tool approval request:', streamMessage.tool_calls, 'Request ID:', streamMessage.request_id);
              if (streamMessage.tool_calls && streamMessage.tool_calls.length > 0 && streamMessage.request_id) {
                const approvalMessage: ChatMessage = {
                  id: `approval_${Date.now()}`,
                  role: 'tool_approval',
                  content: `🔧 The assistant wants to execute ${streamMessage.tool_calls.length} tool(s). Please approve or deny the request.`,
                  timestamp: new Date().toISOString(),
                  tool_calls: streamMessage.tool_calls,
                  approval_request: {
                    request_id: streamMessage.request_id,
                    tool_calls: streamMessage.tool_calls,
                  },
                };
                setMessages(prev => [...prev, approvalMessage]);
                setStreamingMessage('');
                streamingContentRef.current = '';
              }
            } else if (streamMessage.type === 'tool_execution') {
              console.log('Tool executions:', streamMessage.tool_executions);
              if (streamMessage.tool_executions && streamMessage.tool_executions.length > 0) {
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
              
              // Clear any streaming content since we have an error
              setStreamingMessage('');
              streamingContentRef.current = '';
              
              // Create a proper error message to display in the chat
              const errorMessage: ChatMessage = {
                id: `error_${Date.now()}`,
                role: 'assistant',
                content: `❌ **Error**: ${streamMessage.error}`,
                timestamp: new Date().toISOString(),
              };
              
              setMessages(prev => [...prev, errorMessage]);
              setIsLoading(false);
            }
          },
          (error: string) => {
            console.error('Stream connection error:', error);
            
            // Clear any streaming content since we have an error
            setStreamingMessage('');
            streamingContentRef.current = '';
            
            // Create a proper error message to display in the chat
            const errorMessage: ChatMessage = {
              id: `connection_error_${Date.now()}`,
              role: 'assistant',
              content: `🔌 **Connection Error**: ${error}`,
              timestamp: new Date().toISOString(),
            };
            
            setMessages(prev => [...prev, errorMessage]);
            setIsLoading(false);
            onComplete?.();
          },
          () => {
            // Stream completed - use ref value (always current)
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
            
            // Clear any processed approval requests
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
            onComplete?.();
          }
        );
        
        streamCleanupRef.current = cleanup;
      }
    } catch (error) {
      console.error('Error sending message:', error);
      
      // Create a proper error message to display in the chat
      const errorMessage: ChatMessage = {
        id: `send_error_${Date.now()}`,
        role: 'assistant',
        content: `❌ **Failed to send message**: ${error instanceof Error ? error.message : String(error)}`,
        timestamp: new Date().toISOString(),
      };
      
      setMessages(prev => [...prev, errorMessage]);
      setIsLoading(false);
      setStreamingMessage('');
      streamingContentRef.current = '';
      onComplete?.();
    }
  }, [sessionId, isLoading, sessionInitialized, createSessionOnFirstMessage]);

  // Handle initial message - send it automatically when widget opens
  useEffect(() => {
    if (initialMessage && isOpen) {
      // Don't show the message in the input field, just send it directly
      // Use a timeout to ensure the widget is ready
      const timeoutId = setTimeout(async () => {
        if (!isLoading) {
          await sendMessageWithStreaming(initialMessage, [], () => {
            // Call onMessageSent when the AI response is complete
            onMessageSent?.();
          });
        }
      }, 300);

      return () => clearTimeout(timeoutId);
    }
  }, [initialMessage, isOpen, isLoading, onMessageSent, sendMessageWithStreaming]);

  // Expose methods to parent component
  useImperativeHandle(ref, () => ({
    openAndSendMessage: (message: string) => {
      setIsOpen(true);
      // Don't show the message in the input field, just send it directly
      // Use setTimeout to ensure the widget is open before sending
      setTimeout(() => {
        sendMessageWithStreaming(message);
      }, 200);
    },
  }), [sendMessageWithStreaming]);

  const handleSendMessage = async () => {
    if (!inputMessage.trim() || isLoading) return;

    const userMessage = inputMessage.trim();
    const currentAttachments = [...attachments];
    
    // Clear input and attachments immediately
    setInputMessage('');
    setAttachments([]);

    // Send the message using unified function
    await sendMessageWithStreaming(userMessage, currentAttachments);
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
    onOpenChange?.(false);
  };

  const handleOpen = () => {
    setIsOpen(true);
    onOpenChange?.(true);
  };

  const handleToolApproval = async (requestId: string, approvedIds?: string[]) => {
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
    
    // Store tool executions to preserve parameters
    let collectedToolExecutions: ToolExecution[] = [];

    // Use the same streaming chat endpoint
    const cleanup = await aiChatAPI.streamChatWithSeparateEndpoints(
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
            // Store tool executions to preserve parameters for later display
            collectedToolExecutions = [...collectedToolExecutions, ...streamMessage.tool_executions];
            
            // Add detailed tool execution results to streaming content
            let toolContent = '\n\n**Tool Execution Results:**\n\n';
            streamMessage.tool_executions.forEach(exec => {
              toolContent += `🔧 **${exec.tool_name}** (${exec.duration_ms}ms)\n`;
              if (exec.arguments) {
                try {
                  const args = JSON.parse(exec.arguments);
                  toolContent += `📝 Parameters: \`${JSON.stringify(args)}\`\n`;
                } catch {
                  toolContent += `📝 Parameters: \`${exec.arguments}\`\n`;
                }
              }
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
          
          // Clear any streaming content since we have an error
          setStreamingMessage('');
          streamingContentRef.current = '';
          
          // Create a proper error message to display in the chat
          const errorMessage: ChatMessage = {
            id: `approval_error_${Date.now()}`,
            role: 'assistant',
            content: `❌ **Tool Approval Error**: ${streamMessage.error}`,
            timestamp: new Date().toISOString(),
          };
          
          setMessages(prev => [...prev, errorMessage]);
          setIsLoading(false);
        }
      },
      (error: string) => {
        console.error('Approval stream connection error:', error);
        
        // Clear any streaming content since we have an error
        setStreamingMessage('');
        streamingContentRef.current = '';
        
        // Create a proper error message to display in the chat
        const errorMessage: ChatMessage = {
          id: `approval_connection_error_${Date.now()}`,
          role: 'assistant',
          content: `🔌 **Tool Approval Connection Error**: ${error}`,
          timestamp: new Date().toISOString(),
        };
        
        setMessages(prev => [...prev, errorMessage]);
        setIsLoading(false);
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
            // tool_executions: collectedToolExecutions.length > 0 ? collectedToolExecutions : undefined,
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

  const handleToolDenial = async (requestId: string) => {
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
    const cleanup = await aiChatAPI.streamChatWithSeparateEndpoints(
      sessionId,
      denialMessage,
      (streamMessage: StreamMessage) => {
        if (streamMessage.type === 'message') {
          const newContent = streamMessage.content || '';
          streamingContentRef.current += newContent;
          setStreamingMessage(streamingContentRef.current);
        } else if (streamMessage.type === 'error') {
          console.error('Denial stream error:', streamMessage.error);
          
          // Clear any streaming content since we have an error
          setStreamingMessage('');
          streamingContentRef.current = '';
          
          // Create a proper error message to display in the chat
          const errorMessage: ChatMessage = {
            id: `denial_error_${Date.now()}`,
            role: 'assistant',
            content: `❌ **Tool Denial Error**: ${streamMessage.error}`,
            timestamp: new Date().toISOString(),
          };
          
          setMessages(prev => [...prev, errorMessage]);
          setIsLoading(false);
        }
      },
      (error: string) => {
        console.error('Denial stream connection error:', error);
        
        // Clear any streaming content since we have an error
        setStreamingMessage('');
        streamingContentRef.current = '';
        
        // Create a proper error message to display in the chat
        const errorMessage: ChatMessage = {
          id: `denial_connection_error_${Date.now()}`,
          role: 'assistant',
          content: `🔌 **Tool Denial Connection Error**: ${error}`,
          timestamp: new Date().toISOString(),
        };
        
        setMessages(prev => [...prev, errorMessage]);
        setIsLoading(false);
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
        onClick={handleOpen}
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
        <Toolbar variant="dense" sx={{ minHeight: 56 }}>
          <AIIcon sx={{ mr: 1, fontSize: 28, color: '#ffffff' }} />
          <Typography variant="h6" component="div" sx={{ flexGrow: 1, fontWeight: 600, color: '#ffffff' }}>
            AI Assistant
          </Typography>
          
          {/* Action buttons with better spacing and visibility */}
          <Box sx={{ display: 'flex', gap: 0.5 }}>
            <Tooltip title={
              toolsLoading 
                ? "Loading MCP Tools..." 
                : availableTools.length > 0 
                  ? `View MCP Tools List (${availableTools.length} available)` 
                  : "No MCP Tools Available"
            } arrow>
              <IconButton
                onClick={() => setShowTools(true)}
                size="medium"
                disabled={toolsLoading || availableTools.length === 0}
                sx={{ 
                  backgroundColor: !toolsLoading && availableTools.length > 0 
                    ? 'rgba(255, 255, 255, 0.15)' 
                    : 'rgba(255, 255, 255, 0.05)',
                  '&:hover': !toolsLoading && availableTools.length > 0 ? { 
                    backgroundColor: 'rgba(255, 255, 255, 0.25)',
                    transform: 'scale(1.05)',
                  } : {},
                  borderRadius: 2,
                  border: '1px solid rgba(255, 255, 255, 0.2)',
                  transition: 'all 0.2s ease-in-out',
                  opacity: !toolsLoading && availableTools.length > 0 ? 1 : 0.5,
                }}
              >
                {toolsLoading ? (
                  <CircularProgress size={20} sx={{ color: '#ffffff' }} />
                ) : (
                  <ListIcon sx={{ 
                    fontSize: 20, 
                    color: availableTools.length > 0 ? '#ffffff' : 'rgba(255, 255, 255, 0.5)', 
                    fontWeight: 'bold' 
                  }} />
                )}
              </IconButton>
            </Tooltip>
            
            <Tooltip title="Refresh MCP Tools" arrow>
              <IconButton 
                onClick={handleRefreshTools} 
                size="medium"
                sx={{ 
                  backgroundColor: 'rgba(255, 255, 255, 0.15)',
                  '&:hover': { 
                    backgroundColor: 'rgba(255, 255, 255, 0.25)',
                    transform: 'scale(1.05)',
                  },
                  borderRadius: 2,
                  border: '1px solid rgba(255, 255, 255, 0.2)',
                  transition: 'all 0.2s ease-in-out',
                }}
              >
                <RefreshIcon sx={{ fontSize: 20, color: '#ffffff', fontWeight: 'bold' }} />
              </IconButton>
            </Tooltip>
            
            <Tooltip title="Chat Sessions" arrow>
              <IconButton 
                onClick={() => setShowSessionSelector(true)} 
                size="medium"
                sx={{ 
                  backgroundColor: 'rgba(255, 255, 255, 0.15)',
                  '&:hover': { 
                    backgroundColor: 'rgba(255, 255, 255, 0.25)',
                    transform: 'scale(1.05)',
                  },
                  borderRadius: 2,
                  border: '1px solid rgba(255, 255, 255, 0.2)',
                  transition: 'all 0.2s ease-in-out',
                }}
              >
                <HistoryIcon sx={{ fontSize: 20, color: '#ffffff', fontWeight: 'bold' }} />
              </IconButton>
            </Tooltip>
            
            <Tooltip title="Clear Chat History" arrow>
              <IconButton 
                onClick={handleClearHistory} 
                size="medium"
                sx={{ 
                  backgroundColor: 'rgba(255, 255, 255, 0.15)',
                  '&:hover': { 
                    backgroundColor: 'rgba(255, 255, 255, 0.25)',
                    transform: 'scale(1.05)',
                  },
                  borderRadius: 2,
                  border: '1px solid rgba(255, 255, 255, 0.2)',
                  transition: 'all 0.2s ease-in-out',
                }}
              >
                <ClearIcon sx={{ fontSize: 20, color: '#ffffff', fontWeight: 'bold' }} />
              </IconButton>
            </Tooltip>
            
            <Tooltip title="Close Chat" arrow>
              <IconButton 
                onClick={handleClose} 
                size="medium"
                sx={{ 
                  backgroundColor: 'rgba(255, 255, 255, 0.15)',
                  '&:hover': { 
                    backgroundColor: 'rgba(255, 255, 255, 0.25)',
                    transform: 'scale(1.05)',
                  },
                  borderRadius: 2,
                  border: '1px solid rgba(255, 255, 255, 0.2)',
                  transition: 'all 0.2s ease-in-out',
                  ml: 0.5,
                }}
              >
                <CloseIcon sx={{ fontSize: 20, color: '#ffffff', fontWeight: 'bold' }} />
              </IconButton>
            </Tooltip>
          </Box>
        </Toolbar>
      </AppBar>

      {/* Tools indicator */}
      {(toolsLoading || availableTools.length > 0) && (
        <Box sx={{ 
          p: 1.5, 
          borderBottom: '1px solid #e0e0e0',
          backgroundColor: toolsLoading ? '#fff3e0' : '#f0f7ff',
          display: 'flex',
          alignItems: 'center',
          gap: 1,
        }}>
          {toolsLoading ? (
            <>
              <CircularProgress size={18} sx={{ color: '#ff9800' }} />
              <Typography variant="body2" sx={{ fontWeight: 600, color: '#f57c00' }}>
                Loading MCP tools...
              </Typography>
            </>
          ) : (
            <>
              <ListIcon sx={{ fontSize: 18, color: '#1976d2', fontWeight: 'bold' }} />
              <Typography variant="body2" sx={{ fontWeight: 600, color: '#1565c0' }}>
                {availableTools.length} MCP tool{availableTools.length !== 1 ? 's' : ''} available
              </Typography>
            </>
          )}
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
            {!sessionId && (
              <Typography variant="caption" color="textSecondary" sx={{ mt: 1, fontStyle: 'italic' }}>
                Your session will be created when you send your first message
              </Typography>
            )}
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

      {/* Session Selector Dialog */}
      <SessionSelector
        open={showSessionSelector}
        onClose={() => setShowSessionSelector(false)}
        onSessionSelect={handleSessionSelect}
        currentSessionId={sessionId}
      />

    </Paper>
  );
}); 