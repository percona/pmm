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
import { FC, useState, useCallback } from 'react';
import { useAdreModels } from 'hooks/api/useAdre';
import { adreChat, adreChatStream } from 'api/adre';
import { useSnackbar } from 'notistack';

export const AdreChatPanel: FC = () => {
  const { data: models = [] } = useAdreModels();
  const { enqueueSnackbar } = useSnackbar();
  const [ask, setAsk] = useState('');
  const [model, setModel] = useState('');
  const [stream, setStream] = useState(true);
  const [response, setResponse] = useState('');
  const [loading, setLoading] = useState(false);
  const [history, setHistory] = useState<unknown[]>([]);

  const handleSend = useCallback(async () => {
    if (!ask.trim()) return;
    setLoading(true);
    setResponse('');
    try {
      const req = {
        ask: ask.trim(),
        conversationHistory: history,
        model: model || undefined,
        stream,
      };
      if (stream) {
        await adreChatStream(req, (chunk) => {
          setResponse((prev) => prev + chunk);
        });
      } else {
        const res = await adreChat(req);
        setResponse(res.analysis);
        if (res.conversationHistory?.length) {
          setHistory(res.conversationHistory as unknown[]);
        }
      }
    } catch (err) {
      enqueueSnackbar(
        err instanceof Error ? err.message : 'Chat request failed',
        { variant: 'error' }
      );
    } finally {
      setLoading(false);
    }
  }, [ask, history, model, stream, enqueueSnackbar]);

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
            <Button
              variant={stream ? 'contained' : 'outlined'}
              size="small"
              onClick={() => setStream(!stream)}
            >
              Stream
            </Button>
          </Stack>
          <Stack direction="row" gap={1}>
            <TextField
              size="small"
              placeholder="Ask something..."
              value={ask}
              onChange={(e) => setAsk(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && !e.shiftKey && handleSend()}
              fullWidth
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
