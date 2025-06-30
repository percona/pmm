import React, { useState, useEffect } from 'react';
import {
  Box,
  Button,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  List,
  ListItem,
  ListItemText,
  ListItemSecondaryAction,
  IconButton,
  Typography,
  TextField,
  Chip,
  Divider,
  CircularProgress,
  Alert,
} from '@mui/material';
import {
  Add as AddIcon,
  Delete as DeleteIcon,
  Edit as EditIcon,
  Chat as ChatIcon,
  History as HistoryIcon,
} from '@mui/icons-material';
import { aiChatAPI, ChatSession } from '../../api/aichat';

interface SessionSelectorProps {
  open: boolean;
  onClose: () => void;
  onSessionSelect: (sessionId: string) => void;
  currentSessionId?: string;
}

export const SessionSelector: React.FC<SessionSelectorProps> = ({
  open,
  onClose,
  onSessionSelect,
  currentSessionId,
}) => {
  const [sessions, setSessions] = useState<ChatSession[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [editingSession, setEditingSession] = useState<ChatSession | null>(null);
  const [editTitle, setEditTitle] = useState('');
  const [newSessionTitle, setNewSessionTitle] = useState('');
  const [showNewSessionForm, setShowNewSessionForm] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [sessionIdToDelete, setSessionIdToDelete] = useState<string | null>(null);

  useEffect(() => {
    if (open) {
      loadSessions();
    }
  }, [open]);

  const loadSessions = async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await aiChatAPI.listSessions(1, 50); // Load first 50 sessions
      setSessions(response.sessions);
      console.log('📋 Loaded sessions:', response.sessions.length, 'current:', currentSessionId);
    } catch (error) {
      console.error('Failed to load sessions:', error);
      setError('Failed to load chat sessions');
    } finally {
      setLoading(false);
    }
  };

  const handleCreateSession = async () => {
    if (!newSessionTitle.trim()) return;

    try {
      const newSession = await aiChatAPI.createSession({ title: newSessionTitle.trim() });
      setSessions(prev => [newSession, ...prev]);
      setNewSessionTitle('');
      setShowNewSessionForm(false);
      onSessionSelect(newSession.id);
      onClose();
    } catch (error) {
      console.error('Failed to create session:', error);
      setError('Failed to create new session');
    }
  };

  const handleEditSession = async (session: ChatSession) => {
    if (!editTitle.trim()) return;

    try {
      await aiChatAPI.updateSession(session.id, { title: editTitle.trim() });
      setSessions(prev => 
        prev.map(s => s.id === session.id ? { ...s, title: editTitle.trim() } : s)
      );
      setEditingSession(null);
      setEditTitle('');
    } catch (error) {
      console.error('Failed to update session:', error);
      setError('Failed to update session');
    }
  };

  const handleDeleteSession = async (sessionId: string) => {
    try {
      await aiChatAPI.deleteSession(sessionId);
      setSessions(prev => prev.filter(s => s.id !== sessionId));
    } catch (error) {
      console.error('Failed to delete session:', error);
      setError('Failed to delete session');
    }
  };

  const handleSessionSelect = (sessionId: string) => {
    onSessionSelect(sessionId);
    onClose();
  };

  const formatDate = (dateString: string) => {
    if (!dateString) {
      return 'Unknown';
    }
    
    const date = new Date(dateString);
    
    // Check if date is valid
    if (isNaN(date.getTime())) {
      console.warn('Invalid date string:', dateString);
      return 'Invalid Date';
    }
    
    const now = new Date();
    
    // Reset time to midnight for accurate day comparison
    const dateOnly = new Date(date.getFullYear(), date.getMonth(), date.getDate());
    const nowOnly = new Date(now.getFullYear(), now.getMonth(), now.getDate());
    
    const diffTime = nowOnly.getTime() - dateOnly.getTime();
    const diffDays = Math.floor(diffTime / (1000 * 60 * 60 * 24));

    if (diffDays === 0) {
      // Same day - show time
      return `Today at ${date.toLocaleTimeString('en-US', { 
        hour: '2-digit', 
        minute: '2-digit',
        hour12: true 
      })}`;
    } else if (diffDays === 1) {
      return `Yesterday at ${date.toLocaleTimeString('en-US', { 
        hour: '2-digit', 
        minute: '2-digit',
        hour12: true 
      })}`;
    } else if (diffDays <= 7) {
      return `${diffDays} days ago`;
    } else if (diffDays <= 30) {
      return `${Math.floor(diffDays / 7)} weeks ago`;
    } else {
      // For older dates, show the actual date
      return date.toLocaleDateString('en-US', { 
        year: 'numeric', 
        month: 'short', 
        day: 'numeric' 
      });
    }
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>
        <Box display="flex" alignItems="center" gap={1}>
          <HistoryIcon />
          <Typography variant="h6">Chat Sessions</Typography>
        </Box>
      </DialogTitle>
      
      <DialogContent>
        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}

        {/* New Session Form */}
        <Box sx={{ mb: 2 }}>
          {!showNewSessionForm ? (
            <Button
              variant="text"
              startIcon={<AddIcon />}
              onClick={() => setShowNewSessionForm(true)}
              fullWidth
              sx={{ 
                color: 'text.secondary',
                '&:hover': { backgroundColor: 'action.hover' },
                textTransform: 'none',
                justifyContent: 'flex-start',
                fontSize: '0.875rem'
              }}
            >
              Create New Session (or just start typing a message)
            </Button>
          ) : (
            <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
              <TextField
                size="small"
                placeholder="Enter session title..."
                value={newSessionTitle}
                onChange={(e) => setNewSessionTitle(e.target.value)}
                onKeyPress={(e) => e.key === 'Enter' && handleCreateSession()}
                sx={{ flexGrow: 1 }}
              />
              <Button
                variant="contained"
                size="small"
                onClick={handleCreateSession}
                disabled={!newSessionTitle.trim()}
              >
                Create
              </Button>
              <Button
                variant="outlined"
                size="small"
                onClick={() => {
                  setShowNewSessionForm(false);
                  setNewSessionTitle('');
                }}
              >
                Cancel
              </Button>
            </Box>
          )}
        </Box>

        <Divider sx={{ mb: 2 }} />

        {/* Sessions List */}
        {loading ? (
          <Box display="flex" justifyContent="center" p={3}>
            <CircularProgress />
          </Box>
        ) : sessions.length === 0 ? (
          <Box textAlign="center" p={3}>
            <ChatIcon sx={{ fontSize: 48, color: 'text.secondary', mb: 1 }} />
            <Typography color="textSecondary" variant="body1" gutterBottom>
              No chat sessions yet
            </Typography>
            <Typography color="textSecondary" variant="body2">
              Sessions are created automatically when you send your first message
            </Typography>
          </Box>
        ) : (
          <List>
            {sessions.map((session) => (
              <ListItem
                key={session.id}
                button
                onClick={() => handleSessionSelect(session.id)}
                sx={{
                  border: 1,
                  borderColor: session.id === currentSessionId ? 'primary.main' : 'divider',
                  borderRadius: 1,
                  mb: 1,
                  backgroundColor: session.id === currentSessionId ? 'action.selected' : 'transparent',
                }}
              >
                <ListItemText
                  primary={
                    editingSession?.id === session.id ? (
                      <TextField
                        size="small"
                        value={editTitle}
                        onChange={(e) => setEditTitle(e.target.value)}
                        onKeyPress={(e) => {
                          if (e.key === 'Enter') {
                            handleEditSession(session);
                          } else if (e.key === 'Escape') {
                            setEditingSession(null);
                            setEditTitle('');
                          }
                        }}
                        onBlur={() => handleEditSession(session)}
                        autoFocus
                        onClick={(e) => e.stopPropagation()}
                      />
                    ) : (
                      <Box display="flex" alignItems="center" gap={1}>
                        <Typography variant="subtitle2">{session.title}</Typography>
                        {session.id === currentSessionId && (
                          <Chip label="Current" size="small" color="primary" />
                        )}
                      </Box>
                    )
                  }
                  secondary={`Created ${formatDate(session.created_at)} • Updated ${formatDate(session.updated_at)}`}
                />
                <ListItemSecondaryAction>
                  <IconButton
                    size="small"
                    onClick={(e) => {
                      e.stopPropagation();
                      setEditingSession(session);
                      setEditTitle(session.title);
                    }}
                    disabled={editingSession?.id === session.id}
                  >
                    <EditIcon />
                  </IconButton>
                  <IconButton
                    size="small"
                    onClick={(e) => {
                      e.stopPropagation();
                      setSessionIdToDelete(session.id);
                      setDeleteDialogOpen(true);
                    }}
                    disabled={session.id === currentSessionId}
                  >
                    <DeleteIcon />
                  </IconButton>
                </ListItemSecondaryAction>
              </ListItem>
            ))}
          </List>
        )}
      </DialogContent>

      <DialogActions>
        <Button onClick={onClose}>Close</Button>
      </DialogActions>

      <Dialog open={deleteDialogOpen} onClose={() => setDeleteDialogOpen(false)}>
        <DialogTitle>Delete Session</DialogTitle>
        <DialogContent>
          <Typography>Are you sure you want to delete this session?</Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteDialogOpen(false)} color="primary">
            Cancel
          </Button>
          <Button
            onClick={() => {
              if (sessionIdToDelete) {
                handleDeleteSession(sessionIdToDelete);
              }
              setDeleteDialogOpen(false);
              setSessionIdToDelete(null);
            }}
            color="error"
            variant="contained"
          >
            Delete
          </Button>
        </DialogActions>
      </Dialog>
    </Dialog>
  );
}; 