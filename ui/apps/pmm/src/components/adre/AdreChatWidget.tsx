import {
  Box,
  IconButton,
  Paper,
  Stack,
  TextField,
  Typography,
} from '@mui/material';
import ChatIcon from '@mui/icons-material/Chat';
import CloseIcon from '@mui/icons-material/Close';
import SendIcon from '@mui/icons-material/Send';
import { FC, useState, useCallback } from 'react';
import { useAdreSettings } from 'hooks/api/useAdre';
import { adreChatStream } from 'api/adre';
import { useSnackbar } from 'notistack';

export const AdreChatWidget: FC = () => {
  const { data: settings } = useAdreSettings();
  const { enqueueSnackbar } = useSnackbar();
  const [open, setOpen] = useState(false);
  const [ask, setAsk] = useState('');
  const [response, setResponse] = useState('');
  const [loading, setLoading] = useState(false);

  const isConfigured = settings?.enabled && !!settings?.url;

  const handleSend = useCallback(async () => {
    if (!ask.trim() || !isConfigured) return;
    setLoading(true);
    setResponse('');
    try {
      await adreChatStream(
        { ask: ask.trim(), stream: true },
        (contentChunk) => {
          if (contentChunk) setResponse((prev) => prev + contentChunk);
        }
      );
    } catch (err) {
      enqueueSnackbar(
        err instanceof Error ? err.message : 'Chat failed',
        { variant: 'error' }
      );
    } finally {
      setLoading(false);
    }
  }, [ask, isConfigured, enqueueSnackbar]);

  if (!isConfigured) return null;

  return (
    <>
      <IconButton
        onClick={() => setOpen((o) => !o)}
        sx={{
          position: 'fixed',
          bottom: 24,
          right: 24,
          bgcolor: 'primary.main',
          color: 'primary.contrastText',
          width: 56,
          height: 56,
          boxShadow: 2,
          '&:hover': { bgcolor: 'primary.dark' },
        }}
        aria-label="Open ADRE chat"
      >
        <ChatIcon />
      </IconButton>
      {open && (
        <Paper
          elevation={8}
          sx={{
            position: 'fixed',
            bottom: 90,
            right: 24,
            width: 360,
            maxWidth: 'calc(100vw - 48px)',
            height: 420,
            display: 'flex',
            flexDirection: 'column',
            zIndex: 1300,
          }}
        >
          <Stack
            direction="row"
            alignItems="center"
            justifyContent="space-between"
            sx={{ p: 1, borderBottom: 1, borderColor: 'divider' }}
          >
            <Typography variant="subtitle1">ADRE Chat</Typography>
            <IconButton size="small" onClick={() => setOpen(false)}>
              <CloseIcon />
            </IconButton>
          </Stack>
          <Box
            sx={{
              flex: 1,
              overflow: 'auto',
              p: 1,
              bgcolor: 'action.hover',
            }}
          >
            {response ? (
              <Typography component="pre" sx={{ whiteSpace: 'pre-wrap', fontSize: '0.875rem' }}>
                {response}
              </Typography>
            ) : (
              <Typography color="text.secondary" variant="body2">
                Ask a question...
              </Typography>
            )}
          </Box>
          <Stack direction="row" gap={0.5} sx={{ p: 1 }}>
            <TextField
              size="small"
              placeholder="Ask..."
              value={ask}
              onChange={(e) => setAsk(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && !e.shiftKey && handleSend()}
              fullWidth
            />
            <IconButton
              color="primary"
              onClick={handleSend}
              disabled={loading || !ask.trim()}
            >
              <SendIcon />
            </IconButton>
          </Stack>
        </Paper>
      )}
    </>
  );
};
