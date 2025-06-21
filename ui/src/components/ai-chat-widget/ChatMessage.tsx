import React from 'react';
import {
  Box,
  Paper,
  Typography,
  Avatar,
  Chip,
  Fade,
  IconButton,
  Tooltip,
  Button,
  Divider,
} from '@mui/material';
import {
  Person as PersonIcon,
  SmartToy as BotIcon,
  Build as ToolIcon,
  Settings as SystemIcon,
  AttachFile as AttachFileIcon,
  Image as ImageIcon,
  Description as DocumentIcon,
  ContentCopy as CopyIcon,
  Warning as WarningIcon,
  Check as CheckIcon,
  Close as CloseIcon,
} from '@mui/icons-material';
import { ChatMessage } from '../../api/aichat';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';

interface ChatMessageProps {
  message: ChatMessage;
  isStreaming?: boolean;
  onToolApproval?: (requestId: string, approvedIds?: string[]) => void;
  onToolDenial?: (requestId: string) => void;
}

export const ChatMessageComponent: React.FC<ChatMessageProps> = ({
  message,
  isStreaming = false,
  onToolApproval,
  onToolDenial,
}) => {
  const isUser = message.role === 'user';
  const isAssistant = message.role === 'assistant';
  const isTool = message.role === 'tool';
  const isSystem = message.role === 'system';
  const isToolApproval = message.role === 'tool_approval';

  // Debug logging for attachments
  React.useEffect(() => {
    if (message.attachments && message.attachments.length > 0) {
      console.log('💾 ChatMessage: Message has attachments:', message.attachments);
    }
  }, [message.attachments]);

  // Copy to clipboard function
  const copyToClipboard = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text);
      // You could add a toast notification here if desired
      console.log('Code copied to clipboard');
    } catch (err) {
      console.error('Failed to copy code:', err);
    }
  };

  const getAvatarIcon = () => {
    if (isUser) return <PersonIcon />;
    if (isAssistant) return <BotIcon />;
    if (isTool) return <ToolIcon />;
    if (isToolApproval) return <WarningIcon />;
    return <SystemIcon />;
  };

  const getAvatarColor = () => {
    if (isUser) return 'primary';
    if (isAssistant) return 'secondary';
    if (isTool) return 'warning';
    if (isToolApproval) return 'warning';
    return 'default';
  };

  const getRoleName = () => {
    if (isUser) return 'You';
    if (isAssistant) return 'AI Assistant';
    if (isTool) return 'Tool Result';
    if (isToolApproval) return 'Tool Approval Request';
    return 'System';
  };

  return (
    <Fade in timeout={300}>
      <Box
        sx={{
          display: 'flex',
          flexDirection: isUser ? 'row-reverse' : 'row',
          alignItems: 'flex-start',
          gap: 1,
          width: '100%',
          mb: 1,
        }}
      >
        <Avatar
          sx={{
            bgcolor: `${getAvatarColor()}.main`,
            width: 32,
            height: 32,
          }}
        >
          {getAvatarIcon()}
        </Avatar>

        <Box
          sx={{
            maxWidth: '80%',
            minWidth: 0, // Allow shrinking below content size
            display: 'flex',
            flexDirection: 'column',
            alignItems: isUser ? 'flex-end' : 'flex-start',
          }}
        >
          {/* Role name and timestamp */}
          <Typography
            variant="caption"
            color="textSecondary"
            sx={{ mb: 0.5, px: 1 }}
          >
            {getRoleName()}
            {message.timestamp && (
              <span style={{ marginLeft: 8 }}>
                {new Date(message.timestamp).toLocaleTimeString([], {
                  hour: '2-digit',
                  minute: '2-digit',
                })}
              </span>
            )}
            {isStreaming && (
              <Chip
                label="Typing..."
                size="small"
                color="primary"
                variant="outlined"
                sx={{ ml: 1, height: 16, fontSize: '0.6rem' }}
              />
            )}
          </Typography>

          {/* Message content */}
          <Paper
            elevation={1}
            sx={{
              p: 1.5,
              backgroundColor: isUser
                ? 'primary.main'
                : isTool
                ? 'warning.light'
                : 'grey.100',
              color: isUser ? 'primary.contrastText' : 'text.primary',
              borderRadius: isUser ? '16px 16px 4px 16px' : '16px 16px 16px 4px',
              maxWidth: '100%',
              minWidth: 0, // Allow shrinking below content size
              wordBreak: 'break-word',
              overflowWrap: 'break-word',
              overflow: 'hidden', // Prevent content from overflowing
            }}
          >
            {isAssistant || isTool ? (
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
                                    code: ({ inline, children, ...props }: any) => {
                    // Since inline prop is unreliable, detect based on content characteristics
                    const isInlineCode = inline !== false && !String(children).includes('\n');
                    console.log('Code element:', { inline, isInlineCode, children: String(children).substring(0, 50) });
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
                        <Box
                          component="pre"
                          sx={{
                            backgroundColor: 'rgba(0, 0, 0, 0.1)',
                            padding: 2,
                            borderRadius: 1,
                            overflow: 'auto',
                            overflowWrap: 'break-word',
                            wordBreak: 'break-word',
                            fontSize: '0.8em',
                            lineHeight: 1.3,
                            display: 'block',
                            whiteSpace: 'pre-wrap',
                            maxWidth: '100%',
                            maxHeight: '300px', // Limit height for very long code blocks
                            border: '1px solid rgba(0, 0, 0, 0.1)',
                            // Custom scrollbar styling
                            '&::-webkit-scrollbar': {
                              width: '8px',
                              height: '8px',
                            },
                            '&::-webkit-scrollbar-track': {
                              backgroundColor: 'rgba(0, 0, 0, 0.05)',
                              borderRadius: '4px',
                            },
                            '&::-webkit-scrollbar-thumb': {
                              backgroundColor: 'rgba(0, 0, 0, 0.2)',
                              borderRadius: '4px',
                              '&:hover': {
                                backgroundColor: 'rgba(0, 0, 0, 0.3)',
                              },
                            },
                            // Firefox scrollbar styling
                            scrollbarWidth: 'thin',
                            scrollbarColor: 'rgba(0, 0, 0, 0.2) rgba(0, 0, 0, 0.05)',
                          }}
                        >
                          <Box
                            component="code"
                            sx={{
                              fontFamily: 'monospace',
                              fontSize: 'inherit',
                            }}
                          >
                            {children}
                          </Box>
                        </Box>
                        
                        {/* Copy button */}
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
                            <CopyIcon fontSize="small" />
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
                  blockquote: ({ children }) => (
                    <Box
                      sx={{
                        borderLeft: '4px solid',
                        borderLeftColor: isUser ? 'primary.contrastText' : 'primary.main',
                        pl: 2,
                        ml: 1,
                        mb: 1,
                        '&:last-child': { mb: 0 },
                      }}
                    >
                      {children}
                    </Box>
                  ),
                }}
              >
                {message.content}
              </ReactMarkdown>
            ) : (
              <Typography variant="body2" sx={{ whiteSpace: 'pre-wrap' }}>
                {message.content}
              </Typography>
            )}

            {/* Tool calls display */}
            {message.tool_calls && message.tool_calls.length > 0 && (
              <Box sx={{ mt: 1, pt: 1, borderTop: '1px solid rgba(0,0,0,0.1)' }}>
                <Typography variant="caption" color="textSecondary" sx={{ mb: 1 }}>
                  Tools used:
                </Typography>
                {message.tool_calls.map((toolCall, index) => (
                  <Chip
                    key={toolCall.id || index}
                    label={toolCall.function.name}
                    size="small"
                    icon={<ToolIcon />}
                    sx={{ mr: 0.5, mb: 0.5 }}
                  />
                ))}
              </Box>
            )}

            {/* Tool executions display */}
            {message.tool_executions && message.tool_executions.length > 0 && (
              <Box sx={{ mt: 1, pt: 1, borderTop: '1px solid rgba(0,0,0,0.1)' }}>
                <Typography variant="caption" color="textSecondary" sx={{ mb: 1 }}>
                  Tool executions:
                </Typography>
                {message.tool_executions.map((execution, index) => (
                  <Box key={execution.id || index} sx={{ mb: 1 }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 0.5 }}>
                      <Chip
                        label={`${execution.tool_name} (${execution.duration_ms}ms)`}
                        size="small"
                        icon={<ToolIcon />}
                        color={execution.error ? 'error' : 'success'}
                        sx={{ mr: 0.5 }}
                      />
                    </Box>
                    {execution.arguments && (
                      <Typography variant="caption" color="textSecondary" sx={{ display: 'block', mb: 0.5 }}>
                        Args: {execution.arguments}
                      </Typography>
                    )}
                    {execution.result && (
                      <Typography variant="caption" sx={{ display: 'block', fontFamily: 'monospace', fontSize: '0.7rem' }}>
                        {execution.error ? `Error: ${execution.error}` : `Result: ${execution.result.substring(0, 100)}${execution.result.length > 100 ? '...' : ''}`}
                      </Typography>
                    )}
                  </Box>
                ))}
              </Box>
            )}

            {/* Attachments display */}
            {message.attachments && message.attachments.length > 0 && (
              <Box sx={{ mt: 1, pt: 1, borderTop: '1px solid rgba(0,0,0,0.1)' }}>
                <Typography variant="caption" color="textSecondary" sx={{ mb: 1 }}>
                  Attachments:
                </Typography>
                {message.attachments.map((attachment, index) => {
                  const isImage = attachment.mime_type.startsWith('image/');
                  return (
                    <Box key={index} sx={{ mb: 1 }}>
                      <Chip
                        label={`${attachment.filename} (${Math.round(attachment.size / 1024)}KB)`}
                        size="small"
                        icon={isImage ? <ImageIcon /> : <DocumentIcon />}
                        sx={{ mr: 0.5, mb: 0.5 }}
                      />
                      {isImage && attachment.content && (
                        <Box
                          sx={{
                            mt: 1,
                            maxWidth: '200px',
                            borderRadius: 1,
                            overflow: 'hidden',
                            border: '1px solid rgba(0,0,0,0.1)',
                          }}
                        >
                          <img
                            src={`data:${attachment.mime_type};base64,${attachment.content}`}
                            alt={attachment.filename}
                            style={{
                              width: '100%',
                              height: 'auto',
                              display: 'block',
                            }}
                          />
                        </Box>
                      )}
                    </Box>
                  );
                })}
              </Box>
            )}

            {/* Tool approval request */}
            {isToolApproval && message.approval_request && (
              <Box sx={{ mt: 2, pt: 2, borderTop: '1px solid rgba(0,0,0,0.1)' }}>
                <Typography variant="body2" sx={{ mb: 2, fontWeight: 'bold' }}>
                  🔧 Tool Execution Request
                </Typography>
                <Typography variant="body2" color="textSecondary" sx={{ mb: 2 }}>
                  The AI assistant wants to execute the following tool(s):
                </Typography>
                
                {/* Tool list */}
                <Box sx={{ mb: 2 }}>
                  {message.approval_request.tool_calls.map((toolCall, index) => (
                    <Box key={toolCall.id || index} sx={{ mb: 1 }}>
                      <Paper 
                        variant="outlined" 
                        sx={{ 
                          p: 1.5, 
                          backgroundColor: 'rgba(255, 152, 0, 0.05)',
                          border: '1px solid rgba(255, 152, 0, 0.2)'
                        }}
                      >
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                          <ToolIcon color="warning" fontSize="small" />
                          <Typography variant="subtitle2" fontWeight="bold">
                            {toolCall.function.name}
                          </Typography>
                          <Chip 
                            label={toolCall.type} 
                            size="small" 
                            variant="outlined"
                            color="warning"
                          />
                        </Box>
                        
                        {toolCall.function.arguments && (
                          <Box>
                            <Typography variant="caption" color="textSecondary" sx={{ mb: 0.5, display: 'block' }}>
                              Arguments:
                            </Typography>
                            <Typography 
                              variant="caption" 
                              component="pre"
                              sx={{ 
                                fontFamily: 'monospace',
                                fontSize: '0.7rem',
                                backgroundColor: 'rgba(0, 0, 0, 0.05)',
                                p: 0.5,
                                borderRadius: 0.5,
                                display: 'block',
                                whiteSpace: 'pre-wrap',
                                wordBreak: 'break-word',
                                maxWidth: '100%',
                                overflow: 'auto'
                              }}
                            >
                              {(() => {
                                try {
                                  return JSON.stringify(JSON.parse(toolCall.function.arguments), null, 2);
                                } catch {
                                  return toolCall.function.arguments;
                                }
                              })()}
                            </Typography>
                          </Box>
                        )}
                      </Paper>
                    </Box>
                  ))}
                </Box>



                {/* Action buttons - only show if not processed */}
                {!message.approval_request.processed && (
                  <Box sx={{ display: 'flex', gap: 1, justifyContent: 'flex-end' }}>
                    <Button 
                      variant="outlined" 
                      color="error"
                      size="small"
                      startIcon={<CloseIcon />}
                      onClick={() => onToolDenial?.(message.approval_request!.request_id)}
                    >
                      Deny
                    </Button>
                    <Button 
                      variant="contained" 
                      color="warning"
                      size="small"
                      startIcon={<CheckIcon />}
                      onClick={() => onToolApproval?.(message.approval_request!.request_id)}
                    >
                      Approve All
                    </Button>
                  </Box>
                )}

                {/* Show processed status */}
                {message.approval_request.processed && (
                  <Box sx={{ 
                    display: 'flex', 
                    justifyContent: 'center', 
                    p: 1, 
                    backgroundColor: 'rgba(0, 0, 0, 0.05)', 
                    borderRadius: 1 
                  }}>
                    <Typography variant="caption" color="textSecondary">
                      ⏳ Processing request...
                    </Typography>
                  </Box>
                )}
              </Box>
            )}
          </Paper>
        </Box>
      </Box>
    </Fade>
  );
}; 