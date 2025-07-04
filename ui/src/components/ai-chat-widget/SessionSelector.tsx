import React, { useState, useEffect, useRef } from 'react';
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
  const [page, setPage] = useState(1);
  const [hasMore, setHasMore] = useState(true);
  const loadingRef = useRef(false); // Ref to track loading state reliably in scroll handler

  useEffect(() => {
    if (open) {
      setSessions([]); // Clear sessions when opening the dialog
      setPage(1); // Reset page to 1
      setHasMore(true); // Assume there are more sessions to load initially
      loadSessions(1, true); // Load first page when dialog opens
    }
  }, [open]);

  const loadSessions = async (pageNumber: number, reset: boolean = false) => {
    if (loadingRef.current) return; // Prevent multiple simultaneous loads
    
    loadingRef.current = true;
    setLoading(true);
    setError(null);
    try {
      const pageSize = 20; // Define page size
      const response = await aiChatAPI.listSessions(pageNumber, pageSize);
      
      if (reset) {
        setSessions(response.sessions);
      } else {
        setSessions(prev => [...prev, ...response.sessions]);
      }

      setHasMore(response.sessions.length === pageSize); // If fewer than pageSize, no more sessions
      setPage(pageNumber);
    } catch (error) {
      console.error('Failed to load sessions:', error);
      setError('Failed to load chat sessions');
    } finally {
      setLoading(false);
      loadingRef.current = false;
    }
  };

  const handleLoadMore = () => {
    if (!loading && hasMore) {
      loadSessions(page + 1);
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
      
      <DialogContent dividers>
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
              New Session
            </Button>
          ) : (
            <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
              <TextField
                autoFocus
                margin="dense"
                label="New Session Title"
                type="text"
                fullWidth
                variant="outlined"
                value={newSessionTitle}
                onChange={(e) => setNewSessionTitle(e.target.value)}
                onKeyPress={(e) => {
                  if (e.key === 'Enter') {
                    handleCreateSession();
                  }
                }}
              />
              <Button onClick={handleCreateSession} variant="contained">Create</Button>
              <Button onClick={() => setShowNewSessionForm(false)} variant="outlined">Cancel</Button>
            </Box>
          )}
        </Box>

        <List dense>
          {sessions.length === 0 && !loading && !error && (
            <Typography variant="body2" color="text.secondary" sx={{ textAlign: 'center', py: 2 }}>
              No chat sessions found.
            </Typography>
          )}
          {sessions.map(session => (
            <ListItem
              key={session.id}
              button
              selected={session.id === currentSessionId}
              onClick={() => handleSessionSelect(session.id)}
              sx={{ borderRadius: 1, mb: 0.5 }}
            >
              <ListItemText 
                primary={session.title || 'Untitled Session'}
                secondary={formatDate(session.created_at)}
              />
              <ListItemSecondaryAction>
                <IconButton
                  edge="end"
                  aria-label="edit"
                  onClick={(e) => {
                    e.stopPropagation();
                    setEditingSession(session);
                    setEditTitle(session.title || '');
                  }}
                >
                  <EditIcon fontSize="small" />
                </IconButton>
                <IconButton
                  edge="end"
                  aria-label="delete"
                  onClick={(e) => {
                    e.stopPropagation();
                    setSessionIdToDelete(session.id);
                    setDeleteDialogOpen(true);
                  }}
                >
                  <DeleteIcon fontSize="small" />
                </IconButton>
              </ListItemSecondaryAction>
            </ListItem>
          ))}
          {loading && (
            <Box sx={{ display: 'flex', justifyContent: 'center', py: 2 }}>
              <CircularProgress size={24} />
            </Box>
          )}
          {!loading && hasMore && (
            <Button
              onClick={handleLoadMore}
              fullWidth
              variant="outlined"
              sx={{ mt: 2 }}
            >
              Load More
            </Button>
          )}
        </List>
      </DialogContent>

      <DialogActions>
        <Button onClick={onClose}>Close</Button>
      </DialogActions>

      {/* Edit Session Dialog */}
      <Dialog
        open={!!editingSession}
        onClose={() => setEditingSession(null)}
        aria-labelledby="form-dialog-title"
      >
        <DialogTitle id="form-dialog-title">Edit Session Title</DialogTitle>
        <DialogContent>
          <TextField
            autoFocus
            margin="dense"
            label="Session Title"
            type="text"
            fullWidth
            value={editTitle}
            onChange={(e) => setEditTitle(e.target.value)}
            onKeyPress={(e) => {
              if (e.key === 'Enter') {
                handleEditSession(editingSession!); // eslint-disable-line @typescript-eslint/no-non-null-assertion
              }
            }}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setEditingSession(null)} color="primary">
            Cancel
          </Button>
          <Button onClick={() => handleEditSession(editingSession!)} color="primary">
            Save
          </Button>
        </DialogActions>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog
        open={deleteDialogOpen}
        onClose={() => setDeleteDialogOpen(false)}
        aria-labelledby="alert-dialog-title"
        aria-describedby="alert-dialog-description"
      >
        <DialogTitle id="alert-dialog-title">{"Confirm Deletion"}</DialogTitle>
        <DialogContent>
          <Typography id="alert-dialog-description">
            Are you sure you want to delete this session? This action cannot be undone.
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteDialogOpen(false)} color="primary">
            Cancel
          </Button>
          <Button
            onClick={() => {
              if (sessionIdToDelete) {
                handleDeleteSession(sessionIdToDelete);
                setDeleteDialogOpen(false);
                setSessionIdToDelete(null);
              }
            }}
            color="primary" 
            autoFocus
          >
            Delete
          </Button>
        </DialogActions>
      </Dialog>
    </Dialog>
  );
}; 