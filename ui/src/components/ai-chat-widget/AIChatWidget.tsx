import React, { useEffect, useRef, useCallback, forwardRef, useImperativeHandle, useReducer } from 'react';
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

// State interface for useReducer
interface ChatState {
  isOpen: boolean;
  messages: ChatMessage[];
  inputMessage: string;
  isLoading: boolean;
  sessionId: string;
  availableTools: MCPTool[];
  showTools: boolean;
  streamingMessage: string;
  attachments: FileAttachment[];
  toolsLoading: boolean;
  showSessionSelector: boolean;
  sessionInitialized: boolean;
}

// Action types for the reducer
type ChatAction =
  | { type: 'SET_IS_OPEN'; payload: boolean }
  | { type: 'SET_MESSAGES'; payload: ChatMessage[] }
  | { type: 'ADD_MESSAGE'; payload: ChatMessage }
  | { type: 'UPDATE_MESSAGES'; payload: (messages: ChatMessage[]) => ChatMessage[] }
  | { type: 'SET_INPUT_MESSAGE'; payload: string }
  | { type: 'SET_IS_LOADING'; payload: boolean }
  | { type: 'SET_SESSION_ID'; payload: string }
  | { type: 'SET_AVAILABLE_TOOLS'; payload: MCPTool[] }
  | { type: 'SET_SHOW_TOOLS'; payload: boolean }
  | { type: 'SET_STREAMING_MESSAGE'; payload: string }
  | { type: 'SET_ATTACHMENTS'; payload: FileAttachment[] }
  | { type: 'SET_TOOLS_LOADING'; payload: boolean }
  | { type: 'SET_SHOW_SESSION_SELECTOR'; payload: boolean }
  | { type: 'SET_SESSION_INITIALIZED'; payload: boolean }
  | { type: 'CLEAR_CHAT_STATE' }
  | { type: 'RESET_SESSION_STATE'; payload: { sessionId: string; messages: ChatMessage[] } };

// Initial state
const createInitialState = (defaultOpen: boolean): ChatState => ({
  isOpen: defaultOpen,
  messages: [],
  inputMessage: '',
  isLoading: false,
  sessionId: '',
  availableTools: [],
  showTools: false,
  streamingMessage: '',
  attachments: [],
  toolsLoading: true,
  showSessionSelector: false,
  sessionInitialized: false,
});

