import Add from '@mui/icons-material/Add';
import DeleteOutline from '@mui/icons-material/DeleteOutline';
import Search from '@mui/icons-material/Search';
import {
  Box,
  Button,
  CircularProgress,
  Divider,
  IconButton,
  InputAdornment,
  List,
  ListItem,
  ListItemButton,
  ListItemText,
  Stack,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material';
import { type AdreConversation, type AdreSearchHit } from 'api/adre';
import { formatTimestamp } from 'hooks/useAdreChat';
import { useDebouncedAdreMessageSearch } from 'hooks/useDebouncedAdreMessageSearch';
import { formatAdreSearchSnippet } from 'utils/adreSearchSnippet';
import { FC, useCallback } from 'react';

export interface AdreConversationsSidebarProps {
  conversationId: number | null;
  conversations: AdreConversation[];
  loading: boolean;
  searchHits: AdreSearchHit[];
  searchLoading: boolean;
  onNewChat: () => void | Promise<void>;
  onDeleteConversation: (id: number) => void | Promise<void>;
  onSelectConversation: (id: number, options?: { focusMessageId?: number }) => void | Promise<void>;
  onSearch: (q: string) => void | Promise<void>;
}

export const AdreConversationsSidebar: FC<AdreConversationsSidebarProps> = ({
  conversationId,
  conversations,
  loading,
  searchHits,
  searchLoading,
  onNewChat,
  onDeleteConversation,
  onSelectConversation,
  onSearch,
}) => {
  const { query, setQuery, clearQuery, searchPending } = useDebouncedAdreMessageSearch(searchLoading, onSearch);

  const q = query.trim();

  const onHitClick = useCallback(
    (hit: AdreSearchHit) => {
      void onSelectConversation(hit.conversationId, { focusMessageId: hit.messageId });
      clearQuery();
    },
    [onSelectConversation, clearQuery]
  );

  return (
    <Stack
      sx={{
        height: '100%',
        minHeight: 0,
        borderRight: { md: 1 },
        borderColor: 'divider',
        bgcolor: '#1a1a1a',
      }}
    >
      <Stack direction="row" alignItems="center" justifyContent="space-between" sx={{ px: 1, py: 1, gap: 0.5 }}>
        <Typography variant="subtitle2" color="text.secondary" sx={{ pl: 0.5 }}>
          Chats
        </Typography>
        <IconButton size="small" aria-label="New chat" onClick={() => void onNewChat()} color="primary">
          <Add fontSize="small" />
        </IconButton>
      </Stack>
      <Box sx={{ px: 1, pb: 1 }}>
        <TextField
          size="small"
          fullWidth
          placeholder="Search messages…"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <Search fontSize="small" sx={{ color: 'text.secondary' }} />
              </InputAdornment>
            ),
          }}
          sx={{
            '& .MuiOutlinedInput-root': {
              bgcolor: '#252525',
              '& fieldset': { borderColor: 'rgba(255,255,255,0.12)' },
            },
          }}
        />
      </Box>
      <Divider sx={{ borderColor: 'rgba(255,255,255,0.08)' }} />
      <Box sx={{ flex: 1, minHeight: 0, overflow: 'auto' }}>
        {loading && conversations.length === 0 ? (
          <Box sx={{ display: 'flex', justifyContent: 'center', py: 3 }}>
            <CircularProgress size={22} />
          </Box>
        ) : !q ? (
          conversations.length === 0 ? (
            <Typography variant="body2" color="text.secondary" sx={{ px: 2, py: 2 }}>
              No conversations yet. Start typing below.
            </Typography>
          ) : (
            <List dense disablePadding>
              {conversations.map((c) => {
                const label = c.title?.trim() || `Chat ${c.id}`;
                const sub =
                  c.lastMessageAt || c.updatedAt
                    ? formatTimestamp(new Date(c.lastMessageAt || c.updatedAt).getTime())
                    : '';
                return (
                  <ListItem
                    key={c.id}
                    disablePadding
                    secondaryAction={
                      <Tooltip title="Delete conversation">
                        <IconButton
                          edge="end"
                          size="small"
                          aria-label={`Delete conversation ${label}`}
                          onClick={(e) => {
                            e.stopPropagation();
                            void onDeleteConversation(c.id);
                          }}
                          sx={{ color: 'text.secondary', mr: 0.5 }}
                        >
                          <DeleteOutline fontSize="small" />
                        </IconButton>
                      </Tooltip>
                    }
                  >
                    <ListItemButton
                      selected={c.id === conversationId}
                      onClick={() => void onSelectConversation(c.id)}
                      sx={{ pr: 6 }}
                    >
                      <ListItemText
                        primary={label}
                        secondary={sub}
                        primaryTypographyProps={{ noWrap: true, variant: 'body2' }}
                        secondaryTypographyProps={{ variant: 'caption' }}
                      />
                    </ListItemButton>
                  </ListItem>
                );
              })}
            </List>
          )
        ) : searchPending ? (
          <Box sx={{ display: 'flex', justifyContent: 'center', py: 3 }}>
            <CircularProgress size={22} />
          </Box>
        ) : searchHits.length > 0 ? (
          <>
            <Typography variant="caption" color="text.secondary" sx={{ px: 1.5, py: 0.75, display: 'block' }}>
              Results
            </Typography>
            <List dense disablePadding>
              {searchHits.map((hit) => (
                <ListItemButton
                  key={`${hit.conversationId}-${hit.messageId}`}
                  selected={hit.conversationId === conversationId}
                  onClick={() => onHitClick(hit)}
                  sx={{ alignItems: 'flex-start', py: 1 }}
                >
                  <ListItemText
                    primaryTypographyProps={{ variant: 'body2', sx: { wordBreak: 'break-word' } }}
                    secondaryTypographyProps={{ variant: 'caption' }}
                    primary={formatAdreSearchSnippet(hit.snippet)}
                    secondary={
                      hit.createdAt
                        ? `Conv #${hit.conversationId} · ${formatTimestamp(new Date(hit.createdAt).getTime())}`
                        : `Conv #${hit.conversationId}`
                    }
                  />
                </ListItemButton>
              ))}
            </List>
          </>
        ) : (
          <Typography variant="body2" color="text.secondary" sx={{ px: 2, py: 2 }}>
            No matching messages.
          </Typography>
        )}
      </Box>
      <Box sx={{ p: 1, borderTop: 1, borderColor: 'rgba(255,255,255,0.08)' }}>
        <Button fullWidth size="small" variant="outlined" startIcon={<Add />} onClick={() => void onNewChat()}>
          New chat
        </Button>
      </Box>
    </Stack>
  );
};
