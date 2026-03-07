import {
  Box,
  Button,
  Card,
  CardContent,
  FormControl,
  InputLabel,
  MenuItem,
  Select,
  Stack,
  TextField,
  Typography,
} from '@mui/material';
import { FC, useState, useCallback, useEffect } from 'react';
import { useAdreModels } from 'hooks/api/useAdre';
import { adreChatStream } from 'api/adre';
import { useSnackbar } from 'notistack';

const STORAGE_KEY = 'pmm-adre-chat';

function loadFromStorage(): { response: string; history: unknown[] } {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw) {
      const parsed = JSON.parse(raw) as { response?: string; history?: unknown[] };
      return {
        response: typeof parsed.response === 'string' ? parsed.response : '',
        history: Array.isArray(parsed.history) ? parsed.history : [],
      };
    }
  } catch {
    // ignore
  }
  return { response: '', history: [] };
}

function saveToStorage(response: string, history: unknown[]) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ response, history }));
  } catch {
    // ignore
  }
}

export const AdreChatPanel: FC = () => {
  const { data: models = [] } = useAdreModels();
  const { enqueueSnackbar } = useSnackbar();
  const [ask, setAsk] = useState('');
  const [model, setModel] = useState('');
  const [response, setResponse] = useState(() => loadFromStorage().response);
  const [loading, setLoading] = useState(false);
  const [history, setHistory] = useState<unknown[]>(() => loadFromStorage().history);

  useEffect(() => {
    saveToStorage(response, history);
  }, [response, history]);

  const handleSend = useCallback(async () => {
    if (!ask.trim()) return;
    const userAsk = ask.trim();
    setLoading(true);
    setResponse('');
    setAsk('');
    try {
      const req = {
        ask: userAsk,
        conversationHistory: history,
        model: model || undefined,
        stream: true,
      };
      let fullResponse = '';
      await adreChatStream(req, (chunk) => {
        fullResponse += chunk;
        setResponse(fullResponse);
      });
      setHistory((prev) => [
        ...prev,
        { role: 'user', content: userAsk },
        { role: 'assistant', content: fullResponse },
      ]);
    } catch (err) {
      enqueueSnackbar(
        err instanceof Error ? err.message : 'Chat request failed',
        { variant: 'error' }
      );
    } finally {
      setLoading(false);
    }
  }, [ask, history, model, enqueueSnackbar]);

  return (
    <Card variant="outlined" sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <CardContent sx={{ flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0 }}>
        <Typography variant="h6" gutterBottom>
          Chat
        </Typography>
        <Stack gap={1} sx={{ flex: 1, minHeight: 0 }}>
          <Box
            sx={{
              flex: 1,
              minHeight: 120,
              maxHeight: 300,
              overflow: 'auto',
              p: 1,
              bgcolor: 'action.hover',
              borderRadius: 1,
            }}
          >
            {response ? (
              <Typography component="pre" sx={{ whiteSpace: 'pre-wrap', fontFamily: 'inherit' }}>
                {response}
              </Typography>
            ) : (
              <Typography color="text.secondary" variant="body2">
                Ask a question about your database environment...
              </Typography>
            )}
          </Box>
          <Stack direction="row" gap={1} alignItems="center">
            <FormControl size="small" sx={{ minWidth: 140 }}>
              <InputLabel>Model</InputLabel>
              <Select
                value={model}
                label="Model"
                onChange={(e) => setModel(e.target.value)}
              >
                <MenuItem value="">Default</MenuItem>
                {models.map((m) => (
                  <MenuItem key={m} value={m}>
                    {m}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
          </Stack>
          <Stack direction="row" gap={1}>
            <TextField
              size="small"
              placeholder="Ask something..."
              value={ask}
              onChange={(e) => setAsk(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && !e.shiftKey && handleSend()}
              fullWidth
              multiline
              minRows={2}
              maxRows={6}
            />
            <Button
              variant="contained"
              onClick={handleSend}
              disabled={loading || !ask.trim()}
            >
              Send
            </Button>
          </Stack>
        </Stack>
      </CardContent>
    </Card>
  );
};
