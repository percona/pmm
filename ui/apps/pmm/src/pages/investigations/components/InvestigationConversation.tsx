import ArrowUpward from '@mui/icons-material/ArrowUpward';
import {
  Box,
  Button,
  IconButton,
  InputBase,
  Stack,
  Typography,
} from '@mui/material';
import { FC, KeyboardEvent } from 'react';
import Markdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeRaw from 'rehype-raw';
import type {
  InvestigationComment,
  InvestigationMessage,
} from 'api/investigations';
import { getMarkdownComponents } from 'components/adre/adre-chat-markdown.helpers';
import { HolmesUsageFooter } from 'components/adre/HolmesUsageFooter';
import {
  adreColors,
  answerBodySx,
  headingFont,
} from 'components/adre/adreAnswerSx';

const sectionTitleSx = {
  fontFamily: headingFont,
  fontSize: 18,
  fontWeight: 600,
  color: adreColors.text,
  mb: 1.5,
} as const;

const metaSx = { fontSize: 12, color: adreColors.muted, mb: 0.5 } as const;

const composerSx = {
  p: 1.5,
  pl: 2,
  borderRadius: '16px',
  bgcolor: adreColors.card,
  border: `1px solid ${adreColors.border}`,
  '&:focus-within': { borderColor: 'rgba(182,178,255,0.45)' },
} as const;

const when = (iso: string) => new Date(iso).toLocaleString();

// A question reads as a heading; pasted logs and long messages stay body text.
const isShortQuestion = (text: string) =>
  text.length <= 200 && !text.includes('\n');

export interface InvestigationChatProps {
  messages: InvestigationMessage[];
  value: string;
  onChange: (value: string) => void;
  onSend: () => void;
  sending: boolean;
}

/** The follow-up chat about an investigation, laid out like the Assistant. */
export const InvestigationChat: FC<InvestigationChatProps> = ({
  messages,
  value,
  onChange,
  onSend,
  sending,
}) => {
  const canSend = !!value.trim() && !sending;
  const onKeyDown = (e: KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      if (canSend) onSend();
    }
  };
  // The API lists the newest first.
  const shown = [...messages]
    .filter((m) => m.role === 'user' || m.role === 'assistant')
    .reverse();

  return (
    <Box component="section" aria-label="Chat">
      <Typography component="h2" sx={sectionTitleSx}>
        Chat
      </Typography>
      <Stack gap={3}>
        {shown.length === 0 && (
          <Typography sx={{ fontSize: 14, color: adreColors.muted }}>
            Ask a follow-up question about this investigation.
          </Typography>
        )}
        {shown.map((m) =>
          m.role === 'user' ? (
            <Box key={m.id}>
              <Typography sx={metaSx}>You · {when(m.createdAt)}</Typography>
              <Typography
                sx={
                  isShortQuestion(m.content)
                    ? {
                        fontFamily: headingFont,
                        fontSize: 17,
                        fontWeight: 500,
                        color: adreColors.text,
                      }
                    : {
                        fontSize: 15,
                        color: adreColors.body,
                        whiteSpace: 'pre-wrap',
                        overflowWrap: 'anywhere',
                      }
                }
              >
                {m.content}
              </Typography>
            </Box>
          ) : (
            <Box key={m.id}>
              <Typography sx={metaSx}>
                Assistant · {when(m.createdAt)}
              </Typography>
              <Box sx={answerBodySx}>
                <Markdown
                  remarkPlugins={[remarkGfm]}
                  rehypePlugins={[rehypeRaw]}
                  components={getMarkdownComponents(m.content)}
                >
                  {m.content}
                </Markdown>
              </Box>
              <HolmesUsageFooter
                usage={{
                  model: m.model,
                  promptTokens: m.promptTokens ?? m.prompt_tokens,
                  completionTokens: m.completionTokens ?? m.completion_tokens,
                  totalTokens: m.totalTokens ?? m.total_tokens,
                  cachedTokens: m.cachedTokens ?? m.cached_tokens,
                  totalCost: m.totalCost ?? m.total_cost,
                }}
                align="left"
              />
            </Box>
          )
        )}
        <Box sx={composerSx}>
          <InputBase
            multiline
            minRows={2}
            maxRows={8}
            fullWidth
            placeholder="Ask about this investigation…"
            value={value}
            onChange={(e) => onChange(e.target.value)}
            onKeyDown={onKeyDown}
            inputProps={{ 'aria-label': 'Ask about this investigation' }}
            sx={{ fontSize: 15, lineHeight: 1.5, color: adreColors.text }}
          />
          <Stack direction="row" justifyContent="flex-end">
            <IconButton
              aria-label="Send"
              onClick={onSend}
              disabled={!canSend}
              sx={{
                width: 36,
                height: 36,
                bgcolor: adreColors.accent,
                color: adreColors.panel,
                '&:hover': { bgcolor: '#C9C6FF' },
                '&.Mui-disabled': {
                  bgcolor: 'rgba(255,255,255,0.08)',
                  color: adreColors.muted,
                },
              }}
            >
              <ArrowUpward fontSize="small" />
            </IconButton>
          </Stack>
        </Box>
      </Stack>
    </Box>
  );
};

export interface InvestigationCommentsProps {
  comments: InvestigationComment[];
  value: string;
  onChange: (value: string) => void;
  onAdd: () => void;
  adding: boolean;
}

/** Notes people leave on an investigation. */
export const InvestigationComments: FC<InvestigationCommentsProps> = ({
  comments,
  value,
  onChange,
  onAdd,
  adding,
}) => (
  <Box component="section" aria-label="Comments">
    <Typography component="h2" sx={sectionTitleSx}>
      Comments
    </Typography>
    <Stack gap={1.5}>
      {comments.map((c) => (
        <Box
          key={c.id}
          sx={{
            p: 1.75,
            borderRadius: '12px',
            bgcolor: adreColors.panel,
            border: `1px solid ${adreColors.divider}`,
          }}
        >
          <Typography sx={metaSx}>
            {c.author || 'Anonymous'} · {when(c.createdAt)}
          </Typography>
          <Typography
            sx={{
              fontSize: 14,
              color: adreColors.body,
              whiteSpace: 'pre-wrap',
              overflowWrap: 'anywhere',
            }}
          >
            {c.content}
          </Typography>
        </Box>
      ))}
      <Box sx={composerSx}>
        <InputBase
          multiline
          minRows={2}
          fullWidth
          placeholder="Add a comment…"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          onKeyDown={(e) => {
            if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
              e.preventDefault();
              onAdd();
            }
          }}
          inputProps={{ 'aria-label': 'Add a comment' }}
          sx={{ fontSize: 14, color: adreColors.text }}
        />
        <Stack direction="row" justifyContent="flex-end">
          <Button
            size="small"
            onClick={onAdd}
            disabled={!value.trim() || adding}
            sx={{ color: adreColors.accentText }}
          >
            Add comment
          </Button>
        </Stack>
      </Box>
    </Stack>
  </Box>
);
