import Add from '@mui/icons-material/Add';
import DeleteOutline from '@mui/icons-material/DeleteOutline';
import KeyboardDoubleArrowLeft from '@mui/icons-material/KeyboardDoubleArrowLeft';
import Search from '@mui/icons-material/Search';
import {
  Box,
  Button,
  CircularProgress,
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
  onSelectConversation: (
    id: number,
    options?: { focusMessageId?: number }
  ) => void | Promise<void>;
  onSearch: (q: string) => void | Promise<void>;
  /** Folds the panel to the side; without it the panel has no collapse button. */
  onCollapse?: () => void;
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
  onCollapse,
}) => {
  const { query, setQuery, clearQuery, searchPending } =
    useDebouncedAdreMessageSearch(searchLoading, onSearch);

  const q = query.trim();

  const onHitClick = useCallback(
    (hit: AdreSearchHit) => {
      void onSelectConversation(hit.conversationId, {
        focusMessageId: hit.messageId,
      });
      clearQuery();
    },
    [onSelectConversation, clearQuery]
  );

  return (
    <Stack
      component="nav"
      aria-label="Chats"
      gap={1.5}
      sx={{
        height: '100%',
        minHeight: 0,
        px: 1.5,
        py: 2.25,
        boxSizing: 'border-box',
        bgcolor: '#17161A',
        borderRight: { md: 1 },
        borderColor: 'rgba(255,255,255,0.06) !important',
      }}
    >
      <Stack direction="row" gap={0.75} sx={{ flexShrink: 0 }}>
        <Button
          variant="contained"
          disableElevation
          startIcon={<Add />}
          onClick={() => void onNewChat()}
          sx={{ flex: 1, height: 40 }}
        >
          New chat
        </Button>
        {onCollapse && (
          <Tooltip title="Collapse chats">
            <IconButton
              aria-label="Collapse chats"
              onClick={onCollapse}
              sx={{ color: '#A9A7B0' }}
            >
              <KeyboardDoubleArrowLeft fontSize="small" />
            </IconButton>
          </Tooltip>
        )}
      </Stack>
      <TextField
        size="small"
        fullWidth
        placeholder="Search messages"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        inputProps={{ 'aria-label': 'Search messages' }}
        InputProps={{
          startAdornment: (
            <InputAdornment position="start">
              <Search fontSize="small" sx={{ color: '#8E8C95' }} />
            </InputAdornment>
          ),
        }}
        sx={{
          flexShrink: 0,
          '& .MuiOutlinedInput-root': {
            borderRadius: '10px',
            bgcolor: 'transparent',
            '& fieldset': { borderColor: 'rgba(255,255,255,0.08)' },
          },
        }}
      />
      <Box sx={{ flex: 1, minHeight: 0, overflow: 'auto', mx: -0.5, px: 0.5 }}>
        {loading && conversations.length === 0 ? (
          <Box sx={{ display: 'flex', justifyContent: 'center', py: 3 }}>
            <CircularProgress size={22} />
          </Box>
        ) : !q ? (
          conversations.length === 0 ? (
            <Typography
              variant="body2"
              color="text.secondary"
              sx={{ px: 2, py: 2 }}
            >
              No conversations yet. Start typing below.
            </Typography>
          ) : (
            <List
              disablePadding
              sx={{ display: 'flex', flexDirection: 'column', gap: 0.25 }}
            >
              {conversations.map((c) => {
                const label = c.title?.trim() || `Chat ${c.id}`;
                const sub =
                  c.lastMessageAt || c.updatedAt
                    ? formatTimestamp(
                        new Date(c.lastMessageAt || c.updatedAt).getTime()
                      )
                    : '';
                const selected = c.id === conversationId;
                return (
                  <ListItem
                    key={c.id}
                    disablePadding
                    sx={{
                      '& .adre-chat-delete': { opacity: selected ? 1 : 0 },
                      '&:hover .adre-chat-delete, &:focus-within .adre-chat-delete':
                        { opacity: 1 },
                    }}
                    secondaryAction={
                      <Tooltip title="Delete conversation">
                        <IconButton
                          className="adre-chat-delete"
                          edge="end"
                          size="small"
                          aria-label={`Delete conversation ${label}`}
                          onClick={(e) => {
                            e.stopPropagation();
                            void onDeleteConversation(c.id);
                          }}
                          sx={{ color: '#A9A7B0', mr: 0.25 }}
                        >
                          <DeleteOutline fontSize="small" />
                        </IconButton>
                      </Tooltip>
                    }
                  >
                    <ListItemButton
                      selected={selected}
                      onClick={() => void onSelectConversation(c.id)}
                      sx={{
                        pr: 5,
                        py: 1,
                        borderRadius: '10px',
                        '&.Mui-selected, &.Mui-selected:hover': {
                          bgcolor: 'rgba(182,178,255,0.12)',
                        },
                      }}
                    >
                      <ListItemText
                        primary={label}
                        secondary={sub}
                        sx={{ my: 0 }}
                        primaryTypographyProps={{
                          noWrap: true,
                          sx: {
                            fontSize: 14,
                            fontWeight: selected ? 500 : 400,
                            color: selected ? '#F4F3F6' : '#D9D7DE',
                          },
                        }}
                        secondaryTypographyProps={{
                          sx: {
                            fontSize: 12,
                            color: selected ? '#B9B5E8' : '#8E8C95',
                          },
                        }}
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
            <Typography
              variant="caption"
              color="text.secondary"
              sx={{ px: 1.5, py: 0.75, display: 'block' }}
            >
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
                    primaryTypographyProps={{
                      variant: 'body2',
                      sx: { wordBreak: 'break-word' },
                    }}
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
          <Typography
            variant="body2"
            color="text.secondary"
            sx={{ px: 2, py: 2 }}
          >
            No matching messages.
          </Typography>
        )}
      </Box>
    </Stack>
  );
};