// Reducer function
const chatReducer = (state: ChatState, action: ChatAction): ChatState => {
  switch (action.type) {
    case 'SET_IS_OPEN':
      return { ...state, isOpen: action.payload };
    case 'SET_MESSAGES':
      return { ...state, messages: action.payload };
    case 'ADD_MESSAGE':
      return { ...state, messages: [...state.messages, action.payload] };
    case 'UPDATE_MESSAGES':
      return { ...state, messages: action.payload(state.messages) };
    case 'SET_INPUT_MESSAGE':
      return { ...state, inputMessage: action.payload };
    case 'SET_IS_LOADING':
      return { ...state, isLoading: action.payload };
    case 'SET_SESSION_ID':
      return { ...state, sessionId: action.payload };
    case 'SET_AVAILABLE_TOOLS':
      return { ...state, availableTools: action.payload };
    case 'SET_SHOW_TOOLS':
      return { ...state, showTools: action.payload };
    case 'SET_STREAMING_MESSAGE':
      return { ...state, streamingMessage: action.payload };
    case 'SET_ATTACHMENTS':
      return { ...state, attachments: action.payload };
    case 'SET_TOOLS_LOADING':
      return { ...state, toolsLoading: action.payload };
    case 'SET_SHOW_SESSION_SELECTOR':
      return { ...state, showSessionSelector: action.payload };
    case 'SET_SESSION_INITIALIZED':
      return { ...state, sessionInitialized: action.payload };
    case 'CLEAR_CHAT_STATE':
      return {
        ...state,
        messages: [],
        streamingMessage: '',
        attachments: [],
        showSessionSelector: false,
        sessionInitialized: true,
      };
    case 'RESET_SESSION_STATE':
      return {
        ...state,
        sessionId: action.payload.sessionId,
        messages: action.payload.messages,
        streamingMessage: '',
        attachments: [],
        showSessionSelector: false,
        sessionInitialized: true,
      };
    default:
      return state;
  }
};

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
  const [state, dispatch] = useReducer(chatReducer, createInitialState(defaultOpen));

  // Handle controlled open state
  useEffect(() => {
    if (open !== undefined) {
      dispatch({ type: 'SET_IS_OPEN', payload: open });
    }
  }, [open]);

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
  }, [state.messages, state.streamingMessage, scrollToBottom]);

  // Load available tools on component mount (no session creation)
  useEffect(() => {
    loadAvailableTools();
  }, []); // Load tools immediately, no session needed

  // Load chat history when session is selected/changed
  useEffect(() => {
    if (state.sessionInitialized && state.sessionId) {
      loadChatHistory();
    }
  }, [state.sessionId, state.sessionInitialized]); // Dependencies: sessionId and sessionInitialized

  const loadChatHistory = async () => {
    try {
      const history = await aiChatAPI.getHistory(state.sessionId);
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
      
      dispatch({ type: 'SET_MESSAGES', payload: processedMessages });
    } catch (error) {
      console.error('Failed to load chat history:', error);
    }
  };

  const loadAvailableTools = async (forceRefresh: boolean = false) => {
    try {
      dispatch({ type: 'SET_TOOLS_LOADING', payload: true });
      console.log(`🔧 Loading MCP tools${forceRefresh ? ' (force refresh)' : ''}...`);
      
      // Log the exact URL being called
      const url = forceRefresh ? '/v1/chat/mcp/tools?force_refresh=true' : '/v1/chat/mcp/tools';
      console.log(`🌐 Calling API endpoint: ${url}`);
      
      const toolsResponse = await aiChatAPI.getMCPTools(forceRefresh);
      console.log(`🔧 MCP Tools response:`, toolsResponse);
      dispatch({ type: 'SET_AVAILABLE_TOOLS', payload: toolsResponse.tools });
      console.log(`✅ ${toolsResponse.tools.length} MCP tools available:`, toolsResponse.tools.map(t => t.name));
    } catch (error) {
      console.error('❌ Failed to load MCP tools:', error);
      console.error('❌ Error details:', {
        message: error instanceof Error ? error.message : 'Unknown error',
        stack: error instanceof Error ? error.stack : undefined,
      });
      // Set empty array on error to ensure UI updates
      dispatch({ type: 'SET_AVAILABLE_TOOLS', payload: [] });
    } finally {
      dispatch({ type: 'SET_TOOLS_LOADING', payload: false });
    }
  };

  const handleRefreshTools = () => {
    loadAvailableTools(true);
  };

  // Create session on first message (removed automatic initialization)
  const createSessionOnFirstMessage = useCallback(async (): Promise<string> => {
    console.log('🆕 Creating new session for first message...');
    
    // No need to create session explicitly - backend will create it automatically
    // Just return empty string and let backend handle session creation
    const newSessionId = ''; // Empty session ID triggers backend session creation
    dispatch({ type: 'SET_SESSION_ID', payload: newSessionId });
    dispatch({ type: 'SET_SESSION_INITIALIZED', payload: true });
    console.log('✅ Session will be created automatically by backend on first message');
    return newSessionId;
  }, []);

  // Session management handlers
  const handleSessionSelect = async (newSessionId: string) => {
    console.log('🔄 Session select starting:', { newSessionId, currentSessionId: state.sessionId });
    
    dispatch({ type: 'CLEAR_CHAT_STATE' });
    streamingContentRef.current = '';
    
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
      dispatch({ type: 'RESET_SESSION_STATE', payload: { sessionId: newSessionId, messages: processedMessages } });
    } catch (error) {
      console.error('❌ Failed to load session history:', error);
    }
  };

  // Helper function to ensure session exists
  const ensureSession = useCallback(async (): Promise<string> => {
    let currentSessionId = state.sessionId;
    if (!state.sessionInitialized || !currentSessionId) {
      console.log('🆕 No session exists, creating session on first message...');
      currentSessionId = await createSessionOnFirstMessage();
    }
    return currentSessionId;
  }, [state.sessionId, state.sessionInitialized, createSessionOnFirstMessage]);

  // Helper function to add user message to UI
  const addUserMessage = useCallback((messageText: string, attachments: FileAttachment[] = []): ChatMessage => {
    const newUserMessage: ChatMessage = {
      id: `user_${Date.now()}`,
      role: 'user',
      content: messageText,
      timestamp: new Date().toISOString(),
      attachments: attachments,
    };
    dispatch({ type: 'ADD_MESSAGE', payload: newUserMessage });
    return newUserMessage;
  }, []);

  // Helper function to handle file message uploads
  const handleFileMessage = useCallback(async (
    messageText: string,
    attachments: FileAttachment[],
    sessionId: string
  ): Promise<string> => {
    const response = await aiChatAPI.sendMessageWithFiles({
      message: messageText,
      session_id: sessionId,
      attachments: attachments,
    });

    if (response.error) {
      throw new Error(response.error);
    }

    if (response.message) {
      dispatch({ type: 'ADD_MESSAGE', payload: response.message });
    }

    // Return updated session ID if backend created a new one
    return response.session_id || sessionId;
  }, []);

  // Helper function to handle individual stream messages
  const handleStreamMessage = useCallback((streamMessage: StreamMessage, currentSessionId: string): string => {
    let updatedSessionId = currentSessionId;

    // Update session ID if backend provides a new one in stream messages
    if (streamMessage.session_id && streamMessage.session_id !== currentSessionId) {
      console.log('🔄 Backend created new session via stream:', streamMessage.session_id);
      dispatch({ type: 'SET_SESSION_ID', payload: streamMessage.session_id });
      updatedSessionId = streamMessage.session_id;
    }

    if (streamMessage.type === 'message') {
      const newContent = streamMessage.content || '';
      streamingContentRef.current += newContent;
      dispatch({ type: 'SET_STREAMING_MESSAGE', payload: streamingContentRef.current });
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
        dispatch({ type: 'ADD_MESSAGE', payload: approvalMessage });
        dispatch({ type: 'SET_STREAMING_MESSAGE', payload: '' });
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
        dispatch({ type: 'SET_STREAMING_MESSAGE', payload: streamingContentRef.current });
      }
    } else if (streamMessage.type === 'error') {
      console.error('Stream error:', streamMessage.error);
      
      // Clear any streaming content since we have an error
      dispatch({ type: 'SET_STREAMING_MESSAGE', payload: '' });
      streamingContentRef.current = '';
      
      // Create a proper error message to display in the chat
      const errorMessage: ChatMessage = {
        id: `error_${Date.now()}`,
        role: 'assistant',
        content: `❌ **Error**: ${streamMessage.error}`,
        timestamp: new Date().toISOString(),
      };
      
      dispatch({ type: 'ADD_MESSAGE', payload: errorMessage });
      dispatch({ type: 'SET_IS_LOADING', payload: false });
    }

    return updatedSessionId;
  }, []);

  // Helper function to handle streaming messages
  const handleStreamingMessage = useCallback(async (
    messageText: string,
    sessionId: string,
    onComplete?: () => void
  ): Promise<void> => {
    const cleanup = await aiChatAPI.streamChatWithSeparateEndpoints(
      sessionId,
      messageText,
      (streamMessage: StreamMessage) => {
        handleStreamMessage(streamMessage, sessionId);
      },
      (error: string) => {
        console.error('Stream connection error:', error);
        
        // Clear any streaming content since we have an error
        dispatch({ type: 'SET_STREAMING_MESSAGE', payload: '' });
        streamingContentRef.current = '';
        
        // Create a proper error message to display in the chat
        const errorMessage: ChatMessage = {
          id: `connection_error_${Date.now()}`,
          role: 'assistant',
          content: `🔌 **Connection Error**: ${error}`,
          timestamp: new Date().toISOString(),
        };
        
        dispatch({ type: 'ADD_MESSAGE', payload: errorMessage });
        dispatch({ type: 'SET_IS_LOADING', payload: false });
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
          dispatch({ type: 'ADD_MESSAGE', payload: assistantMessage });
        }
        
        // Clear any processed approval requests
        dispatch({ type: 'UPDATE_MESSAGES', payload: (messages) => 
          messages.map(msg => 
            msg.approval_request?.processed 
              ? { ...msg, approval_request: undefined }
              : msg
          )
        });
        
        dispatch({ type: 'SET_STREAMING_MESSAGE', payload: '' });
        streamingContentRef.current = '';
        dispatch({ type: 'SET_IS_LOADING', payload: false });
        streamCleanupRef.current = null;
        onComplete?.();
      }
    );
    
    streamCleanupRef.current = cleanup;
  }, [handleStreamMessage]);

  // Helper function to handle send errors
  const handleSendError = useCallback((error: unknown, onComplete?: () => void): void => {
    console.error('Error sending message:', error);
    
    // Create a proper error message to display in the chat
    const errorMessage: ChatMessage = {
      id: `send_error_${Date.now()}`,
      role: 'assistant',
      content: `❌ **Failed to send message**: ${error instanceof Error ? error.message : String(error)}`,
      timestamp: new Date().toISOString(),
    };
    
    dispatch({ type: 'ADD_MESSAGE', payload: errorMessage });
    dispatch({ type: 'SET_IS_LOADING', payload: false });
    dispatch({ type: 'SET_STREAMING_MESSAGE', payload: '' });
    streamingContentRef.current = '';
    onComplete?.();
  }, []);

  // Unified function to send a message with streaming support
  const sendMessageWithStreaming = useCallback(async (
    messageText: string, 
    messageAttachments: FileAttachment[] = [],
    onComplete?: () => void
  ) => {
    if (!messageText.trim() || state.isLoading) return;

    const userMessage = messageText.trim();
    
    try {
      // Step 1: Ensure session exists
      const currentSessionId = await ensureSession();
      
      // Step 2: Set loading state and clear streaming content
      dispatch({ type: 'SET_IS_LOADING', payload: true });
      dispatch({ type: 'SET_STREAMING_MESSAGE', payload: '' });
      streamingContentRef.current = '';

      // Step 3: Add user message to UI immediately
      addUserMessage(userMessage, messageAttachments);

      // Step 4: Handle message based on whether it has attachments
      if (messageAttachments.length > 0) {
        await handleFileMessage(userMessage, messageAttachments, currentSessionId);
        dispatch({ type: 'SET_IS_LOADING', payload: false });
        onComplete?.();
      } else {
        await handleStreamingMessage(userMessage, currentSessionId, onComplete);
      }
    } catch (error) {
      handleSendError(error, onComplete);
    }
  }, [state.isLoading, ensureSession, addUserMessage, handleFileMessage, handleStreamingMessage, handleSendError]);

  // Handle initial message - send it automatically when widget opens
  useEffect(() => {
    if (initialMessage && state.isOpen) {
      // Don't show the message in the input field, just send it directly
      // Use a timeout to ensure the widget is ready
      const timeoutId = setTimeout(async () => {
        if (!state.isLoading) {
          await sendMessageWithStreaming(initialMessage, [], () => {
            // Call onMessageSent when the AI response is complete
            onMessageSent?.();
          });
        }
      }, 300);

      return () => clearTimeout(timeoutId);
    }
  }, [initialMessage, state.isOpen, state.isLoading, onMessageSent, sendMessageWithStreaming]);

  // Expose methods to parent component
  useImperativeHandle(ref, () => ({
    openAndSendMessage: (message: string) => {
      dispatch({ type: 'SET_IS_OPEN', payload: true });
      // Don't show the message in the input field, just send it directly
      // Use setTimeout to ensure the widget is open before sending
      setTimeout(() => {
        sendMessageWithStreaming(message);
      }, 200);
    },
  }), [sendMessageWithStreaming]);

  const handleSendMessage = async () => {
    if (!state.inputMessage.trim() || state.isLoading) return;

    const userMessage = state.inputMessage.trim();
    const currentAttachments = [...state.attachments];
    
    // Clear input and attachments immediately
    dispatch({ type: 'SET_INPUT_MESSAGE', payload: '' });
    dispatch({ type: 'SET_ATTACHMENTS', payload: [] });

    // Send the message using unified function
    await sendMessageWithStreaming(userMessage, currentAttachments);
  };

  const handleClearHistory = async () => {
    try {
      await aiChatAPI.clearHistory(state.sessionId);
      dispatch({ type: 'SET_MESSAGES', payload: [] });
      dispatch({ type: 'SET_STREAMING_MESSAGE', payload: '' });
      dispatch({ type: 'SET_ATTACHMENTS', payload: [] });
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
    dispatch({ type: 'SET_IS_OPEN', payload: false });
    onOpenChange?.(false);
  };

  const handleOpen = () => {
    dispatch({ type: 'SET_IS_OPEN', payload: true });
    onOpenChange?.(true);
  };

  // Helper function to handle tool decisions (approval or denial)
  const handleToolDecision = useCallback(async (
    requestId: string, 
    decision: 'approve' | 'deny', 
    approvedIds?: string[]
  ) => {
    console.log(`Tool ${decision}:`, requestId, approvedIds);
    
    // Mark this approval request as processed to hide buttons
    dispatch({ type: 'UPDATE_MESSAGES', payload: (messages) => 
      messages.map(msg => 
        msg.approval_request?.request_id === requestId 
          ? { ...msg, approval_request: { ...msg.approval_request, processed: true } }
          : msg
      )
    });
    
    // Send decision as a special chat message
    const decisionMessage = decision === 'approve' 
      ? `[APPROVE_TOOLS:${requestId}]` 
      : `[DENY_TOOLS:${requestId}]`;
    
    dispatch({ type: 'SET_IS_LOADING', payload: true });
    dispatch({ type: 'SET_STREAMING_MESSAGE', payload: '' });
    streamingContentRef.current = '';
    
    // Store tool executions to preserve parameters (only for approvals)
    let collectedToolExecutions: ToolExecution[] = [];

    // Use the same streaming chat endpoint
    const cleanup = await aiChatAPI.streamChatWithSeparateEndpoints(
      state.sessionId,
      decisionMessage,
      (streamMessage: StreamMessage) => {
        if (streamMessage.type === 'message') {
          const newContent = streamMessage.content || '';
          streamingContentRef.current += newContent;
          dispatch({ type: 'SET_STREAMING_MESSAGE', payload: streamingContentRef.current });
        } else if (streamMessage.type === 'tool_execution' && decision === 'approve') {
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
            dispatch({ type: 'SET_STREAMING_MESSAGE', payload: streamingContentRef.current });
          }
        } else if (streamMessage.type === 'error') {
          const actionType = decision === 'approve' ? 'Approval' : 'Denial';
          console.error(`${actionType} stream error:`, streamMessage.error);
          
          // Clear any streaming content since we have an error
          dispatch({ type: 'SET_STREAMING_MESSAGE', payload: '' });
          streamingContentRef.current = '';
          
          // Create a proper error message to display in the chat
          const errorMessage: ChatMessage = {
            id: `${decision}_error_${Date.now()}`,
            role: 'assistant',
            content: `❌ **Tool ${actionType} Error**: ${streamMessage.error}`,
            timestamp: new Date().toISOString(),
          };
          
          dispatch({ type: 'ADD_MESSAGE', payload: errorMessage });
          dispatch({ type: 'SET_IS_LOADING', payload: false });
        }
      },
      (error: string) => {
        const actionType = decision === 'approve' ? 'Approval' : 'Denial';
        console.error(`${actionType} stream connection error:`, error);
        
        // Clear any streaming content since we have an error
        dispatch({ type: 'SET_STREAMING_MESSAGE', payload: '' });
        streamingContentRef.current = '';
        
        // Create a proper error message to display in the chat
        const errorMessage: ChatMessage = {
          id: `${decision}_connection_error_${Date.now()}`,
          role: 'assistant',
          content: `🔌 **Tool ${actionType} Connection Error**: ${error}`,
          timestamp: new Date().toISOString(),
        };
        
        dispatch({ type: 'ADD_MESSAGE', payload: errorMessage });
        dispatch({ type: 'SET_IS_LOADING', payload: false });
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
          dispatch({ type: 'ADD_MESSAGE', payload: assistantMessage });
        }
        
        // Clear any processed approval requests to remove "Processing request..." status
        dispatch({ type: 'UPDATE_MESSAGES', payload: (messages) => 
          messages.map(msg => 
            msg.approval_request?.processed 
              ? { ...msg, approval_request: undefined }
              : msg
          )
        });
        
        dispatch({ type: 'SET_STREAMING_MESSAGE', payload: '' });
        streamingContentRef.current = '';
        dispatch({ type: 'SET_IS_LOADING', payload: false });
      }
    );

    // Store cleanup function
    streamCleanupRef.current = cleanup;
  }, [state.sessionId]);

  const handleToolApproval = async (requestId: string, approvedIds?: string[]) => {
    await handleToolDecision(requestId, 'approve', approvedIds);
  };

  const handleToolDenial = async (requestId: string) => {
    await handleToolDecision(requestId, 'deny');
  };

  const positionStyles = {
    'bottom-right': { bottom: 16, right: 16 },
    'bottom-left': { bottom: 16, left: 16 },
  };

  // Floating action button to open chat
  if (!state.isOpen) {
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
              state.toolsLoading 
                ? "Loading MCP Tools..." 
                : state.availableTools.length > 0 
                  ? `View MCP Tools List (${state.availableTools.length} available)` 
                  : "No MCP Tools Available"
            } arrow>
              <IconButton
                onClick={() => dispatch({ type: 'SET_SHOW_TOOLS', payload: true })}
                size="medium"
                disabled={state.toolsLoading || state.availableTools.length === 0}
                sx={{ 
                  backgroundColor: !state.toolsLoading && state.availableTools.length > 0 
                    ? 'rgba(255, 255, 255, 0.15)' 
                    : 'rgba(255, 255, 255, 0.05)',
                  '&:hover': !state.toolsLoading && state.availableTools.length > 0 ? { 
                    backgroundColor: 'rgba(255, 255, 255, 0.25)',
                    transform: 'scale(1.05)',
                  } : {},
                  borderRadius: 2,
                  border: '1px solid rgba(255, 255, 255, 0.2)',
                  transition: 'all 0.2s ease-in-out',
                  opacity: !state.toolsLoading && state.availableTools.length > 0 ? 1 : 0.5,
                }}
              >
                {state.toolsLoading ? (
                  <CircularProgress size={20} sx={{ color: '#ffffff' }} />
                ) : (
                  <ListIcon sx={{ 
                    fontSize: 20, 
                    color: state.availableTools.length > 0 ? '#ffffff' : 'rgba(255, 255, 255, 0.5)', 
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
                onClick={() => dispatch({ type: 'SET_SHOW_SESSION_SELECTOR', payload: true })} 
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
      {(state.toolsLoading || state.availableTools.length > 0) && (
        <Box sx={{ 
          p: 1.5, 
          borderBottom: '1px solid #e0e0e0',
          backgroundColor: state.toolsLoading ? '#fff3e0' : '#f0f7ff',
          display: 'flex',
          alignItems: 'center',
          gap: 1,
        }}>
          {state.toolsLoading ? (
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
                {state.availableTools.length} MCP tool{state.availableTools.length !== 1 ? 's' : ''} available
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
        {state.messages.length === 0 && !state.streamingMessage && (
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
            {!state.sessionId && (
              <Typography variant="caption" color="textSecondary" sx={{ mt: 1, fontStyle: 'italic' }}>
                Your session will be created when you send your first message
              </Typography>
            )}
          </Box>
        )}

        <List sx={{ p: 0 }}>
          {state.messages.map((message) => (
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
        {state.streamingMessage && (
          <ListItem sx={{ p: 0, mb: 1 }}>
            <ChatMessageComponent
              message={{
                id: 'streaming',
                role: 'assistant',
                content: state.streamingMessage,
                timestamp: new Date().toISOString(),
              }}
              isStreaming
            />
          </ListItem>
        )}

        {/* Loading indicator */}
        {state.isLoading && !state.streamingMessage && (
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
          attachments={state.attachments}
          onAttachmentsChange={(attachments) => dispatch({ type: 'SET_ATTACHMENTS', payload: attachments })}
          triggerRef={fileUploadTriggerRef}
        />
        
        {/* Message input */}
        <Box sx={{ display: 'flex', gap: 1 }}>
          <FileUploadButton
            onClick={() => fileUploadTriggerRef.current?.()}
            disabled={state.attachments.length >= 5}
            hasAttachments={state.attachments.length > 0}
            maxFiles={5}
            maxFileSize={10 * 1024 * 1024}
          />
          <TextField
            fullWidth
            multiline
            maxRows={3}
            placeholder="Type your message..."
            value={state.inputMessage}
            onChange={(e) => dispatch({ type: 'SET_INPUT_MESSAGE', payload: e.target.value })}
            onKeyPress={handleKeyPress}
            disabled={state.isLoading}
            variant="outlined"
            size="small"
          />
          <IconButton
            color="primary"
            onClick={handleSendMessage}
            disabled={(!state.inputMessage.trim() && state.attachments.length === 0) || state.isLoading}
            sx={{ alignSelf: 'flex-end' }}
          >
            <SendIcon />
          </IconButton>
        </Box>
      </Box>

      {/* MCP Tools Dialog */}
      <MCPToolsDialog
        open={state.showTools}
        onClose={() => dispatch({ type: 'SET_SHOW_TOOLS', payload: false })}
        tools={state.availableTools}
      />

      {/* Session Selector Dialog */}
      <SessionSelector
        open={state.showSessionSelector}
        onClose={() => dispatch({ type: 'SET_SHOW_SESSION_SELECTOR', payload: false })}
        onSessionSelect={handleSessionSelect}
        currentSessionId={state.sessionId}
      />

    </Paper>
  );
}); 